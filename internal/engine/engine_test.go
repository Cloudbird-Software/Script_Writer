package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Cloudbird-Software/Script_Writer/internal/state"
)

// e2e：examples/demo 是一份干净样例，全量 Run 必须零硬失败、P0 清账、五件套齐备。
func TestRunDemoClean(t *testing.T) {
	c, eps, err := LoadManifest(filepath.Join("..", "..", "examples", "demo", "manifest.yaml"))
	if err != nil {
		t.Fatalf("load demo manifest: %v", err)
	}
	res, err := Run(c, eps)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.HasError() {
		t.Fatalf("干净样例不应有硬失败：\n%s", state.FormatViolations(res.Violations))
	}
	if res.Deliverable.Blocked {
		t.Fatal("干净样例 P0 应已清账")
	}
	if len(res.Deliverable.Beats) != 5 || len(res.Deliverable.Loops) != 4 {
		t.Fatalf("五件套数量不符：beats=%d loops=%d", len(res.Deliverable.Beats), len(res.Deliverable.Loops))
	}
	for _, lp := range res.Deliverable.Loops {
		if lp.Status != "closed" {
			t.Fatalf("demo 全部伏笔应回收：%+v", lp)
		}
	}
	md := res.Deliverable.RenderMarkdown()
	for _, want := range []string{"人物表", "伏笔台账", "卖点覆盖表", "风险清单", "beat + 钩子表", "可交付"} {
		if !strings.Contains(md, want) {
			t.Fatalf("五件套报表缺 %q", want)
		}
	}
}

// 脏样例 e2e：黑名单名 + 凭空引文 + 暖收 + P0 未清账 → Blocked，且 Sweep 有逐句建议。
func TestRunDirtyBlocked(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join("..", "..", "examples", "demo")
	if err := os.CopyFS(dir, os.DirFS(src)); err != nil {
		t.Fatalf("copy demo: %v", err)
	}
	// e05 改脏：黑名单名 + 凭空引文；并把 hooks_closed 抹掉 → P0 未清账。
	dirty := "渔捕快站在房契旁，柳青眉想起那句『过客有期』，账没结成。"
	if err := os.WriteFile(filepath.Join(dir, "episodes", "e05.md"), []byte(dirty), 0o644); err != nil {
		t.Fatal(err)
	}
	mf, err := os.ReadFile(filepath.Join(dir, "manifest.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	mf = []byte(strings.Replace(string(mf), "hooks_closed: [L_MUPAI, L_GUANCHAI, L_DENG, L_FANGQI]", "hooks_closed: []", 1))
	if err := os.WriteFile(filepath.Join(dir, "manifest.yaml"), mf, 0o644); err != nil {
		t.Fatal(err)
	}

	c, eps, err := LoadManifest(filepath.Join(dir, "manifest.yaml"))
	if err != nil {
		t.Fatalf("load dirty manifest: %v", err)
	}
	res, err := Run(c, eps)
	if err != nil {
		t.Fatal(err)
	}
	if !res.HasError() || !res.Deliverable.Blocked {
		t.Fatalf("脏样例必须 Blocked：\n%s", state.FormatViolations(res.Violations))
	}
	gates := map[string]bool{}
	for _, v := range res.Violations {
		gates[v.Gate] = true
	}
	for _, want := range []string{"consistency", "quote-grounding", "format"} {
		if !gates[want] {
			t.Fatalf("脏样例应触发 %s 门：\n%s", want, state.FormatViolations(res.Violations))
		}
	}
	if len(res.Suggestions) == 0 {
		t.Fatal("Sweep 应产出 diff 建议")
	}
	for _, s := range res.Suggestions {
		if s.Excerpt == "" {
			t.Fatalf("建议应带原句摘录：%+v", s)
		}
	}
}

// 复现 P0-#6：E14 重跑后换钩子、E15 未承接 → Linkage 必拦；承接后 OK。
func TestLinkageRerunReproE14E15(t *testing.T) {
	c, eps, err := LoadManifest(filepath.Join("..", "..", "examples", "demo", "manifest.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	// e04 重跑后钩子承接词丢失：把 e04 的 pickup 改成 e05 开头没有的词。
	eps[3].Delta.HooksOpened[0].PickupKeywords = []string{"银针"}
	if vs := Linkage(c, eps, 4); !containsMsg(vs, "联动断裂") {
		t.Fatalf("重跑断裂未拦截：%s", state.FormatViolations(vs))
	}
	eps[3].Delta.HooksOpened[0].PickupKeywords = []string{"房契"}
	if vs := Linkage(c, eps, 4); len(vs) != 0 {
		t.Fatalf("承接完整不应违规：%s", state.FormatViolations(vs))
	}
	if vs := Linkage(c, eps, 99); !containsMsg(vs, "不在 episodes") {
		t.Fatalf("未知集号应报错：%s", state.FormatViolations(vs))
	}
}

func TestLoadManifestErrors(t *testing.T) {
	if _, _, err := LoadManifest(filepath.Join(t.TempDir(), "none.yaml")); err == nil {
		t.Fatal("缺 manifest 应报错")
	}
}

func containsMsg(vs []state.Violation, sub string) bool {
	for _, v := range vs {
		if strings.Contains(v.Message, sub) {
			return true
		}
	}
	return false
}
