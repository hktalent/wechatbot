package blevExp

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/gin-gonic/gin"
	"github.com/hktalent/51pwnPlatform/cache"
	"github.com/hktalent/51pwnPlatform/lib/hacker"
	"github.com/hktalent/colly"
	"github.com/hktalent/colly/extensions"
	util "github.com/hktalent/go-utils"
	"github.com/robfig/cron/v3"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"hash/fnv"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

var devDebug = true
var Ryear = regexp.MustCompile(`\{year\}`)

type IndexData struct {
	Index string
	Id    string
	Doc   interface{}
	FnCbk func()
	FnEnd func()
}

// 爬数据，并写入索引
type CrawlerEngine struct {
	cc         *colly.Collector
	Site       []*Site
	cron       *cron.Cron
	AsyncVisit chan string
	Doc        chan *IndexData
	cacheVisit *cache.Cache
	VisitPool  chan string
	// Host2Site  map[string]*Site `json:"Host2Site"`
	Host2Site *MapCcSiteImp `json:"Host2Site"`
}

func (r *CrawlerEngine) InitRoter(r001 *gin.Engine) {
	r001.Handle("POST", "/api/crawler", func(c *gin.Context) {
		if s, ok := c.GetPostForm("u"); ok {
			if a := strings.Split(strings.TrimSpace(s), "\n"); 0 < len(a) {
				st := r.GetUrlSite(ChinaWebUrl)
				st.Start = append(st.Start, a...)
				st.NoRptStart()
				for _, x := range a {
					r.Visit(x)
				}
			}
		}
	})
}

var Craler *CrawlerEngine

// 创建实例
func NewCrawlerImp(r001 *gin.Engine) *CrawlerEngine {
	if nil != Craler {
		return Craler
	}
	Craler = &CrawlerEngine{
		AsyncVisit: make(chan string, 1000),
		Doc:        make(chan *IndexData, 1000),
		Host2Site:  NewMapCcSiteImp(),
		cacheVisit: cache.NewCache(2*time.Minute, time.Minute)}
	Craler.init()
	Craler.InitRoter(r001)
	return Craler
}

// ip url，都算做chinaweb
func (r *CrawlerEngine) TestIp(s string, fnCbk func(...any) any, a ...any) any {
	if aIps := ipv4Reg.FindAllString(s, -1); 0 < len(aIps) {
		var a1 = []interface{}{ChinaWebUrl}
		a1 = append(a1, a...)
		return fnCbk(a1...)
	}
	return nil
}

// 判断 s 是否在a，或者在 st 中
func (r *CrawlerEngine) Has(a1 ...any) bool {
	return r.Has1(a1...).(bool)
}
func (r *CrawlerEngine) GetSite4kv(host string) *Site {
	var st = &Site{}
	r.Host2Site.Get(func(i *Site) {
		st = i
	}, host)
	if nil == st.Start || 0 == len(st.Start) {
		return nil
	}
	return r.initSite(st)
}
func (r *CrawlerEngine) Has1(a1 ...any) any {
	var s string = a1[0].(string)
	var a []string = a1[1].([]string)
	var st *Site = a1[2].(*Site)
	if oU1, err := url.Parse(s); nil == err {
		if nil != r.GetSite4kv(oU1.Host) {
			return true
		}
		if a3 := r.TestIp(s, r.Has1, a, st); nil != a3 {
			return a3
		}
	}
	return false
}

func (r *CrawlerEngine) FixedUrl(a1 string, e1 *colly.HTMLElement) string {
	return strings.ReplaceAll(r.FixedUrl1(a1, e1), " ", "")
}

// 修正不规范的url
func (r *CrawlerEngine) FixedUrl1(a1 string, e1 *colly.HTMLElement) string {
	if strings.HasPrefix(a1, "https://security.snyk.io/vuln/") {
		s := string(Fix4Snyk.ReplaceAll([]byte(a1), []byte(".io/vuln/")))
		return s
	}
	oR := regexp.MustCompile("(" + regexp.QuoteMeta(e1.Request.URL.Scheme+"://"+e1.Request.URL.Host) + `[^ \"\r\n]*)`)
	aR := oR.FindAllString(a1, -1)
	if 0 < len(aR) {
		return aR[0]
	}
	a1 = strings.TrimSpace(string(FixUrl.ReplaceAll([]byte(a1), []byte(""))))
	a := strings.Split(a1, "\n")
	return strings.TrimSpace(a[len(a)-1])
}

// 处理页面中的
func (r *CrawlerEngine) DoAHref(e *colly.HTMLElement, st *Site) bool {
	bRst := false
	e.ForEach("a", func(i int, e1 *colly.HTMLElement) {
		a1 := strings.TrimSpace(e1.Attr("href"))
		if "" == a1 {
			return
		}
		//log.Println("math ", a1)
		// url 转换为全路径
		szUrl := strings.TrimSpace(e.Request.URL.String())
		oBase, err := url.Parse(szUrl)
		if nil != err {
			log.Println(err, szUrl)
			return
		}
		// 同域名检查，特殊情况，例如china的情况，纯ip的链接需要跳过
		x := r.GetUrlSite(szUrl)
		// ip的情况不跳过
		if nil == x {
			return
		}
		oU, err := url.Parse(a1)
		if nil != err {
			a2 := r.FixedUrl(a1, e1)
			oU, err = url.Parse(a2)
			if nil != err {
				log.Println(err, a1)
				return
			} else {
				a1 = a2
			}
		}
		aIps := ipv4Reg.FindAllString(a1, -1)
		var szUrl1 string
		if 0 < len(aIps) {
			szUrl1 = a1
		} else {
			szUrl1 = oBase.ResolveReference(oU).String()
		}
		var mT = map[string]bool{}
		for _, k := range x.aHrefR {
			if _, ok := mT[szUrl1]; ok {
				continue
			}
			if r1 := k.FindStringSubmatch(a1); 0 < len(r1) {
				//r.SetTitle(szUrl, e1.Text)
				// 满足条件的，就继续爬
				szUrl1 = r.FixedUrl(szUrl1, e)
				log.Println("start crapy ", szUrl)
				mT[szUrl1] = true
				r.Visit(szUrl1)
				bRst = true
			}
		}
	})
	return bRst
}
func (r *CrawlerEngine) forEachSite(fnCbk func(site *Site)) {
	for _, s := range r.Site {
		fnCbk(s)
	}
}
func (r *CrawlerEngine) AbsoluteURL(s string) string {
	return s
}

// 图片自动处理
func (r *CrawlerEngine) doImg4Str(s string) string {
	if e, err := goquery.NewDocumentFromReader(strings.NewReader(s)); nil == err {
		e.Find("img").Each(func(i int, e1 *goquery.Selection) {
			if s1, ok := e1.Attr("src"); ok {
				if "" == s1 {
					if s2, ok := e1.Attr("data-original"); ok {
						s1 = s2
					}
				}
				s1 = strings.TrimSpace(s1)
				if "" != s1 {
					if strings.HasPrefix(strings.ToLower(s1), "data:image") {
						return
					}
					if "" != s1 {
						s2 := GetImg2Base64(r.AbsoluteURL(s1), s)
						if "" == s2 {
							s2 = GetImg2Base64(s1, s)
						}
						if "" != s2 {
							r1 := regexp.MustCompile("(?i)<img.*src=\"?" + regexp.QuoteMeta(s1) + "[^>]*>")
							s = string(r1.ReplaceAll([]byte(s), []byte(`<img src="data:image/jpeg;base64,`+s2+`">`)))
						}
					}
				}
			}
		})
	}
	return s
}

// 图片自动处理
func (r *CrawlerEngine) doImg(e *colly.HTMLElement, s string) string {
	e.ForEach("img", func(i int, e1 *colly.HTMLElement) {
		s1 := e1.Attr("src")
		if "" == s1 {
			s1 = e1.Attr("data-original")
		}
		s1 = strings.TrimSpace(s1)
		if strings.HasPrefix(strings.ToLower(s1), "data:image") {
			return
		}
		if "" != s1 {
			s2 := GetImg2Base64(e.Request.AbsoluteURL(s1), e.Request.URL.String())
			if "" == s2 {
				s2 = GetImg2Base64(s1, e.Request.URL.String())
			}
			if "" != s2 {
				r1 := regexp.MustCompile("(?i)<img.*src=\"?" + regexp.QuoteMeta(s1) + "[^>]*>")
				s = string(r1.ReplaceAll([]byte(s), []byte(`<img src="data:image/jpeg;base64,`+s2+`">`)))
			}
		}
	})
	return s
}

// 日期数据的解析
func (r *CrawlerEngine) ParseDate(s string, af []string) *time.Time {
	return ParseDate(s, af)
}

// 域名相同则认为一个规则
//   1、获取url的site配置信息
//   2、判断url是否再允许爬的范围
func (r *CrawlerEngine) GetUrlSite1(a ...any) any {
	var u string = a[0].(string)
	if oU, err := url.Parse(u); nil == err {
		if k := r.GetSite4kv(oU.Host); nil != k {
			return k
		}
		if k := r.GetSite4kv(u); nil != k {
			return k
		}
	}
	if aIps := r.TestIp(u, r.GetUrlSite1); nil != aIps {
		return aIps
	}
	return nil
}

// 基于kvdb后存在一些不好的问题
func (r *CrawlerEngine) GetUrlSite(u string) *Site {
	x1 := r.GetUrlSite1(u)
	if x1 != nil {
		st := r.initSite(x1.(*Site))
		return st
	}
	if u != ChinaWebUrl && (strings.Contains(u, ".gov.cn") || strings.Contains(u, ".edu.cn") || 0 < len(ipReg.FindAllString(u, -1)) || IsChinaUrl(u)) {
		return r.GetUrlSite(ChinaWebUrl)
	}
	return nil
}
func (r *CrawlerEngine) IsStart(u string) bool {
	return r.IsStart1(u).(bool)
}

// 开始的首页不保存
func (r *CrawlerEngine) IsStart1(a ...any) any {
	var u string = a[0].(string)
	var bR = false
	if nil != r.GetSite4kv(u) {
		bR = true
	} else if a3 := r.TestIp(u, r.IsStart1); nil != a3 {
		return a3
	}

	return bR
}

// 获取url的hash
// You will find FNV32, FNV32a, FNV64, FNV64a, MD5, SHA1, SHA256 and SHA512
func (r *CrawlerEngine) GetUrlHash(u string) string {
	return SHA1(u)
}

// 用各Site定义的规则获取title、Body、Date
func (r *CrawlerEngine) doArticleData(e *colly.HTMLElement) {
	site := r.GetSite(e.Request)
	var ad = ArticleData{UrlRaw: e.Request.URL.String()}
	var szT = ""
	if e11, szTxt, ss1 := r.GetHtml(e, site.Body); nil != e11 && "" != ss1 && "" != szTxt {
		ad.Body = ss1
		szT = szTxt
	}
	ad.Body = strings.TrimSpace(ad.Body)
	if "" == ad.Body {
		return
	}
	if "" != site.Title {
		ad.Title = e.ChildText(site.Title)
	} else {
		ad.Title = e.ChildText("Title")
	}
	ad.Title = strings.TrimSpace(ad.Title)
	if ad.Title == "" {
		return
	}
	szDate := ""
	if "" != site.Date {
		szDate = e.ChildAttr(site.Date, "datetime")
		if "" == szDate {
			szDate = e.ChildText(site.Date)
		}
		if "" == szDate {
			szDate = e.ChildAttr(site.Date, "data-original-title")
		}
	} else {
		if szDate = e.Response.Headers.Get("last-modified"); "" == szDate {
			szDate = e.Response.Headers.Get("date")
		}
	}
	id := e.Request.URL.String()
	ad.ID = r.GetReqId(id)
	ad.LastModified = r.ParseDate(e.Response.Headers.Get("last-modified"), site.DateFormat)
	ad.Date = r.ParseDate(strings.TrimSpace(szDate), site.DateFormat)
	ad.ExtData = site.ExtData
	//log.Printf("%+v %s", ad, szT)
	if "" != szT {
		ad.Tags = r.ExtractTags(szT)
		// 因为保存是异步，所以，返回的结果没有意义,改用队列
		r.Doc <- &IndexData{Index: DefaulIndexName, Id: r.GetUrlHash(id), Doc: ad, FnCbk: func() {
			SetCC(id)
		}, FnEnd: func() {}}
		//SaveIndexData(DefaulIndexName, r.GetUrlHash(id), ad, func(i ...any) {
		//	// 一旦保存成功，避免重复
		//	//log.Println("save ok ", id)
		//	util.PutAny[string](id, "1")
		//}, func() {
		//})
	}
}

func (r *CrawlerEngine) GetReqId(u string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(u))
	uHash := h.Sum64()
	return uHash
}

func (r *CrawlerEngine) RmOkSite(i *Site, u string) {
	if oU, err := url.Parse(u); nil == err {
		r.Host2Site.Delete(oU.Host, u)
	}
	i.RmStart(u)
}

func (r *CrawlerEngine) InitSite(i *Site, u string) {
	if oU, err := url.Parse(u); nil == err {
		r.Host2Site.Put(oU.Host, i)
	}
}
func (r *CrawlerEngine) InitSite4Start(i *Site, u string) {
	i.AddStart(u)
	r.Host2Site.Put(u, i)
}
func (r *CrawlerEngine) GetSite(e *colly.Request) *Site {
	if st := r.GetUrlSite(e.URL.String()); nil != st {
		return r.initSite(st)
	}
	if s1 := r.GetSite2(e); nil == s1 {
		return r.GetUrlSite(strings.Join([]string{e.URL.Scheme, "://", e.URL.Hostname()}, ""))
	} else {
		return s1
	}
}

//var muLk sync.RWMutex

// 10次错误就删除
type CC4Url struct {
	UrlStr string        `json:"url_str"`
	ErrCnt int           `json:"err_cnt"`
	Params []interface{} `json:"params"`
}

func (r *CrawlerEngine) GetCvtCCUrl(c []byte) *CC4Url {
	var o interface{}
	var oR *CC4Url
	if nil != json.Unmarshal(c, &oR) {
		json.Unmarshal(c, &o)
	}
	if nil != oR {
		return oR
	}
	if x1, ok := o.(string); ok {
		oR = &CC4Url{UrlStr: x1, ErrCnt: 0}
		r.Host2Site.Put(x1+"_cc01", oR)
	} else if x1, ok := o.(*CC4Url); ok {
		oR = x1
	} else if m, ok := o.(map[string]interface{}); ok {
		oR = &CC4Url{
			UrlStr: fmt.Sprintf("%v", m["url_str"]),
			ErrCnt: int(m["err_cnt"].(float64)),
		}
	}
	return oR
}

func (r *CrawlerEngine) GetCCUrl(c string) *CC4Url {
	var oR *CC4Url
	szK := c + "_cc01"
	r.Host2Site.Get(nil, func(i []byte) {
		oR = r.GetCvtCCUrl(i)
	}, szK)
	return oR
}

// 除了china site，cve、cnnvd都用，保存url
// 以及文件大小、最后的更新日期
func (r *CrawlerEngine) SaveCcUrls(c string, a ...interface{}) {
	szK := c + "_cc01"
	var oR *CC4Url = r.GetCCUrl(c)
	if nil == oR {
		oR = &CC4Url{UrlStr: c, ErrCnt: 0, Params: a}
	} else if oR.ErrCnt >= 10 {
		r.Host2Site.Delete(szK)
		return
	}
	if 0 == len(a) {
		oR.ErrCnt++
	}

	if !r.Host2Site.Put(szK, oR) {
		log.Println("SaveCcUrls not ok")
	}
}

func (r *CrawlerEngine) RmCcUrls(c string) {
	r.Host2Site.Delete(c + "_cc01")
}

// 修复过去 SiteType_China 重复压入的数据，全部rm，合并到
func (r *CrawlerEngine) FixCCKey(k string, m, cnst *Site) bool {
	cnst.Start = append(cnst.Start, m.Start...)
	r.Host2Site.Delete(k, "http://"+k, "https://"+k)
	if strings.Contains(k, "https--") {
		cnst.RmStart(k)
	}
	cnst.NoRptStart()
	return true
}
func (r *CrawlerEngine) Iterator(i iterator.Iterator) bool {
	cnst := r.GetUrlSite(ChinaWebUrl)
	defer func() {
		r.InitSite(cnst, ChinaWebUrl)
		r.InitSite4Start(cnst, ChinaWebUrl)
	}()
	var m = Site{}
	var k string
	json.Unmarshal(i.Key(), &k)
	fmt.Printf("%s\r", k)
	if nil == json.Unmarshal(i.Value(), &m) {
		if m.Type == SiteType_China {
			r.FixCCKey(k, &m, cnst)
			return true
		}
	}
	//fmt.Printf("%s\r", k)
	if strings.HasSuffix(k, "_cc01") {
		oR := r.GetCvtCCUrl(i.Value())
		if nil == oR || "" == oR.UrlStr || 10 <= oR.ErrCnt {
			r.Host2Site.Delete(k)
			return true
		}
		AddDomain(r, oR.UrlStr, cnst, false)
	}

	return true
}

func (r *CrawlerEngine) GetCcUrls() {
	cnst := r.GetUrlSite(ChinaWebUrl)
	//var m = &Site{}
	r.Host2Site.Iterator(func(a ...any) bool {
		k := fmt.Sprintf("%v", a[0])
		//if x, ok := a[1].(*Site); ok {
		//	m = x
		//}
		//fmt.Printf("%s\r", k)
		if strings.HasSuffix(k, "_cc01") {
			data, _ := json.Marshal(k)
			oR := r.GetCvtCCUrl(data)
			if nil == oR || "" == oR.UrlStr || 10 <= oR.ErrCnt {
				r.Host2Site.Delete(k)
				return true
			}
			AddDomain(r, oR.UrlStr, cnst, false)
		}
		return true
	}, nil)
	//r.Host2Site.Iterator(func(i iterator.Iterator) bool {
	//	return r.Iterator(i)
	//}, nil)
	r.InitSite(cnst, ChinaWebUrl)
	r.InitSite4Start(cnst, ChinaWebUrl)
}

// 获取当前 e 的Site
func (r *CrawlerEngine) GetSite2(e *colly.Request) *Site {
	a1 := strings.Split(e.URL.Host, ".")
	a := "." + strings.Join(a1[1:], ".")
	if k1 := r.GetSite4kv(e.URL.Host); nil != k1 {
		return k1
	}
	if k1 := r.GetSite4kv(e.URL.String()); nil != k1 {
		return k1
	}
	if aIps := ipv4Reg.FindAllString(e.URL.String(), -1); 0 < len(aIps) {
		return r.GetUrlSite(ChinaWebUrl)
	}

	var v = &Site{}
	//var st = &Site{}
	// 后半段相同，也认为是相同site
	//r.Host2Site.Iterator(func(i iterator.Iterator) bool {
	//	var k string
	//	json.Unmarshal(i.Key(), &k)
	//	if -1 < strings.Index(k, a) {
	//		json.Unmarshal(i.Value(), st)
	//		v = st
	//		//deepcopier.Copy(value).To(&v)
	//		return false
	//	}
	//	return true
	//}, nil)
	r.Host2Site.Iterator(func(x ...any) bool {
		var k string = x[0].(string)
		if -1 < strings.Index(k, a) {
			v = x[1].(*Site)
			return false
		}
		return true
	}, nil)
	if nil != v || ChinaWebUrl == e.URL.String() {
		return r.initSite(v)
	}
	return nil
}

// 获取html，并自动处理其中的图片，直接嵌入
func (r *CrawlerEngine) GetHtml(e *colly.HTMLElement, selector string) (*colly.HTMLElement, string, string) {
	s := ""
	s11 := ""
	var e11 *colly.HTMLElement
	e.ForEach(selector, func(i int, element *colly.HTMLElement) {
		if s1, err := element.DOM.Html(); nil == err {
			s = strings.TrimSpace(s1)
			e11 = element
			s = r.doImg(e11, s)
			s11 = strings.TrimSpace(element.Text)
		}
	})
	return e11, s11, s
}

// 检查定制规则抽取情况
func (r *CrawlerEngine) CheckCustomizeDate(m map[string]interface{}) bool {
	n := len(m)
	j := 0
	for _, v := range m {
		if "" == fmt.Sprintf("%v", v) {
			j++
		}
	}
	if int(float32(n)*0.9) <= j {
		return false
	}
	return true
}

// 自定义模型处理
//  实现了自定义、动态模型的引擎，可以自定义不同（字段）结构的数据集（库）
func (r *CrawlerEngine) CustomizeParse(e *colly.HTMLElement, site *Site) {
	var mD = map[string]interface{}{}
	var aDate []string
	var aNum []string
	var aBool []string
	for k, v := range site.Customize { // 字段 定义
		if v.Html {
			if e11, szTxt, ss1 := r.GetHtml(e, v.ExtractRule); nil != e11 && "" != ss1 && "" != szTxt {
				mD[k] = ss1
			}
		} else {
			s := strings.TrimSpace(e.ChildText(v.ExtractRule))
			switch v.Type {
			case "date":
				aDate = append(aDate, k)
				mD[k] = s //r.ParseDate(s, site.DateFormat)
			case "number":
				aNum = append(aNum, k)
				mD[k] = s
			case "bool":
				aBool = append(aBool, k)
				mD[k] = s
			default:
				mD[k] = s
			}
		}
	}
	if 0 < len(mD) {
		id := e.Request.URL.String()
		szId := r.GetUrlHash(id)
		mD1 := CvtData(mD, szId, &aDate, &aBool, aNum...)
		if m1, ok := mD1.(map[string]interface{}); ok {
			if !r.CheckCustomizeDate(m1) {
				return
			}
			m1["ext_data"] = site.ExtData
			m1["ip"] = hacker.GetDomian2IpsAll(fmt.Sprintf("%v", m1["site"]))
		}
		if "" == site.IndexName {
			site.IndexName = DefaulIndexName
		}
		r.Doc <- &IndexData{Index: site.IndexName, Id: szId, Doc: mD1, FnCbk: func() {
			r.RmCcUrls(id)
			SetCC(id)
		}}
	}
}

// 判断是否已经处理过
// Visited receives and stores a request ID that is visited by the Collector
func (r *CrawlerEngine) Visited(requestID uint64) error {
	//if nil != Query4Key(DefaulIndexName, fmt.Sprintf("id:%d", requestID)) {
	//	return errors.New(fmt.Sprintf("%d Already exists", requestID))
	//}
	return nil
}

// Cookies retrieves stored cookies for a given host
func (r *CrawlerEngine) Cookies(u *url.URL) string {
	if s := r.GetUrlSite(u.String()); nil != s {
		if nil != s.Headers {
			if c, ok := s.Headers["Cookie"]; ok {
				return c
			}
		}
	}
	return ""
}

// SetCookies stores cookies for a given host
func (r *CrawlerEngine) SetCookies(u *url.URL, cookies string) {
	//if s := r.GetUrlSite(u.String()); nil != s {
	//	hd := s.Headers
	//	if nil != hd {
	//		hd = map[string]string{}
	//	}
	//	hd["Cookie"] = cookies
	//	s.Headers = hd
	//}
}

// IsVisited returns true if the request was visited before IsVisited
// is called
func (r *CrawlerEngine) IsVisited(requestID uint64) (bool, error) {
	if doc := Query4Key(DefaulIndexName, fmt.Sprintf("id:%d", requestID)); nil != doc {
		return true, nil
	}
	return false, nil
}

// Init initializes the storage
func (r *CrawlerEngine) Init() error {
	return nil
}

// 初始化
func (r *CrawlerEngine) init() {
	r.cron = cron.New()
	Options := []colly.CollectorOption{colly.MaxDepth(util.GetValAsInt("MaxDepth", 20000)),
		colly.Async(true),
		colly.ParseHTTPErrorResponse(),
		colly.IgnoreRobotsTxt(),
		colly.MaxBodySize(math.MaxInt),
	}

	if util.GetValAsBool("devDebug") {
		Options = append(Options, colly.CacheDir("./data/cache"))
	}
	r.cc = colly.NewCollector(Options...)
	//r.cc.CheckHead = false
	r.cc.AllowURLRevisit = false // 不允许相同url同时处理
	r.cc.WithTransport(&http.Transport{
		Proxy:             http.ProxyFromEnvironment,
		DisableKeepAlives: true,
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS10, Renegotiation: tls.RenegotiateOnceAsClient},
		//DialContext: (&net.Dialer{
		//	Timeout:   180 * time.Second, // 超时时间
		//	KeepAlive: 120 * time.Second, // keepAlive 超时时间
		//	DualStack: true,
		//}).DialContext,
		//MaxIdleConns:          100,              // 最大空闲连接数
		//IdleConnTimeout:       90 * time.Second, // 空闲连接超时
		//TLSHandshakeTimeout:   10 * time.Second, // TLS 握手超时
		//ExpectContinueTimeout: 1 * time.Second,
	})
	r.cc.DetectCharset = true
	//r.cc.SetStorage(r) // 存储检测
	//r.cc.MaxBodySize = 50 * 1024 * 1024
	//r.cc.IgnoreRobotsTxt = true
	extensions.RandomUserAgent(r.cc)
	extensions.Referer(r.cc)
	// root
	r.cc.OnHTML("html", func(e *colly.HTMLElement) {
		//if strings.Contains(e.Request.URL.String(), "/1?page=2") {
		//	log.Println("debug")
		//}
		// 自定义模型
		st := r.GetSite(e.Request)
		if nil != st {
			if st.Customize != nil {
				r.CustomizeParse(e, st)
				r.DoAHref(e, st)
				return
			}
			switch st.Type {
			case SiteType_Weixin:
				GetWxUrl(r, e.Request.URL.String())
				r.DoAHref(e, st)
			case SiteType_Camera:
				ParseCamera(r, e, st)
				r.DoAHref(e, st)
			case SiteType_DftPswd:
				ParsePswd(r, e, st)
			case SiteType_China:
				go ParseOnHTML(r, e, st) // 中途可能会延时1小时后继续
			case SiteType_CNNVD: // 解析zip -> 下载 -> 解压 ->解析
				ParseCNNVDHref(e, st, r.AsyncVisit)
			case SiteType_Repository:
				ParseMvnrepositoryData(e, st, r)
			case SiteType_Oracle:
				ParseOracle(e, st, r)
				r.DoAHref(e, st)
			default:
				// 继续爬的处理
				//fmt.Printf("OnHtml %s\n", e.Request.URL.String())
				r.DoAHref(e, st)
				if !r.IsStart(e.Request.URL.String()) {
					r.doArticleData(e)
				}
			}
		} else {
			log.Println(e.Request.URL.String())
		}
	})

	// 为复合规则的url添加头信息，例如cookie
	r.cc.OnRequest(func(r1 *colly.Request) {
		s := r1.URL.String()
		st009 := r.GetUrlSite(s)
		if nil == st009 {
			r.log("abort %s site is nil\n", s)
			st009.RmStart(s)
			r1.Abort()
		}
		if r.IsSkip(r1.URL) || r.IsDo4Cache(s) {
			st009.RmStart(s)
			r.log("abort skip %s site is cache\n", s)
			r1.Abort()
		}
		//r1.Headers.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; rv:45.0) Gecko/20100101 Firefox/45.0")

		r.doTsHref(s)

		r.DoHeader(r1)
		u := r1.URL.String()
		//if !r.IsStart(u) {
		if doc := GetAllIndexDoc4Url(u); nil != doc {
			//if r.IsStart(r1.URL.String()) {
			//	DelIndexDoc(DefaulIndexName, GetUrlId(u))
			//	util.Cache1.Delete(u)
			//} else {
			//	log.Println("skip ", doc, u)
			//	r1.Abort()
			//}
			r.log("skip doc is exists %s \r", u)
			st009.RmStart(s)
			r1.Abort()
		}
		//}
		r1.Headers.Set("Accept-Encoding", "gzip")
	})
	// 处理特殊输出，例如 zip
	r.cc.OnResponse(func(res *colly.Response) {
		// application/x-zip-compressed
		//if res.Headers.Get("Content-Type") == "application/x-zip-compressed" {
		//	if xx, err := zlib.NewReader(bytes.NewReader(res.Body)); nil == err {
		//		var a []byte
		//		if n, err := xx.Read(a); nil == err && n > len(res.Body) {
		//			res.Body = a[0:n]
		//		}
		//		xx.Close()
		//	} else {
		//		if xx := flate.NewReader(bytes.NewReader(res.Body)); nil != xx {
		//			var a []byte
		//			if n, err := xx.Read(a); nil == err && n > len(res.Body) {
		//				res.Body = a[0:n]
		//			}
		//			xx.Close()
		//		} else {
		//			log.Println(err, res.Request.URL.String())
		//		}
		//	}
		//}
		//r.doCharset(res)
		r.doJsLocation(res)
		r.DoParseOut(res)
		r.cacheVisit.Cache.Set(res.Request.URL.String(), "", time.Minute)
	})
	// 异常处理，例如记录 到 repeat 库，后续重试，或人工介入分析
	r.cc.OnError(func(r1 *colly.Response, err error) {
		s1 := r1.Request.URL.String()
		//if -1 < strings.Index(s1, "https://security.snyk.io/") {
		//	r.Visit(s1)
		//	return
		//}
		// 记忆失败的url，下次重启时再次处理，或者使用浏览器爬虫
		r.SaveCcUrls(s1)
		// r.AppendFile("errUrls.txt", []string{s1}, nil)
		r.doErrHref(r1.Request.URL)
		fmt.Printf("%d %s %v\n", r1.StatusCode, r1.Request.URL, err)
	})

	// 从配置中加载配置
	r.initConfig()
}

func (r *CrawlerEngine) gbk2utf8(data []byte) []byte {
	reader := transform.NewReader(bytes.NewReader(data), simplifiedchinese.HZGB2312.NewDecoder())
	if d, e := ioutil.ReadAll(reader); nil == e {
		return d
	}
	return data
}

// location="../qdfwzx"
// URL=http://www.xizang.gov.cn/zwgk/xxgk_424/xxgkzn/
var rJsLocation = regexp.MustCompile(`(?:location|URL)(\.href)?\s*=\s*["']?([^"';< >]+)["'> ]`)

// 这里会匹配到许多无效到跳转
func (r *CrawlerEngine) doJsLocation(res *colly.Response) {
	if 200 != res.StatusCode {
		if "" != res.Headers.Get("Location") {
			ss1 := res.Request.AbsoluteURL(res.Headers.Get("Location"))
			fmt.Println("Location ", ss1)
			r.Visit(ss1)
			return
		}
		r.SaveCcUrls(res.Request.URL.String())
		log.Printf("%d %s", res.StatusCode, res.Request.URL.String())
		//r.AppendFile("errUrls.txt", []string{res.Request.URL.String()}, nil)
	}
	if nil != res.Body && 0 < len(res.Body) {
		s := string(res.Body)
		if a := rJsLocation.FindStringSubmatch(s); 0 < len(a) {
			if "ctxPath" == a[2] {
				r.Visit(res.Request.AbsoluteURL("/netface/login.do"))
			} else if "" != a[2] {
				u := res.Request.AbsoluteURL(a[2])
				if "" != u {
					r.Visit(u)
				}
			}
		}
	}
}

func (r *CrawlerEngine) doCharset(res *colly.Response) {
	s1 := res.Headers.Get("Content-Type")
	if strings.Contains(s1, "text/html") {
		// http-equiv="Content-Type" content="text/html; charset=utf-8"
		// <meta http-equiv="Content-Type" content="text/html; charset=gb2312" />
		if a := strings.Split(s1, "="); 1 == len(a) {
			s1 = string(res.Body)
			k1 := "content=\"text/html; charset="
			if n1 := strings.Index(s1, k1); -1 < n1 {
				k1 = s1[n1+len(k1):]
				n1 = strings.Index(k1, "\"")
				if -1 < n1 {
					k1 = k1[0:n1]
					if strings.ToLower(k1) != "utf-8" && 7 > len(k1) {
						res.Body = r.gbk2utf8(res.Body)
					}
				}
			}
		}
	}
}

var rTrimEnd = regexp.MustCompile(`\/[^\/]+$`)

func (r *CrawlerEngine) ReplaceAll(r1 *regexp.Regexp, s, rplc string) string {
	return strings.TrimSpace(string(r1.ReplaceAll([]byte(s), []byte(rplc))))
}

// 错误的时候，尝试去除path后面的，逐级向上访问，便于获取更多信息
func (r *CrawlerEngine) doErrHref(s *url.URL) {
	Scheme := "http"
	if Scheme == "http" {
		Scheme = "https"
	}
	if s.Path == "" || s.Path == "/" {
		r.Visit(strings.Join([]string{Scheme, "://", s.Host, s.Path}, ""))
		return
	}
	szLst := strings.Join([]string{s.Scheme, "://", s.Host, s.Path}, "")
	s1 := r.ReplaceAll(rTrimEnd, s.Path, "")
	// 两种协议都各走一次
	for {
		s2 := strings.Join([]string{s.Scheme, "://", s.Host, s1}, "")
		if s2 != szLst {
			r.Visit(s2)
			szLst = s2
			r.Visit(strings.Join([]string{Scheme, "://", s.Host, s1}, ""))
		}
		if s1 == "" || s1 == "/" {
			return
		}
		s1 = r.ReplaceAll(rTrimEnd, s1, "")
	}
}

// http https都走一边
func (r *CrawlerEngine) doTsHref(s string) {
	if 10 < strings.Index(s, " http") {
		a := strings.Split(s, " ")
		for _, x := range a {
			if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
				r.Visit(x)
			}
		}
	}
}

// 追加到文件中
func (r *CrawlerEngine) AppendFile(szFile string, a []string, f1 *os.File) *os.File {
	if nil == f1 {
		f, err := os.OpenFile(szFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		defer f.Close()
		if err != nil {
			log.Println(szFile, err)
			return f
		}
		f1 = f
	}
	buf := bufio.NewWriter(f1)
	buf.Write([]byte(strings.Join(a, "\n") + "\n"))
	buf.Flush()
	return f1
}
func (r *CrawlerEngine) DoHeader(r1 *colly.Request) {
	s := r.GetSite(r1)
	if nil != s && 0 < len(s.Headers) {
		for _, k := range s.Start {
			if strings.HasPrefix(r1.URL.String(), k) {
				for k1, v1 := range s.Headers {
					r1.Headers.Set(k1, v1)
				}
				if s.RandomDelay {
					time.Sleep(time.Duration(r.GetRandome(1, 5)) * time.Second)
				}
				break
			}
		}
		switch s.Type {
		case SiteType_CVE, SiteType_CNNVD:
			if oCC := r.GetCCUrl(r1.URL.String()); nil != oCC && nil != oCC.Params {
				for _, x := range oCC.Params {
					if m, ok := x.(map[string]interface{}); ok {
						for k, v := range m {
							if "lastModifiedDate" == k {
								r1.Headers.Set(k, fmt.Sprintf("%v", v))
							}
						}
					}
				}
			}

		}
	}
	r1.Headers.Set("Host", r1.URL.Host)

}

// 获取给定范围的随机数
func (r *CrawlerEngine) GetRandome(min, max int) int64 {
	return int64(rand.Intn(max-min) + min)
}

// 解码o转换位site对象
func (r *CrawlerEngine) toSite(o interface{}) *Site {
	var s1 = Site{}
	if data, err := json.Marshal(o); nil == err {
		//log.Printf("%s", string(data))
		json.Unmarshal(data, &s1)
	}
	if "" == s1.IndexName { // 默认值设置
		s1.IndexName = DefaulIndexName
	}
	for _, x := range s1.Start {
		r.InitSite(&s1, x)
		r.InitSite4Start(&s1, x)
	}
	s1.CrawlerEngine = r
	return &s1
}

func (r *CrawlerEngine) initSite(r1 *Site) *Site {
	if nil == r1.aHrefR || 0 == len(r1.aHrefR) {
		for _, k := range r1.A {
			r1.aHrefR = append(r1.aHrefR, regexp.MustCompile(k))
		}
	}
	// 设置默认更新频率，每天
	if "" == r1.Rate {
		r1.Rate = DefaultRate
	}
	r1.CrawlerEngine = r
	return r1
}

// 初始化配置文件
func (r *CrawlerEngine) initConfig() {
	if o := util.GetAsAny("target"); nil != o {
		if a, ok := o.([]interface{}); ok {
			for _, x := range a {
				r.Site = append(r.Site, r.toSite(x))
			}
		}
		// 初始化：编译正则表达式
		r.forEachSite(func(r1 *Site) {
			r.initSite(r1)
			// 更新频率处理
			r.cron.AddFunc(r1.Rate, func() {
				for _, u := range r1.Start {
					r.VisitStart(u)
				}
			})
		})
	}
	if nil == r.Site || 0 == len(r.Site) {
		log.Println("config.json has error")
	}
}

// case map，外部可注册
var CaseMp = map[string]SiteTypeFunc{
	SiteType_RSS:     ParseRSSReader,
	SiteType_CNNVD:   ParseCNNVDZip,
	SiteType_CVE:     ParseZipFile,
	SiteType_POST:    ParsePostRawData,
	SiteType_AsnCode: ParseAsnCsv,
	SiteType_Butian:  ParseJson,
}

// 处理特殊情况下的输出，例如 RSS，将来还可以扩张特殊的数据下载，例如监测到数据、敏感信息泄露，顺便下载
func (r CrawlerEngine) DoParseOut(res *colly.Response) {
	if r.cc != nil {
		if nil == r.GetUrlSite(res.Request.URL.String()) {
			return
		}
		r1 := r.GetSite(res.Request)
		if nil == r1 {
			log.Println(res.Request.URL.String())
			return
		}
		if fnCase, ok := CaseMp[r1.Type]; ok && nil != fnCase {
			fnCase(res, r.Doc, r1, r.AsyncVisit)
		}
	}
}

var fixUrl4hash = regexp.MustCompile("#.*$")

func (r *CrawlerEngine) ClearUrl(u string) string {
	return string(fixUrl4hash.ReplaceAll([]byte(u), []byte("")))
}

func (r *CrawlerEngine) doVisitPool(u string) {
	u = r.ClearUrl(u)
	if b, err := util.GetAny[string](u); nil == err && "1" == b {
		return
	}
	if r.CheckVisit(u, "doVisitPool") {
		return
	}
	// 查到了就不再爬了
	//if strings.Contains(u, "/page/") {
	//DelIndexDoc(DefaulIndexName, GetUrlId(u))
	//}
	bIstart := r.IsStart(u)
	if doc := GetAllIndexDoc4Url(u); doc != nil && !bIstart {
		if m1, ok := util.GetJson4Query(doc, "Fields").(map[string]interface{}); ok {
			if s, ok := m1["title"]; ok && "" == fmt.Sprintf("%v", s) {
				DelIndexDoc(DefaulIndexName, GetUrlId(u))
				util.Cache1.Delete(u)
			} else {
				r.log("Already exists %s %s\n", u, s)
				return
			}
		}
	}
	// 开始的页面不在这里处理，如果获取不到Site配置，就不是同源，跳过处理
	if _, err := url.Parse(u); nil == err {
		if !bIstart && nil != r.GetUrlSite(u) {
			// 避免相同url重复执行
			if r.IsDo4Cache(u) {
				r.log("IsDo4Cache skip ", u)
				return
			}
			r.log("r.Visit  %s\n", u)
			if err := r.cc.Visit(u); nil != err {
				r.log(u, err)
			}
		} else {
			r.log("skip not is start and site is nil", u)
		}
	} else {
		r.log(err)
	}
}

func (r *CrawlerEngine) log(a ...any) {
	s := fmt.Sprintf("%v", a[0])
	if -1 < strings.Index(s, "%") {
		fmt.Printf(s, a[1:]...)
	} else {
		fmt.Println(a...)
	}
}

// 内存缓存，避免重复
func (r *CrawlerEngine) IsDo4Cache(u string) bool {
	if _, ok := r.cacheVisit.Cache.Get(u); ok {
		//r.log("skip IsDo4Cache ", u)
		return true
	}
	return false
}

// 跳过内网
func (r *CrawlerEngine) IsSkip(u *url.URL) bool {
	if skipPrivate && net.ParseIP(u.Hostname()).IsPrivate() {
		return true
	}
	return false
}

func (r *CrawlerEngine) CheckVisit(u, k string) bool {
	if _, ok := r.cacheVisit.Cache.Get(u + k); ok {
		return true
	}
	r.cacheVisit.Cache.Set(u+k, "", 120*time.Second)
	return false
}

// 统一入口，防重处理，起点的url不用走这里
func (r *CrawlerEngine) Visit(u string) {
	if oU, err := url.Parse(u); nil == err {
		if r.IsSkip(oU) || r.IsDo4Cache(u) || r.CheckIsOk(u) {
			return
		}
		if r.CheckVisit(u, "Visit") {
			return
		}
		r.AsyncVisit <- u
	} else {
		log.Println(err)
	}
}

// 处理开始url中年的问题
func (r *CrawlerEngine) VisitStart(u string) {
	if oU, err := url.Parse(u); err != nil || r.IsSkip(oU) || r.IsDo4Cache(u) {
		return
	}
	if 0 < len(Ryear.FindAllString(u, -1)) {
		uB := []byte(u)
		for i := time.Now().Year(); i > 2004; i-- {
			s1 := string(Ryear.ReplaceAll(uB, []byte(fmt.Sprintf("%d", i))))
			if err := r.cc.Visit(s1); nil != err {
				log.Println(s1, err)
			}
		}
	} else {
		//fmt.Printf("VisitStart %s\n", u)
		if err := r.cc.Visit(u); nil != err {
			log.Println(u, err)
		}
	}
}
func (r *CrawlerEngine) runBk() {
	util.DoSyncFunc(func() {
		defer r.Host2Site.Close()
		//tck := time.NewTicker(4 * time.Second)
		//defer tck.Stop()
		for {
			select {
			case <-util.Ctx_global.Done():
				return
			//case <-tck.C:
			//	n3 := len(r.Doc)
			//	if 5000 <= n3 {
			//		SaveIndexDoc4Batch(DefaulIndexName, r.Doc, func() {
			//			fmt.Println("save ok docs ", n3)
			//		}, func() {})
			//	}
			case s, ok := <-r.AsyncVisit:
				if ok && "" != s {
					r.doVisitPool(s)
					// 清缓存
					n1 := len(r.AsyncVisit)
					for ; n1 > 0; n1-- {
						r.doVisitPool(<-r.AsyncVisit)
					}
				}
			case s, ok := <-r.Doc:
				if ok {
					//SaveIndexDoc4Batch(DefaulIndexName, r.Doc, func() {}, func() {})
					var a []*IndexData
					a = append(a, s)
					n1 := len(r.Doc) // 处理缓存区
					for i := 0; i < n1; i++ {
						a = append(a, <-r.Doc)
					}
					for _, x := range a {
						//if nil == GetDoc(x.Index, x.Id) {// 允许更新
						SaveIndexDoc(x.Index, x.Id, x.Doc, x.FnCbk, x.FnEnd)
						//}
					}
				}
			default:

			}
		}
	})
}

// 缓存中识别已经处理过
func (r *CrawlerEngine) CheckIsOk(szT string) bool {
	if s, err := util.GetAny[string](szT); nil == err && "1" == s {
		return true
	}
	return false
}
func (r *CrawlerEngine) SetCC(id string, args ...string) {
	SetCC(id, args...)
}

// 加载缓存
func (r *CrawlerEngine) LoaDb(szT string, r1 *Site) {
	r.GetCcUrls()
	//var al = []DbCc{}
	//if data, err := ioutil.ReadFile("errUrls.txt"); nil == err {
	//	if a := strings.Split(strings.TrimSpace(string(data)), "\n"); 0 < len(a) {
	//		for _, x := range a {
	//			x = strings.TrimSpace(x)
	//			if r.CheckIsOk(x) {
	//				continue
	//			}
	//			r.SaveCcUrls(x)
	//			//AddDomain(r, x, r1, true)
	//		}
	//	}
	//}
	//if x1 := db.GetSubQueryLists[DbCc, DbCc](DbCc{}, "", al, 100000, 0, "type=?", SiteType_China); nil != x1 {
	//	for _, x := range *x1 {
	//		if r.CheckIsOk(x.UrlStr) {
	//			db.Delete[DbCc]("url_str", x.UrlStr, &DbCc{UrlStr: x.UrlStr})
	//			continue
	//		}
	//
	//		if oU, err := url.Parse(x.UrlStr); nil != err || r.IsSkip(oU) {
	//			continue
	//		}
	//		r.SaveCcUrls(x.UrlStr)
	//		//AddDomain(r, x.UrlStr, r1, true)
	//		//log.Println("load from db ", x.UrlStr)
	//		//r1.Start = append(r1.Start, x.UrlStr)
	//	}
	//}
}

// 开始爬
func (r *CrawlerEngine) Start() {
	if r.cc != nil {
		go r.runBk()
		r.forEachSite(func(r1 *Site) {
			util.DoSyncFunc(func() {
				// 加载断点未完成任务
				if r1.Type == SiteType_China {
					go r.LoaDb(SiteType_China, r1)
				}
				for _, u := range r1.Start {
					switch r1.Type {
					case SiteType_China:
						r.VisitStart(u)
					case SiteType_Github:
						DoStartGithub(r, u, r1)
					case SiteType_Butian:
						DoStartPostBt(r, u, r1)
					case SiteType_POST:
						DoStartPost(r, u, r1)
					case SiteType_Shodan:
						QueryShodan(r, u, r1)
					default:
						r.VisitStart(u)
					}
				}
			})
		})
	}
	// Limit the number of threads started by colly to two
	// when visiting links which domains' matches "*httpbin.*" glob
	r.cc.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: util.GetValAsInt("collyThread", 64),
		//Delay:      5 * time.Second,
	})
	//go LoadAllXml(r.Doc, r.GetUrlSite("http://www.cnnvd.org.cn/web/xxk/xmlDown.tag"))
	r.cc.Wait()
}

var skipPrivate = false

func init() {
	util.RegInitFunc(func() {
		os.Setenv("IGNORE_ROBOTSTXT", "true")
		os.Setenv("MAX_BODY_SIZE", fmt.Sprintf("%d", 50*1024*1024))
		skipPrivate = util.GetValAsBool("skipPrivate")
		devDebug = util.GetValAsBool("devDebug")
	})
}
