package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// buildCLI 构建一次二进制供冒烟测试复用（测试工作目录 = 包目录）。
// 二进制名带平台后缀：Windows 上 exec 需 .exe 扩展名才能被找到。
func buildCLI(t *testing.T) string {
	t.Helper()
	name := "songguard"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	bin := filepath.Join(t.TempDir(), name)
	out, err := exec.Command("go", "build", "-o", bin, "../../cmd/songguard").CombinedOutput()
	if err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	return bin
}

func runCLI(t *testing.T, bin string, args ...string) (string, error) {
	t.Helper()
	out, err := exec.Command(bin, args...).CombinedOutput()
	return string(out), err
}

// CLI 冒烟：干净样例退出码 0 且 blocked=false。
func TestCLICheckSmoke(t *testing.T) {
	if testing.Short() {
		t.Skip("构建较慢，-short 跳过")
	}
	bin := buildCLI(t)
	outDir := t.TempDir()
	manifest := filepath.Join("..", "..", "examples", "demo", "manifest.yaml")
	out, err := runCLI(t, bin, "check", "-out", outDir, manifest)
	if err != nil {
		t.Fatalf("干净样例应退出 0：%v\n%s", err, out)
	}
	if !strings.Contains(out, `"blocked": false`) {
		t.Fatalf("应输出 blocked=false：%s", out)
	}
	for _, f := range []string{"deliverable.md", "sweep.md"} {
		if _, err := os.Stat(filepath.Join(outDir, f)); err != nil {
			t.Fatalf("报表 %s 未生成：%v", f, err)
		}
	}
}

func TestCLILinkageSmoke(t *testing.T) {
	if testing.Short() {
		t.Skip("构建较慢，-short 跳过")
	}
	bin := buildCLI(t)
	manifest := filepath.Join("..", "..", "examples", "demo", "manifest.yaml")
	out, err := runCLI(t, bin, "linkage", "-manifest", manifest, "-ep", "4")
	if err != nil {
		t.Fatalf("承接完整应退出 0：%v\n%s", err, out)
	}
	if !strings.Contains(out, "OK") {
		t.Fatalf("应输出 OK：%s", out)
	}
}
