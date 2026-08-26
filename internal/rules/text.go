package rules

import (
	"strings"
	"unicode"
)

// IsSentenceBreak 报告 r 是否句读符（。！？；!?; 与换行）。Sentences 切分与
// voice 门禁的说话语境判定共用同一套句读边界。
func IsSentenceBreak(r rune) bool {
	switch r {
	case '。', '！', '？', '；', '!', '?', ';', '\n':
		return true
	}
	return false
}

// Sentences 按中文句读切分，保留原文片段，去掉空白句。
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
		cur.WriteRune(r)
		if IsSentenceBreak(r) {
			flush()
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

// quotePairs 是中文引号配对表（引文接地门与声音指纹门共用同一扫描口径）。
var quotePairs = map[rune]rune{'『': '』', '「': '」', '“': '”'}

// QuoteSpan 是一对引号的开口/闭口 rune 下标。
type QuoteSpan struct{ Open, Close int }

// QuoteSpans 抽取 rs 中全部引号对（『』「」“”）的下标；未闭合的开口引号跳过。
func QuoteSpans(rs []rune) []QuoteSpan {
	var out []QuoteSpan
	for i := 0; i < len(rs); i++ {
		closeR, ok := quotePairs[rs[i]]
		if !ok {
			continue
		}
		for j := i + 1; j < len(rs); j++ {
			if rs[j] == closeR {
				out = append(out, QuoteSpan{i, j})
				break
			}
		}
	}
	return out
}

// PickupKeywordHit 报告承接关键词是否出现在文本中（钩子回收门与 ±1 集联动
// 校验共用）；关键词表中的零值字符串不计为命中，空表不命中。
func PickupKeywordHit(keywords []string, text string) bool {
	for _, kw := range keywords {
		if kw != "" && strings.Contains(text, kw) {
			return true
		}
	}
	return false
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

// IsHan 报告 r 是否汉字。
func IsHan(r rune) bool { return unicode.Is(unicode.Han, r) }

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
			prevHan := i > 0 && IsHan(runes[i-1])
			nextHan := j < len(runes) && IsHan(runes[j])
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

// Normalize 供引文接地：去空白与标点，只留实义字符。
func Normalize(s string) string {
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
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(b)]
}

// Lev 导出编辑距离（子包门禁复用）。
func Lev(a, b string) int { return lev([]rune(a), []rune(b)) }

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
