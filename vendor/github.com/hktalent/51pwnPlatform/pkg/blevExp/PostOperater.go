package blevExp

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/hktalent/colly"
	util "github.com/hktalent/go-utils"
	"strconv"
	"text/template"
)

// 创建模版
func Create4Tplt(name, t string) *template.Template {
	return template.Must(template.New(name).Parse(t))
}

// 合并模版、数据
func GetStr4Tplt(tplt string, data interface{}) string {
	x1 := Create4Tplt("x1", tplt)
	var b bytes.Buffer
	ff := bufio.NewWriter(&b)
	x1.Execute(ff, data)
	ff.Flush()
	return string(b.Bytes())
}

type PostXBreakData struct {
	Total   int // 总记录数
	CurSize int // 当前页数据数目
}

var XBreak = make(map[string]chan *PostXBreakData)

func ToInt(i interface{}) int {
	if n, err := strconv.Atoi(fmt.Sprintf("%v", i)); nil == err {
		return n
	}
	return 0
}

func ParsePostRawData(res *colly.Response, doc chan *IndexData, r1 *Site, reTry chan string) {
	var m1 = map[string]interface{}{}
	if err := json.Unmarshal(res.Body, &m1); nil == err {
		if o := GetJson4Query(m1, ".exploits"); nil != o {
			if a, ok := o.([]interface{}); ok {
				for _, x := range a {
					id := fmt.Sprintf("%v", GetJson4Query(x, ".id"))
					x1 := CvtData(x, id, &[]string{"published"}, nil, "score")
					doc <- &IndexData{Index: r1.IndexName, Id: id, Doc: x1, FnCbk: func() {}, FnEnd: func() {}}
				}
				var Xb = &PostXBreakData{CurSize: len(a), Total: ToInt(GetJson4Query(m1, ".exploits_total"))}
				XBreak[res.Request.URL.String()] <- Xb
				return
			}
		}
	} else {
		reTry <- res.Request.URL.String()
	}
	XBreak[res.Request.URL.String()] <- nil
}

// 开始
// 因为这里会有大量的请求，会阻塞，所以用异步
func DoStartPost(r *CrawlerEngine, u string, r1 *Site) {
	if nil == r1 || nil == r {
		return
	}
	util.DoSyncFunc(func() {
		if r1.PostParmConfig != nil {
			ppc := r1.PostParmConfig
			XBreak[u] = make(chan *PostXBreakData)
			if 0 < len(ppc.Keys) {
				for k, v := range ppc.Keys { // 多个key循环
					for _, i := range v {
						for k1, v1 := range ppc.NumKeys { // 分页范围
							for j := v1[0]; j < v1[1]; j += v1[2] { // 若干页，这里需要加继续、停止的机制
								var m1 = map[string]interface{}{k: i, k1: j}
								postData := GetStr4Tplt(r1.PostData, &m1)
								for _, szUrl := range r1.Start {
									if _, ok := XBreak[u]; !ok {
										XBreak[u] = make(chan *PostXBreakData)
									}

									r.cc.PostRaw(szUrl, []byte(postData))

									pxb := <-XBreak[u]
									if nil != pxb && (j >= pxb.Total || pxb.CurSize < v1[2]) {
										break
									}
								}
							}
						}
					}
				}
			}
		}
	})
}
