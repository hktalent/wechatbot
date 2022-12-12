package blevExp

import (
	"fmt"
	"github.com/hktalent/colly"
	util "github.com/hktalent/go-utils"
)

func ParseJson(res *colly.Response, doc chan *IndexData, r1 *Site, reTry chan string) {
	var m1 = map[string]interface{}{}
	if err := json.Unmarshal(res.Body, &m1); nil == err {
		if r1.JsonPath != "" {
			if o := GetJson4Query(m1, r1.JsonPath); nil != o {
				if a, ok := o.([]interface{}); ok {
					for _, x1 := range a {
						if x, ok := x1.(map[string]interface{}); ok {
							reTry <- fmt.Sprintf("https://www.butian.net/Company/%v", x["company_id"])
							//fmt.Println("get ", x["company_id"], x["company_name"])
						}
					}
				}
			}
		}
	}
}

func DoStartPostBt(r *CrawlerEngine, u string, r1 *Site) {
	if nil == r1 || nil == r {
		return
	}
	util.DoSyncFunc(func() {
		if r1.PostParmConfig != nil {
			ppc := r1.PostParmConfig
			if 0 < len(ppc.Keys) {
				for k, v := range ppc.Keys { // 多个key循环
					for _, i := range v {
						for k1, v1 := range ppc.NumKeys { // 分页范围
							for j := v1[0]; j < v1[1]; j += v1[2] { // 若干页，这里需要加继续、停止的机制
								var m1 = map[string]interface{}{k: i, k1: j}
								postData := GetStr4Tplt(r1.PostData, &m1)
								for _, szUrl := range r1.Start {
									r.cc.PostRaw(szUrl, []byte(postData))
								}
							}
						}
					}
				}
			}
		}
	})
}
