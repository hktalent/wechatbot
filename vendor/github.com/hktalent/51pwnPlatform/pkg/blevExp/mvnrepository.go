package blevExp

import (
	"github.com/hktalent/colly"
	"regexp"
	"strings"
)

type Node struct {
	GroupArtifact string `json:"group_artifact"`
	Ver           string `json:"ver"`
}

var Reg001 = regexp.MustCompile(`^https://mvnrepository.com/artifact/.*/[\d.\\-_]+]$`)

// ,"https://repo1.maven.org/maven2/"
// 解析 repository
func ParseMvnrepositoryData(e *colly.HTMLElement, site *Site, r *CrawlerEngine) {
	defer r.DoAHref(e, site)
	id := GetUrlId(e.Request.URL.String())
	if nil != GetDoc(DefaulIndexName, id) { // 处理过了，就跳过，有没有更新的情况？
		return
	}
	// 符合特征，需要保存
	if Reg001.MatchString(e.Request.URL.String()) {
		var doc = map[string]interface{}{}
		doc["name"] = e.ChildText("body > div > main > div.content > div.im > div.im-header > h2 > a:nth-child(1)")
		doc["ver"] = e.ChildText("body > div > main > div.content > div.im > div.im-header > h2 > a:nth-child(2)")
		// 处理table
		e.ForEach("body > div > main > div.content > table", func(i int, e1 *colly.HTMLElement) {
			e1.ForEach("tr", func(i int, e2 *colly.HTMLElement) {
				k1 := strings.ReplaceAll(e2.ChildText("th"), " ", "")
				szV := strings.TrimSpace(e2.ChildText("td"))
				if "Tags" == k1 || "Repositories" == k1 {
					doc[k1] = strings.ReplaceAll(szV, " ", ",")
				} else {
					doc[k1] = szV
				}
			})
		})
		// Compile Dependencies
		e.ForEach("div.version-section", func(i int, e1 *colly.HTMLElement) {
			szKey := strings.TrimSpace(e1.ChildText("h2"))
			szKey = string(regRplckh.ReplaceAll([]byte(szKey), []byte("")))
			var a = []interface{}{}
			e1.ForEach("tr", func(i int, e2 *colly.HTMLElement) {
				var m1 = map[string]string{}
				m1["Group/Artifact"] = e2.ChildText("td:nth - child(3)")
				m1["Version"] = e2.ChildText("td:nth - child(4)")
				a = append(a, m1)
			})
			if 0 < len(a) {
				doc[szKey] = a
			}
		})
		SendDoc2Buf(r, id, DefaulIndexName, doc)
	}
}

var regRplckh = regexp.MustCompile(`\s*\(\d*\)`)
