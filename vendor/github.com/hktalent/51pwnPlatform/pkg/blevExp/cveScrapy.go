package blevExp

import (
	"fmt"
	"github.com/hktalent/colly"
	"io"
	"io/ioutil"
	"sync/atomic"
)

func ParseZipFile(res *colly.Response, doc chan *IndexData, r1 *Site, reTry chan string) {
	DoZipFile(res, reTry, func(oI io.Reader) {
		if data, err := ioutil.ReadAll(oI); nil == err {
			var i, j int32 = 0, 0
			n := DoJsonData(data, func(n int) {
				atomic.AddInt32(&j, int32(n))
			})
			if i == int32(n) {
				m1 := map[string]interface{}{"lastModifiedDate": res.Headers.Get("lastModifiedDate"), "content-length": res.Headers.Get("content-length")}
				r1.CrawlerEngine.SaveCcUrls(res.Request.URL.String(), m1)
			} else {
				reTry <- res.Request.URL.String()
			}
		}
	})
}

var NoDo = func() {}

func DoJsonData(data []byte, endCbk func(n int)) int {
	var m1 map[string]interface{}
	n11 := 0

	defer func() {
		n1 := len(docs)
		if 0 < n1 {
			SaveIndexDoc4Batch(DefaulIndexName, docs, NoDo, func() {
				endCbk(n1)
			})
		}
	}()
	if err := json.Unmarshal(data, &m1); nil == err {
		if o := GetJson4Query(m1, ".CVE_Items"); nil != o {
			if a, ok := o.([]interface{}); ok {
				n11 = len(a)
				for _, o1 := range a {
					id := GetJson4Query(o1, ".cve.CVE_data_meta.ID")
					data, err = json.Marshal(o1)
					if nil == err {
						szId := fmt.Sprintf("%v", id)
						if GetDoc(DefaulIndexName, szId) != nil {
							endCbk(1)
							continue
						}
						o1 = CvtData(o1, szId, &[]string{"lastModifiedDate", "publishedDate"}, nil, "configurations.CVE_data_version", "cve.data_version", "impact.baseMetricV3.cvssV3.baseScore", "impact.baseMetricV3.cvssV3.version", "impact.baseMetricV3.exploitabilityScore", "impact.baseMetricV3.impactScore")
						docs <- &IndexData{
							Index: DefaulIndexName,
							Id:    szId,
							Doc:   o1,
							FnCbk: NoDo,
							FnEnd: nil,
						}
					}
					n1 := len(docs)
					if 10000 < n1 {
						SaveIndexDoc4Batch(DefaulIndexName, docs, NoDo, func() {
							endCbk(n1)
						})
					}
				}
			}
		}
	}
	return n11
}
