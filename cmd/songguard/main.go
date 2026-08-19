// songguard——小说/短剧跨集一致性校验工具（Go 库 + thin CLI）。
//
// 输入：六张 canon 表(YAML) + 各集正文 + 各集 delta 申报
// 输出：结构化违规报告 + 交付五件套报表（见 internal/passes）。
//
// 子命令随 M1~M3 逐步落地（ADR-0001，issue #1）：
//
//	songguard check <manifest.yaml>   全量校验 + 报表
//	songguard linkage --ep N --old <file>   重跑 ±1 集联动校验
package main

import (
	"fmt"
	"os"
)

var version = "dev"

const usage = `songguard %s——小说/短剧跨集一致性校验工具

用法:
  songguard check <manifest.yaml>          全量校验（canon 结构 + 状态台账 + 门禁 + 全局 pass）
  songguard linkage -manifest <m.yaml> -ep <N> -old <file>
                                           重跑 ±1 集联动校验（E14→E15 类断裂）
  songguard version                        打印版本

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
	case "check", "linkage":
		// M3 落地（ADR-0001 分期）；当前骨架仅声明接口。
		fmt.Fprintf(os.Stderr, "songguard: %s 将在 M3 交付（见 adr/0001）\n", os.Args[1])
		os.Exit(2)
	default:
		fmt.Fprintf(os.Stderr, "songguard: 未知子命令 %q\n", os.Args[1])
		fmt.Printf(usage, version)
		os.Exit(2)
	}
}
