package blevExp

import (
	"archive/zip"
	"bytes"
	"fmt"
	"github.com/hktalent/colly"
	xj "github.com/hktalent/goxml2json"
	jsoniter "github.com/json-iterator/go"
	"io"
	"io/ioutil"
	"regexp"
	"strings"
	"sync"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary
var r009 = regexp.MustCompile("(\\/attached\\/\\/\\d*.zip)")

// 处理、返回 zip 链接
func ParseCNNVDHref(e *colly.HTMLElement, site *Site, sout chan string) {
	if s1, err := e.DOM.Html(); nil == err {
		a := r009.FindAllString(s1, -1)
		x07 := map[string]string{}
		for _, s := range a {
			s = "http://www.cnnvd.org.cn" + strings.ReplaceAll(s, "//", "/")
			if _, ok := x07[s]; ok {
				continue
			}
			x07[s] = "1"
			sout <- s
		}
	}

}

func Cvt2Date(a []string, o map[string]interface{}, si *Site) interface{} {
	for _, x := range a {
		o[x] = ParseDate(fmt.Sprintf("%v", GetJson4Query(o, "."+x)), si.DateFormat)
	}
	return o
}

//var oneDo sync.Once
//var docS = make(chan *IndexData, 5005)
//
//func RunSave() {
//	util.DoSyncFunc(func() {
//		nC := 0
//		tck := time.NewTicker(60 * time.Second)
//		var fnDoOne = func(nC int) func() {
//			return func() {
//				n3 := len(docS)
//				if nC < n3 {
//					SaveIndexDoc4Batch(DefaulIndexName, docS, func() {
//						fmt.Println("Save ok ", n3)
//					}, func() {})
//				}
//			}
//		}
//		defer func() {
//			tck.Stop()
//			fnDoOne(0)()
//		}()
//		for {
//			select {
//			case <-tck.C:
//				if 5000 > len(docS) {
//					nC++
//				} else {
//					nC = 0
//				}
//				if 3 < nC {
//					fnDoOne(0)()
//					break
//				}
//			default:
//				fnDoOne(5000)()
//			}
//		}
//	})
//}

func LoadAllXml(doc chan *IndexData, r1 *Site) {
	//doc = docS
	//oneDo.Do(func() {
	//	RunSave()
	//})
	a := strings.Split(`20221014163931532.zip
20221014163942896.zip
20221014163954752.zip
20221014164013998.zip
20221014164024391.zip
20221014164035756.zip
20221014164045984.zip
20221014164056614.zip
20221014164106145.zip
20221014164116932.zip
20221014164128157.zip
20221014164142764.zip
20221014164154672.zip
20221014164208766.zip
20221014164220545.zip
20221014164231756.zip
20221014164245393.zip
20221014164302843.zip
20221014164314836.zip
20221014164329481.zip
20221014164359978.zip
20221014164411676.zip
20221014164427813.zip
20221020165553725.zip
20221020165605984.zip
20221020165617764.zip`, "\n")
	for _, x := range a {
		szUrl := fmt.Sprintf("http://www.cnnvd.org.cn/attached/%s", x)
		if data, err := ioutil.ReadFile(fmt.Sprintf("/Users/51pwn/MyWork/bleve-explorer/tools/cnnvd/%s", x)); nil == err {
			DoCnnvdReader(data, szUrl, doc, r1)
		}
	}
}

func DoCnnvdReader(data []byte, szUrl string, doc chan *IndexData, r1 *Site) error {
	n11 := int64(len(data))
	if zi, err := zip.NewReader(io.NewSectionReader(bytes.NewReader(data), 0, n11), n11); nil == err {
		for _, f := range zi.File {
			if fio, err := f.Open(); nil == err {
				DoJsonData2(nil, szUrl, doc, r1, fio)
				fio.Close()
			}
		}
		return nil
	} else {
		return err
	}
}

func DoJsonData2(res *colly.Response, szUrl string, doc chan *IndexData, r1 *Site, fio io.Reader) {
	if j1, err := xj.Convert(fio); nil == err {
		var m1 = map[string]interface{}{}
		if err := json.Unmarshal(j1.Bytes(), &m1); nil == err {
			if o, ok := m1["cnnvd"]; ok {
				if m3, ok := o.(map[string]interface{}); ok {
					if m2, ok := m3["entry"]; ok {
						if aE, ok := m2.([]interface{}); ok {
							wg := sync.WaitGroup{}
							for _, x := range aE {
								if m5, ok := x.(map[string]interface{}); ok {
									id := strings.TrimSpace(fmt.Sprintf("%v", m5["vuln-id"]))
									df01 := []string{"published", "modified"}
									//o11 := Cvt2Date(df01, m5, r1)
									wg.Add(1)
									oX1 := CvtData(m5, id, &df01, nil)
									doc <- &IndexData{
										Index: DefaulIndexName,
										Id:    id,
										Doc:   oX1,
										FnCbk: func() {
											fmt.Printf("save ok, cnnvd %s\r", id)
										},
										FnEnd: func() {
											wg.Done()
										},
									}
								}
							}
							wg.Wait()
							if nil != res {
								m1 := map[string]interface{}{"lastModifiedDate": res.Headers.Get("lastModifiedDate"), "content-length": res.Headers.Get("content-length")}
								r1.CrawlerEngine.SaveCcUrls(res.Request.URL.String(), m1)
							} else {
								SetCC(szUrl)
							}
						}

					}
				}
			}
		}
	}
}

// 解压zip，并入库
func ParseCNNVDZip(res *colly.Response, doc chan *IndexData, r1 *Site, reTry chan string) {
	DoZipFile(res, reTry, func(fio io.Reader) {
		DoJsonData2(res, res.Request.URL.String(), doc, r1, fio)
	})
}
