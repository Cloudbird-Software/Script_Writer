package rules

import (
	"strings"

	"github.com/Cloudbird-Software/Script_Writer/internal/canon"
)

// SoleCharacter 返回文本片段中恰好出现的唯一具名角色（id, 命中名）；
// 零个或多个（归属歧义）返回空。台词归属门/声音指纹门共用的规则版归属判定，
// 代词消解（"他只道："）属 M5 LLM 旁路。
func SoleCharacter(c *canon.Canon, text string) (id, name string) {
	for _, e := range c.Entities {
		if e.Type != "character" {
			continue
		}
		for _, n := range append([]string{e.CanonicalName}, e.Aliases...) {
			if n != "" && strings.Contains(text, n) {
				if id != "" {
					return "", "" // 多个角色，歧义
				}
				id, name = e.ID, n
				break
			}
		}
	}
	return id, name
}
