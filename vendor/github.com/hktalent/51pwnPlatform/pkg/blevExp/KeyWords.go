package blevExp

import (
	_ "embed"
	util "github.com/hktalent/go-utils"
	"regexp"
	"strings"
)

// 关键词识别、标识
//go:embed dict/KeyWords.txt
var dicts string

var keyReg *regexp.Regexp

func init() {
	util.RegInitFunc(func() {
		dicts = strings.TrimSpace(dicts)
		keyReg = regexp.MustCompile(`(?i)\b(` + strings.ReplaceAll(dicts, "\n", "|") + `)\b`)
	})
}

// 抽取关键词
func (r *CrawlerEngine) ExtractTags(s string) string {
	var m = map[string]string{}
	var a1 []string
	a := keyReg.FindAllString(s, -1)
	if 0 < len(a) {
		for _, x := range a {
			if _, ok := m[x]; ok {
				continue
			}
			a1 = append(a1, x)
			m[x] = ""
		}
	}
	return strings.Join(a1, ",")
}
