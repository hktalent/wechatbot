package blevExp

import (
	"fmt"
	"github.com/hktalent/gson"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func GoogleFy(s string) string {
	//ppHttp.UseHttp2 = true
	//ppHttp.Client = ppHttp.GetClient4Http2()
	szRst := ""
	ppHttp.DoGetWithClient4SetHd(ppHttp.Client, "https://translate.google.cn/translate_a/single", "POST",
		strings.NewReader(fmt.Sprintf("client=webapp&sl=en&tl=zh-CN&hl=zh-CN&dt=at&tk=&q=%s", url.QueryEscape(s))),
		func(resp *http.Response, err error, szU string) {
			if nil == err && 200 == resp.StatusCode {
				var o = map[string]interface{}{}
				if data, err := ioutil.ReadAll(resp.Body); nil == err {
					obj := gson.NewFrom(string(data))
					szRst = obj.Get("trans_result.0.dst").Str()
					//json.Unmarshal(data, &o)
				}
				if "" == szRst {
					if a, ok := o["trans_result"].([]interface{}); ok {
						if m, ok := a[0].(map[string]interface{}); ok {
							szRst = fmt.Sprintf("%v", m["dst"])
						}
					}
				}
			}
		}, func() map[string]string {
			return map[string]string{"content-type": "application/x-www-form-urlencoded",
				"x-requested-with": "XMLHttpRequest", "Connection": "close", "Cache-Control": "no-cache",
				"Host":       "translate.google.cn",
				"Referer":    "https://translate.google.cn/?hl=zh-CN",
				"User-Agent": "Mozilla/4.0 (compatible; MSIE 6.0; Windows NT 5.1; SV1)"}
		}, true)
	return szRst
}

func ShouGouFy(s string) string {
	//ppHttp.UseHttp2 = true
	//ppHttp.Client = ppHttp.GetClient4Http2()
	szRst := ""
	ppHttp.DoGetWithClient4SetHd(ppHttp.Client, "http://config.pinyin.sogou.com/api/app75/translate/getTranslate.php", "POST",
		strings.NewReader(fmt.Sprintf("lang_id=0&srclang=%s&t=%d", url.QueryEscape(s), time.Now().UnixNano())),
		func(resp *http.Response, err error, szU string) {
			if nil == err && 200 == resp.StatusCode {
				var o = map[string]interface{}{}
				if data, err := ioutil.ReadAll(resp.Body); nil == err {
					obj := gson.NewFrom(string(data))
					szRst = obj.Get("trans_result.0.dst").Str()
					//json.Unmarshal(data, &o)
				}
				if "" == szRst {
					if a, ok := o["trans_result"].([]interface{}); ok {
						if m, ok := a[0].(map[string]interface{}); ok {
							szRst = fmt.Sprintf("%v", m["dst"])
						}
					}
				}
			}
		}, func() map[string]string {
			return map[string]string{"content-type": "application/x-www-form-urlencoded",
				"x-requested-with": "XMLHttpRequest", "Connection": "close", "Cache-Control": "no-cache",
				"Host":       "config.pinyin.sogou.com",
				"User-Agent": "Mozilla/4.0 (compatible; MSIE 6.0; Windows NT 5.1; SV1)"}
		}, true)
	return szRst
}
