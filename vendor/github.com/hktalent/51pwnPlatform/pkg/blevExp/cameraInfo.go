package blevExp

import (
	"fmt"
	"github.com/hktalent/colly"
	util "github.com/hktalent/go-utils"
	"strconv"
	"strings"
)

func ParseCamera(r *CrawlerEngine, e *colly.HTMLElement, r1 *Site) {
	defer SetCC(e.Request.URL.String())
	a11 := strings.Split(e.Request.URL.String(), "/")
	if 5 > len(a11) {
		return
	}
	s11 := a11[4]
	e.ForEach("#camera_models", func(i int, e1 *colly.HTMLElement) {
		e1.ForEach("tr", func(i int, e2 *colly.HTMLElement) {
			var ac = strings.Split("Model\tProtocol\tPath\tPort", "\t")
			var m = map[string]interface{}{}
			e2.ForEach("td", func(i int, e3 *colly.HTMLElement) {
				switch i {
				case 0, 1, 2:
					m[ac[i]] = strings.TrimSpace(e3.Text)
				case 3:
					s1 := strings.TrimSpace(e3.Text)
					if n, err := strconv.Atoi(s1); err == nil {
						m[ac[i]] = n
					} else {
						m[ac[i]] = s1
					}
				}
			})
			if s1, ok := m["Protocol"]; ok && -1 < strings.Index(fmt.Sprintf("%v", s1), "://") {
				m["type"] = SiteType_Camera
				//DelIndexDoc(DefaulIndexName, util.GetSha1(m))
				m["product"] = s11
				r.Doc <- &IndexData{
					Index: DefaulIndexName,
					Doc:   m,
					Id:    util.GetSha1(m),
					FnCbk: func() {
						r.log("ok ", m["Model"])
					}, FnEnd: func() {},
				}
			} else if o, ok := m["Model"]; !ok || nil == o {
				DelIndexDoc(DefaulIndexName, util.GetSha1(m))
			}
		})
	})
}
