package canon

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

// FuzzValidate 验证任意输入下「YAML 解析 + 六表结构校验」不 panic：
// 非法输入要么解析报错，要么落到 Validate 的 Problem 列表，均不允许崩溃。
// go test 仅执行种子用例；持续 fuzz 由 go test -fuzz=FuzzValidate 触发。
func FuzzValidate(f *testing.F) {
	f.Add([]byte("entities: []"))
	f.Add([]byte("props:\n- id: p1\n  states: [on]\n  transitions: {on: [on]}\n"))
	f.Add([]byte("::not yaml\x00\xff"))
	if files, err := filepath.Glob("testdata/demo/*.yaml"); err == nil {
		for _, p := range files {
			if raw, err := os.ReadFile(p); err == nil {
				f.Add(raw)
			}
		}
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		c := &Canon{}
		if err := yaml.Unmarshal(raw, c); err != nil {
			return
		}
		_ = c.Validate()
	})
}
