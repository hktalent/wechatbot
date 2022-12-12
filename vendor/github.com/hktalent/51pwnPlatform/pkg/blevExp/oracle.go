package blevExp

import (
	"encoding/csv"
	"github.com/hktalent/colly"
	"log"
	"os"
	"strconv"
	"strings"
)

// 追加到文件中
func AppendFile(szFile string, a []string, f1 *os.File) *os.File {
	if nil == f1 {
		f, err := os.OpenFile(szFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Println(err)
			return f
		}
		f1 = f
	}
	//defer f.Close()
	w := csv.NewWriter(f1)
	if err := w.Write(a); nil != err {
		log.Println(err)
	}
	w.Flush()
	return f1
}

var bFst = true

func ToCsv(hd, data []string) {
	if bFst {
		AppendFile("oracleAlert.csv", hd, nil)
	}
	AppendFile("oracleAlert.csv", data, nil)
	bFst = false
}

//
func ParseOracle(e *colly.HTMLElement, site *Site, r *CrawlerEngine) {
	e.ForEach("div.otable-w1", func(i int, e1 *colly.HTMLElement) {
		e1.ForEach("tbody", func(i int, e3 *colly.HTMLElement) {
			e3.ForEach("tr", func(i int, e2 *colly.HTMLElement) {
				var doc = map[string]interface{}{"CVE": strings.TrimSpace(e2.ChildText("th"))}
				if !strings.HasPrefix(strings.TrimSpace(e2.ChildText("th")), "CVE-") {
					return
				}
				var aData = []string{strings.TrimSpace(e2.ChildText("th"))}
				var a = strings.Split(`CVE
Product
Component
Protocol
Remote Exploit without Auth
BaseScore
Attack Vector
Attack Complex
Privs Req'd
User Interact
Scope
Confidentiality
Integrity
Availability
Supported Versions Affected
Notes`, "\n")
				var n = 1
				e2.ForEach("td", func(i int, e4 *colly.HTMLElement) {
					if n >= len(a) {
						return
					}
					aData = append(aData, strings.TrimSpace(e4.Text))
					doc[a[n]] = strings.TrimSpace(e4.Text)
					if "BaseScore" == a[n] {
						if f1, err := strconv.ParseFloat(strings.TrimSpace(e4.Text), 32); nil == err {
							doc[a[n]] = f1
						}
					}
					n += 1
				})
				if 3 < len(doc) {
					go ToCsv(a, aData)
					r.Doc <- &IndexData{Index: DefaulIndexName, Id: strings.TrimSpace(e2.ChildText("th")), Doc: doc, FnCbk: func() {}, FnEnd: func() {}}
				}
			})
		})
	})
}
