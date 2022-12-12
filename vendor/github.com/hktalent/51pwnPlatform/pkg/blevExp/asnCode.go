package blevExp

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"github.com/hktalent/colly"
	"io"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

func Data2Map(hd, data []string) map[string]interface{} {
	var oR = map[string]interface{}{}
	var err error
	for i, x := range data {
		if 0 == i {
			a := strings.Split(strings.TrimSpace(x), "-")
			oR["start"], err = strconv.Atoi(a[0])
			if nil != err {
				oR["start"] = a[0]
			}
			if 1 < len(a) {
				oR["end"], err = strconv.Atoi(a[1])
				if nil != err {
					oR["end"] = a[1]
				}
			}
		} else if 1 == i {
			oR[hd[i]] = strings.TrimSpace(strings.ReplaceAll(x, "Assigned by ", ""))
		} else {
			oR[hd[i]] = strings.TrimSpace(x)
		}
	}
	return oR
}
func ReadTxt(doc chan *IndexData) {
	if data, err := ioutil.ReadFile("/Users/51pwn/Downloads/asnList.txt"); nil == err {
		a := strings.Split(strings.TrimSpace(string(data)), "\n")
		var err error
		for _, x := range a {
			var oR = map[string]interface{}{}
			x = strings.TrimSpace(x)
			if i := strings.Index(x, " "); 0 < i {
				oR["start"], err = strconv.Atoi(x[2:i])
				if nil != err {
					log.Println(err)
				}
				oR["end"] = oR["start"]
				oR["Description"] = strings.TrimSpace(x[i+1:])
			}
			if 0 < len(oR) {
				doc <- &IndexData{
					Index: DefaulIndexName,
					Id:    fmt.Sprintf("%d-%d", oR["start"], oR["end"]),
					Doc:   oR,
					FnCbk: func() {}, FnEnd: func() {},
				}
			}
		}
	}
}

func ParseAsnCsv(res *colly.Response, doc chan *IndexData, r1 *Site, reTry chan string) {
	//go ReadTxt(doc)
	szUrl := res.Request.URL.String()
	if strings.HasSuffix(szUrl, ".csv") {
		id := GetUrlId(szUrl)
		if o := GetDoc(DefaulIndexName, id); nil == o {
			r := csv.NewReader(bytes.NewReader(res.Body))
			bFirst := true
			var Hd []string
			var n int32 = 0
			var aN int32 = 0
			Wg := sync.WaitGroup{}
			for {
				record, err := r.Read()
				if err == io.EOF {
					break
				}
				if err != nil {
					log.Println("r.Read() is err", err)
					continue
				}
				if bFirst { // header
					bFirst = false
					Hd = record
					continue
				}
				if oR := Data2Map(Hd, record); 0 < len(oR) {
					Wg.Add(1)
					doc <- &IndexData{
						Index: DefaulIndexName,
						Id:    record[0],
						Doc:   oR,
						FnCbk: func() {
							atomic.AddInt32(&n, 1)
						}, FnEnd: func() {
							Wg.Done()
							atomic.AddInt32(&aN, 1)
						},
					}
				}
			}
			Wg.Wait()
			if aN == n {
				SetCC(szUrl)
			}
		}
	}
}
