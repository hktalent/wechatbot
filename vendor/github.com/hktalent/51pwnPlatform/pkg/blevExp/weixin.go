package blevExp

import (
	"context"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
	util "github.com/hktalent/go-utils"
	"log"
	"strings"
	"time"
)

var ctx context.Context
var MyCrmdp *MyChromedp

func init() {
	util.RegInitFunc(func() {
		ctx = context.Background()
		MyCrmdp = GetChromedpInstace(&ctx)
	})
}

var tmot = 10000 * time.Second

func GetWxUrl(r *CrawlerEngine, u string) {
	szP := "https://weixin.sogou.com/link?"
	//u = szP + "__biz=MzU5NDgxODU1MQ==&mid=2247496253&idx=1&sn=83c057ce6f60ce678c7b86f938e48ce4&chksm=fe79d6a5c90e5fb3550db7421e96e74f4955fa5f829741a8555b4c8778cb494db0b3434acbb3&scene=178&cur_album_id=1791937637906219012#rd"
	if !strings.HasPrefix(u, szP) {
		return
	}
	szTitle := "#activity-name"
	c1 := "#js_content"
	var szDate, szWter, szOrg string
	if err := MyCrmdp.DoUrl(u, &map[string]interface{}{}, &tmot, func() *chromedp.Tasks {
		return &chromedp.Tasks{
			chromedp.SendKeys(`body`, kb.End, chromedp.ByQuery),
			chromedp.Text(`#activity-name`, &szTitle, chromedp.ByID),
			chromedp.OuterHTML(`#js_content`, &c1, chromedp.ByID),
			chromedp.Text(`class="rich_media_meta rich_media_meta_text"`, &szDate, chromedp.ByQueryAll),
			chromedp.Text(`#meta_content > span:nth-child(2)`, &szWter, chromedp.ByQueryAll),
			chromedp.Text(`#js_name`, &szOrg, chromedp.ByID),
		}
	}); err != nil {
		log.Println(err)
	}
	if "" != szTitle {
		var m1 = map[string]interface{}{
			"title":  szTitle,
			"org":    szOrg,
			"author": szWter,
			"date":   szDate,
			"body":   r.doImg4Str(c1),
		}
		r.Doc <- &IndexData{
			Doc:   m1,
			Index: DefaulIndexName,
			Id:    GetUrlId(u),
			FnCbk: func() {
				r.log("ok ", szTitle, u)
			}, FnEnd: func() {},
		}
	}
}
