package blevExp

import (
	"github.com/hktalent/colly"
	util "github.com/hktalent/go-utils"
	"io/ioutil"
	"strings"
	"sync"
)

var once sync.Once

var szType = "Default Passwords"

func Load(r *CrawlerEngine) {
	for _, x9 := range []string{"passwords.csv", "DefaultCreds-Cheat-Sheet.csv"} {
		if r.CheckIsOk(x9) {
			continue
		}
		r.SetCC(x9)
		if data, err := ioutil.ReadFile("config/dfpswd/" + x9); nil == err {
			a := strings.Split(strings.TrimSpace(string(data)), "\n")
			hd := strings.Split(a[0], ",")
			a = a[1:]
			for _, x := range a {
				var m = map[string]string{}
				a1 := strings.Split(x, ",")
				for i, j := range a1 {
					j = strings.TrimSpace(j)
					if "" != j {
						m[hd[i]] = j
					}
				}
				m["type"] = szType
				r.Doc <- &IndexData{
					Index: DefaulIndexName,
					Doc:   m,
					Id:    util.GetSha1(m),
					FnCbk: func() {
						r.log("ok ", m)
					}, FnEnd: func() {},
				}
			}
		}
	}
}

func DoOneM(m map[string]string, r *CrawlerEngine) {
	m["type"] = szType
	if 1 < len(m) {
		if _, ok := m["Password"]; ok {
			r.Doc <- &IndexData{
				Index: DefaulIndexName,
				Doc:   m,
				Id:    util.GetSha1(m),
				FnCbk: func() {
					r.log("ok ", m)
				}, FnEnd: func() {},
			}
		}
	}
}

func ParseTable(r *CrawlerEngine, e1 *colly.HTMLElement, hd []string, szTitle string) {
	var m = map[string]string{}
	e1.ForEach("td,th", func(j int, e2 *colly.HTMLElement) {
		m[hd[j]] = strings.TrimSpace(e2.Text)
	})
	if "" != szTitle {
		m["title"] = szTitle
	}
	DoOneM(m, r)
}

func ParsePswd1(r *CrawlerEngine, e *colly.HTMLElement, r1 *Site) {
	aT := []string{
		"#container > div.blogSingle > div.singleBlogBottom > div > div > div > div.c9 > div > div",
		"#tablepress-9",
		"#mntl-sc-block_1-0-7 > div > table",
	}
	szFj := strings.TrimSpace(e.ChildText("#Router_Usernames_and_PasswordsnbspDefault_Credentials"))
	if "" == szFj {
		szFj = strings.TrimSpace(e.ChildText("#article-heading_1-0"))
	}
	for _, s := range aT {
		// table
		e.ForEach(s, func(i int, e1 *colly.HTMLElement) {
			var hd []string
			if 0 == i {
				e1.ForEach("td,th", func(j int, e2 *colly.HTMLElement) {
					hd = append(hd, strings.TrimSpace(e2.Text))
				})
			} else {
				ParseTable(r, e1, hd, szFj)
			}
		})
	}
}

// type:"Default Passwords" +productvendor: 3COM
func ParsePswd(r *CrawlerEngine, e *colly.HTMLElement, r1 *Site) {
	defer r.SetCC(e.Request.URL.String())
	once.Do(func() {
		go Load(r)
	})
	if -1 == strings.Index(e.Request.URL.String(), "https://cirt.net") {
		ParsePswd1(r, e, r1)
		return
	}
	go r.DoAHref(e, r1)
	e.ForEach("div.field-item", func(i int, e1 *colly.HTMLElement) {
		e1.ForEach("table", func(j int, e2 *colly.HTMLElement) {
			var m = map[string]string{}
			e2.ForEach("tr", func(x int, e3 *colly.HTMLElement) {
				switch x {
				case 0:
					m["product"] = strings.TrimSpace(e3.Text)
				default:
					var t, v string
					e3.ForEach("td", func(p int, e4 *colly.HTMLElement) {
						switch p {
						case 0:
							t = strings.TrimSpace(e4.Text)
						case 1:
							v = strings.TrimSpace(e4.Text)
						}
					})
					m[t] = v
				}
			})
			m["type"] = szType
			DoOneM(m, r)
		})
	})
}
