// songguard——小说/短剧跨集一致性校验工具（Go 库 + thin CLI）。
//
// 输入：六张 canon 表(YAML) + 各集正文 + 各集 delta 申报
// 输出：结构化违规报告 + 交付五件套报表（见 internal/passes）。
//
//	songguard check <manifest.yaml>   全量校验 + 报表
//	songguard linkage -manifest <m.yaml> -ep <N>   重跑 ±1 集联动校验
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Cloudbird-Software/Script_Writer/internal/passes"
	"github.com/Cloudbird-Software/Script_Writer/internal/state"
)

var version = "dev"

const usage = `songguard %s——小说/短剧跨集一致性校验工具

用法:
  songguard check [-out <dir>] <manifest.yaml>
                                           全量校验（canon 结构 + 状态台账 + 门禁 + 全局 pass）
  songguard linkage -manifest <m.yaml> -ep <N>
                                           重跑 ±1 集联动校验（E14→E15 类断裂）
  songguard version                        打印版本

check 产出（stdout 摘要 JSON + -out 目录）:
  deliverable.md   交付五件套（人物表/伏笔台账/卖点覆盖/风险清单/beat+钩子）
  sweep.md         一致性巡检建议（只 diff 建议，不重写全文）
  violations.json  结构化违规报告

详见 https://github.com/Cloudbird-Software/Script_Writer/issues/1
`

func main() {
	if len(os.Args) < 2 {
		fmt.Printf(usage, version)
		os.Exit(2)
	}
	switch os.Args[1] {
	case "version":
		fmt.Println(version)
	case "check":
		os.Exit(runCheck(os.Args[2:]))
	case "linkage":
		os.Exit(runLinkage(os.Args[2:]))
	default:
		fmt.Fprintf(os.Stderr, "songguard: 未知子命令 %q\n", os.Args[1])
		fmt.Printf(usage, version)
		os.Exit(2)
	}
}

// checkReport 是 stdout 摘要 JSON 的形态。
type checkReport struct {
	Blocked     bool              `json:"blocked"`
	Errors      int               `json:"errors"`
	Warns       int               `json:"warns"`
	Suggestions int               `json:"suggestions"`
	Violations  []state.Violation `json:"violations"`
}

func runCheck(args []string) int {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	outDir := fs.String("out", ".", "报表输出目录")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "用法: songguard check [-out <dir>] <manifest.yaml>\n")
		return 2
	}
	c, eps, err := passes.LoadManifest(fs.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "songguard: %v\n", err)
		return 2
	}
	res, err := passes.Run(c, eps)
	if err != nil {
		fmt.Fprintf(os.Stderr, "songguard: %v\n", err)
		return 2
	}

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "songguard: 创建输出目录: %v\n", err)
		return 2
	}
	files := map[string]string{
		"deliverable.md":  res.Deliverable.RenderMarkdown(),
		"sweep.md":        passes.RenderSuggestionsMarkdown(res.Suggestions),
		"violations.json": mustJSONIndent(res.Violations),
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(*outDir, name), []byte(content), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "songguard: 写 %s: %v\n", name, err)
			return 2
		}
	}

	rep := checkReport{
		Blocked:     res.Deliverable.Blocked,
		Suggestions: len(res.Suggestions),
		Violations:  res.Violations,
	}
	for _, v := range res.Violations {
		switch v.Severity {
		case state.SeverityError:
			rep.Errors++
		case state.SeverityWarn:
			rep.Warns++
		}
	}
	fmt.Println(mustJSONIndent(rep))
	if res.HasError() {
		fmt.Fprintf(os.Stderr, "songguard: 存在硬失败（errors=%d, blocked=%v），详见 violations.json\n", rep.Errors, rep.Blocked)
		return 1
	}
	return 0
}

func runLinkage(args []string) int {
	fs := flag.NewFlagSet("linkage", flag.ExitOnError)
	manifest := fs.String("manifest", "", "manifest.yaml 路径")
	ep := fs.Int("ep", 0, "被重跑的集号")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *manifest == "" || *ep <= 0 {
		fmt.Fprintf(os.Stderr, "用法: songguard linkage -manifest <m.yaml> -ep <N>\n")
		return 2
	}
	c, eps, err := passes.LoadManifest(*manifest)
	if err != nil {
		fmt.Fprintf(os.Stderr, "songguard: %v\n", err)
		return 2
	}
	vs := passes.Linkage(c, eps, *ep)
	if len(vs) == 0 {
		fmt.Printf("OK: E%d ±1 集联动承接完整（前集钩子已承接、本集钩子已有下文）\n", *ep)
		return 0
	}
	fmt.Fprint(os.Stderr, state.FormatViolations(vs))
	hasErr := false
	for _, v := range vs {
		if v.Severity == state.SeverityError {
			hasErr = true
		}
	}
	if hasErr {
		return 1
	}
	fmt.Println("OK: 仅有 warn 级提示（如未申报 pickup_keywords），无联动断裂")
	return 0
}

func mustJSONIndent(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		// 仅对受控结构序列化，不可达；保守返回空对象。
		return "{}"
	}
	return string(b)
}
