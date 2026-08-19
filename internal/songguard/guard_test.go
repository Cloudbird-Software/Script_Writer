package songguard

import (
	"path/filepath"
	"strings"
	"testing"
)

// 门面契约（深接口唯一入口）：Check 必须产出与 CLI 行为一致的完整报告。
func TestFacadeCheckDemo(t *testing.T) {
	g := New()
	rep, err := g.Check(filepath.Join("..", "..", "examples", "demo", "manifest.yaml"))
	if err != nil {
		t.Fatalf("check demo: %v", err)
	}
	if rep.HasError() {
		t.Fatalf("干净样例不应有硬失败：%s", rep.ViolationsJSON())
	}
	errs, warns, sugg := rep.Counts()
	// demo 是全干净样例：统计必须全零且非负（warns/sugg 允许为 0）。
	if errs != 0 || warns < 0 || sugg < 0 {
		t.Fatalf("counts = (%d, %d, %d)", errs, warns, sugg)
	}
	md := rep.RenderDeliverableMarkdown()
	for _, want := range []string{"人物表", "伏笔台账", "卖点覆盖表"} {
		if !strings.Contains(md, want) {
			t.Fatalf("五件套缺 %q", want)
		}
	}
	if !strings.Contains(rep.SummaryJSON(), `"blocked": false`) {
		t.Fatal("摘要 JSON 应含 blocked=false")
	}
}

func TestFacadeLinkage(t *testing.T) {
	g := New()
	out, err := g.Linkage(filepath.Join("..", "..", "examples", "demo", "manifest.yaml"), 4)
	if err != nil {
		t.Fatalf("linkage: %v", err)
	}
	if !out.OK() {
		t.Fatalf("demo E4 联动应完整：%s", out.Render())
	}
	if _, err := g.Linkage(filepath.Join("..", "..", "examples", "demo", "manifest.yaml"), 99); err == nil {
		t.Fatal("未知集号应报错")
	}
}
