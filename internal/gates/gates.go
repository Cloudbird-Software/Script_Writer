// Package gates 实现 M2 五道硬门（issue #1 §B-2，全部纯函数、零 LLM）：
//
//	Format         格式门：字数区间 ±10%、全半角统一、人名禁非中文字符、无 markdown 残留
//	Consistency    一致性门：forbidden_names 精确比对 + 称谓/地名后缀漂移检测
//	Relationship   关系门：已有相遇记录的角色禁止"初次相识"叙事
//	QuoteGrounding 引文接地门："想起某人说过"必须能在前文逐字或近似检索到出处
//	HookPayoff     钩子/回收门：每集必有钩子/按钮、悬置时限、P0 结局前清账、相邻集钩接
//
// 输入：canon + 各集正文（[]state.Episode）+ 各集后快照（[]state.Snapshot，与 eps 等长同序）。
// 输出：[]state.Violation。门禁不改写正文、不动台账。
package gates

import (
	"strings"
	"unicode"
)

// Sentences 按中文句读切分（。！？；!?; 与换行），保留原文片段，去掉空白句。
func Sentences(text string) []string {
	var out []string
	var cur strings.Builder
	flush := func() {
		if s := strings.TrimSpace(cur.String()); s != "" {
			out = append(out, s)
		}
		cur.Reset()
	}
	for _, r := range text {
		switch r {
		case '。', '！', '？', '；', '!', '?', ';', '\n':
			cur.WriteRune(r)
			flush()
		default:
			cur.WriteRune(r)
		}
	}
	flush()
	return out
}

// CountChars 统计正文字数（非空白 rune 数，issue #1 的"字数"口径）。
func CountChars(text string) int {
	n := 0
	for _, r := range text {
		if !unicode.IsSpace(r) {
			n++
		}
	}
	return n
}

// FirstN 返回文本前 n 个 rune。
func FirstN(text string, n int) string {
	var b strings.Builder
	i := 0
	for _, r := range text {
		if i >= n {
			break
		}
		b.WriteRune(r)
		i++
	}
	return b.String()
}

var fullWidthPunct = "，。：；！？“”‘’（）《》、—…·"
var halfWidthPunct = ",:;!?\"'()"

// PunctMix 返回文本中同时出现的全角/半角标点样本（用于格式门）。
func PunctMix(text string) (fullSample, halfSample string, mixed bool) {
	var f, h string
	for _, r := range text {
		s := string(r)
		if strings.ContainsRune(fullWidthPunct, r) && f == "" {
			f = s
		}
		if strings.ContainsRune(halfWidthPunct, r) && h == "" {
			h = s
		}
	}
	return f, h, f != "" && h != ""
}

func isHan(r rune) bool { return unicode.Is(unicode.Han, r) }
func isLatin(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

// MixedScriptNames 检出"单字母拉丁 + 汉字"的混排名（如 A福）；
// ≥2 个字母的拉丁词（LED/TTS 等）不报。
func MixedScriptNames(text string) []string {
	runes := []rune(text)
	var out []string
	i := 0
	for i < len(runes) {
		if !isLatin(runes[i]) {
			i++
			continue
		}
		j := i
		for j < len(runes) && isLatin(runes[j]) {
			j++
		}
		runLen := j - i
		if runLen == 1 {
			prevHan := i > 0 && isHan(runes[i-1])
			nextHan := j < len(runes) && isHan(runes[j])
			if prevHan || nextHan {
				start := i
				if prevHan {
					start = i - 1
				}
				end := j
				if nextHan {
					end = j + 1
				}
				out = append(out, string(runes[start:end]))
			}
		}
		i = j
	}
	return out
}

// MarkdownResidue 检出 markdown 残留（#/代码围栏/加粗/行首列表标记）。
func MarkdownResidue(text string) []string {
	var out []string
	for _, ln := range strings.Split(text, "\n") {
		t := strings.TrimSpace(ln)
		switch {
		case strings.HasPrefix(t, "#"),
			strings.HasPrefix(t, "```"),
			strings.Contains(t, "**"),
			strings.HasPrefix(t, "- "):
			out = append(out, t)
		}
	}
	return out
}

// normalize 供引文接地：去空白与标点，只留实义字符。
func normalize(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// lev 是 rune 版 Levenshtein 距离（引文/姓名相似度用，正文规模小，够用）。
func lev(a, b []rune) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for j := 0; j <= len(b); j++ {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min3(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(b)]
}

func min3(a, b, c int) int {
	if b < a {
		a = b
	}
	if c < a {
		a = c
	}
	return a
}

// Similarity 归一化编辑相似度（1 = 相同）。
func Similarity(a, b string) float64 {
	ra, rb := []rune(a), []rune(b)
	if len(ra) == 0 && len(rb) == 0 {
		return 1
	}
	d := lev(ra, rb)
	maxLen := len(ra)
	if len(rb) > maxLen {
		maxLen = len(rb)
	}
	return 1 - float64(d)/float64(maxLen)
}
