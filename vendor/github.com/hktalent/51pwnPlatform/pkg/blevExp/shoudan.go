package blevExp

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
)

//var ppHttp = PipelineHttp.NewPipelineHttp()

// 每页只有100条
func DoOneShodan(data []byte, r *CrawlerEngine, u string, r1 *Site) (int, int) {
	var m1 = map[string]interface{}{}
	var nTotal, nCnt = 0, 0
	if err := json.Unmarshal(data, &m1); nil == err {
		if o1 := GetJson4Query(m1, ".total"); nil != o1 {
			if n1, err := strconv.Atoi(fmt.Sprintf("%v", o1)); nil == err {
				nTotal = n1
			}
		}
		if oA := GetJson4Query(m1, ".matches"); nil != oA {
			if a, ok := oA.([]interface{}); ok {
				nCnt = len(a)
				for _, x := range a {
					if m1, ok := x.(map[string]interface{}); ok {
						m1["type"] = r1.Type
						szId := fmt.Sprintf("%v", GetJson4Query(x, "._shodan.id"))
						docs <- &IndexData{
							Index: DefaulIndexName,
							Id:    szId,
							Doc:   m1,
							FnCbk: fnOk,
							FnEnd: nil,
						}
					}
				}
				n9 := len(docs)
				if 0 < n9 {
					SaveIndexDoc4Batch(DefaulIndexName, docs, func() {
						log.Println("save ok ", nCnt, u)
						if n9 == nCnt {
							r.SaveCcUrls(u)
						}
					}, nil)
				}
			}
		}
	} else {
		log.Println(u, "DoOneShodan json.Unmarshal", err, string(data))
	}
	return nTotal, nCnt
}

// https://github.com/IFLinfosec/shodan-dorks
/*
os:"windows xp"
os:"windows 2000"
os:"linux 2.4"

product:apache
product:"microsoft iis"

net:x.x.x.x/x

asn:ASxxxx

ssl.cert.subject.cn:example.com
ssl.cert.expired:true
ssl.cert.issuer.cn:example.com ssl.cert.subject.cn:example.com
Self signed certificates

org:microsoft
org:"United States Department"

hostname:example.com

device:firewall
device:router
device:wap
device:webcam
device:media
device:"broadband router"
device:pbx
device:printer
device:switch
device:storage
device:specialized
device:phone
device:"voip phone"
device:"voip adaptor"
device:"load balancer"
device:"print server"
device:terminal
device:remote
device:telecom
device:power
device:proxy
device:pda
device:bridge

cpe:apple
country:us
ssl.version:sslv3
link:vpn
port:22
server: nginx
*/
// https://api.shodan.io/shodan/host/search?key=FfH1z0IR5MiktkLbfMlQD93M3lPe32vH&page=1&query=weblogic%20country:%22CN%22
// FfH1z0IR5MiktkLbfMlQD93M3lPe32vH
func QueryShodan(r *CrawlerEngine, u string, r1 *Site) {
	for _, k := range r1.A {
		DoOneKey(r, u, r1, k)
	}
}

func DoOneKey(r *CrawlerEngine, u string, r1 *Site, k string) {
	n := 0
	var nTotal = 0
	bBrk := false
	for {
		n++
		szUrl := fmt.Sprintf("https://api.shodan.io/shodan/host/search?key=%s&page=%d&query=%s", r1.Authorization[0], n, url.QueryEscape(k))
		if nil != r.GetCCUrl(szUrl) {
			continue
		}
		ppHttp.DoGetWithClient4SetHd(nil, szUrl, "GET", nil, func(resp *http.Response, err error, szU string) {
			if nil == err && nil != resp {
				defer resp.Body.Close()
				var data []byte
				if data, err = ioutil.ReadAll(resp.Body); nil == err {
					if n1, n2 := DoOneShodan(data, r, szUrl, r1); nTotal == 0 {
						nTotal = n1 - n2
					} else {
						nTotal -= n2
					}
					return
				}
			}
			if nil != err || nTotal <= 0 {
				bBrk = true
				log.Println(szU, err)
			}
		}, func() map[string]string {
			return r1.Headers
		}, true)
		if bBrk {
			return
		}
	}
}
