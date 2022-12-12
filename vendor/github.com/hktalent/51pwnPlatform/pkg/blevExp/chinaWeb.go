package blevExp

import (
	"fmt"
	"github.com/hktalent/colly"
	"gorm.io/gorm"
	"log"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
)

type DbCc struct {
	gorm.Model
	Type   string `json:"type"  gorm:"type:varchar(50);"`
	UrlStr string `json:"url_str" gorm:"unique_index;type:varchar(1000);"`
}

type Field struct {
	Name string `json:"name"` // name or id
	Type string `json:"type"` // 类型
}
type FormInfo struct {
	Action string   `json:"action"`
	Fields []*Field `json:"fields"`
}

type A_herf struct {
	Href  string
	Title string
}

// http://data.chengdu.gov.cn/rc/doc?doc_id=8160157FB3A74DEFBE84A30CAFB9AE8F&signature=uci/gMvzzEqLFU7m2M4M+lOMEMoH6VFl9FhvDH1AqAk=
// 存储数据模型
type ChinaWeb struct {
	Url        string      `json:"url"`
	AChild     []*A_herf   `json:"a_child"`
	Title      string      `json:"title"`
	Css        []string    `json:"css"`
	Js         []string    `json:"js"`
	Img        []string    `json:"img"`
	IsVue      bool        `json:"is_vue"` // 是否未纯js生成
	Forms      []*FormInfo `json:"forms"`
	Emails     []string    `json:"emails"`
	Mobiles    []string    `json:"mobiles"`
	Landline   []string
	Hash       string       `json:"hash"`
	Cert       string       `json:"cert"`        // 证书信息
	CopyRight  string       `json:"copy_right"`  // 网站归属单位,主办、承办 class=footer
	DevSupport string       `json:"dev_support"` // 开发商 维护、技术支持 版权所有
	Header     *http.Header `json:"header"`
	Footer     string       `json:"footer"`
	T3         string       `json:"t3"`
	IIOP       string       `json:"iiop"`
	StatusCode int          `json:"status_code"`
}

var getHrefReg = regexp.MustCompile(`(http[s]?:\/\/[^ <\r\n\"#%,'&]+)`)
var getSupport = regexp.MustCompile(`(?:技术支持|承办|运行维护单位|开发单位)[：:]([^ \r\n\t\s]{5,})\b`)

func DoOneHref(r *CrawlerEngine, e *colly.HTMLElement, doc *ChinaWeb, szCurHref, szTitle string, m01 map[string]bool) {
	szCurHref = e.Request.AbsoluteURL(szCurHref)
	if _, ok := m01[szCurHref]; ok || !IsChinaUrl(szCurHref) {
		log.Println("skip not is china", szCurHref)
		return
	}
	// 没有 http开头的，本站的，特殊登陆的情况
	sLcs := strings.ToLower(szCurHref)
	if strings.Contains(sLcs, "login") || strings.Contains(sLcs, ".jsp") || strings.Contains(sLcs, "Action") || strings.Contains(sLcs, ".do") || strings.HasSuffix(szCurHref, "/") && szCurHref != e.Request.URL.String() {
		r1 := r.GetUrlSite(e.Request.URL.String())
		if nil == r1 {
			r1 = r.GetUrlSite(ChinaWebUrl)
		}
		AddDomain(r, szCurHref, r1, false)
		r.Visit(szCurHref)
		return
	}
	// 不添加相同域名，不同的域名才处理
	if strings.Contains(szCurHref, "://"+e.Request.URL.Host) {
		return
	}
	m01[szCurHref] = true
	var a1 = &A_herf{Title: szTitle, Href: szCurHref}

	if "" != a1.Href && -1 == strings.Index(a1.Href, "javascript") {
		doc.AChild = append(doc.AChild, a1)
	}
}

// 获取所有连接，包含 html 中
func GetAllA(r *CrawlerEngine, e *colly.HTMLElement, doc *ChinaWeb) {
	var m01 = map[string]bool{}
	e.ForEach("a,iframe", func(i int, e1 *colly.HTMLElement) {
		szCurHref := strings.TrimSpace(e1.Attr("href"))
		if "" == szCurHref {
			szCurHref = strings.TrimSpace(e1.Attr("src"))
		}
		DoOneHref(r, e, doc, szCurHref, strings.TrimSpace(e1.Text), m01)
	})
	if s, err := e.DOM.Html(); nil == err {
		if a := getHrefReg.FindAllString(s, -1); 0 < len(a) {
			for _, x := range a {
				DoOneHref(r, e, doc, x, "", m01)
			}
		}
	}
}
func GetPaths(e *colly.HTMLElement, tag string) []string {
	var a []string
	e.ForEach(tag, func(i int, e1 *colly.HTMLElement) {
		if s := e1.Attr("href"); "" == s {
			s = e1.Attr("src")
			if -1 == strings.Index(s, "base64:") {
				a = append(a, s)
			}
		}
	})
	return a
}

func GetFild(e *colly.HTMLElement, tag string) []*Field {
	var a []*Field
	e.ForEach(tag, func(i int, e1 *colly.HTMLElement) {
		x1 := &Field{Name: e1.Attr("name"), Type: e1.Attr("type")}
		if x1.Name == "" {
			x1.Name = e1.Attr("id")
		}
		if x1.Type == "" {
			x1.Type = "text"
		}
		a = append(a, x1)
	})
	return a
}

func GetForms(e *colly.HTMLElement) []*FormInfo {
	var a []*FormInfo
	e.ForEach("form", func(i int, e1 *colly.HTMLElement) {
		var x1 = &FormInfo{Action: e1.Attr("action")}
		x1.Fields = append(GetFild(e1, "input"), GetFild(e1, "textarea")...)
		x1.Fields = append(x1.Fields, GetFild(e1, "button")...)

		if 0 < len(x1.Fields) {
			a = append(a, x1)
		}
	})
	return a
}

// 添加到允许范围,bAdd = false表示未添加
func AddDomain(r *CrawlerEngine, u string, r1 *Site, bAdd bool) bool {
	bRst := false
	n := 0
	if !bAdd {
		for _, x := range r1.Start {
			if !strings.HasPrefix(u, x) {
				n++
			}
		}
	}
	//if "http://en.moe.gov.cn" == u {
	//	log.Println(u)
	//}
	bA1 := n == len(r1.Start)
	st01 := r.GetUrlSite(u)
	if strings.HasPrefix(u, "http") && bA1 && nil != st01 {
		//util.DoSyncFunc(func() {
		//	if 0 == db.Create[DbCc](&DbCc{UrlStr: u, Type: SiteType_China}) {
		//		log.Println("save fail")
		//	}
		//})
		//if strings.Contains(u, "gov.cn") || strings.Contains(u, "edu.cn") || 0 < len(ipReg.FindAllString(u, -1)) {
		{
			if oU, err := url.Parse(u); nil == err {
				// 重新格式化u
				u1 := strings.Join([]string{oU.Scheme, "://", oU.Host, oU.Path}, "")
				if u1 != u {
					log.Println(u, u1)
				}
				r.SaveCcUrls(u1)
				r1.Start = append(r1.Start, u1)
				r.InitSite(r1, ChinaWebUrl)
				r.InitSite4Start(r1, ChinaWebUrl)
				go r.VisitStart(u1)
				bRst = true
			} else {
				log.Println(u, "err")
			}
		}
	} else {
		log.Println("not add ", u)
	}
	return bRst
}

var mcReg sync.Map

// 检测是否包含 ip
var ipReg = regexp.MustCompile(`(\d{1,3}\.){3}(\d{1,3})`)
var ipCl = regexp.MustCompile(`[\d\.]`)

func GetInfo(r1 string, e *colly.HTMLElement) []string {
	s := string(e.Text)
	var r01 *regexp.Regexp
	if r, ok := mcReg.Load(r1); ok {
		r01, _ = r.(*regexp.Regexp)
	} else {
		r01 = regexp.MustCompile(r1)
		mcReg.Store(r1, r01)
	}
	return r01.FindAllString(s, -1)
}

/*
katana -d 10000 -jc -kf all -hl -system-chrome  -headless-options '--blink-settings="imagesEnabled=false",--enable-quic="imagesEnabled=false"'  -u "https://waimai.meituan.com" -o meituan.json -j -nc
*/
// start r.cc.OnHTML
//  1-cache url，避免重复
func ParseOnHTML(r *CrawlerEngine, e *colly.HTMLElement, r1 *Site) {
	szUrl := e.Request.URL.String()
	n01 := int(math.Min(float64(len(szUrl)), 150))
	log.Printf("%s   \r", szUrl[0:n01])
	r.SaveCcUrls(szUrl)
	szHtml := ""
	if sh, err := e.DOM.Html(); nil == err {
		szHtml = sh
	}
	var doc = &ChinaWeb{
		Url:        szUrl,
		Footer:     strings.TrimSpace(e.ChildText(".footer,footer")),
		Header:     e.Response.Headers,
		DevSupport: strings.Join(getSupport.FindAllString(szHtml, -1), " "),
		AChild:     []*A_herf{},
		Title:      strings.TrimSpace(e.ChildText("title")),
		Css:        append(GetPaths(e, "link"), GetPaths(e, "style")...),
		Js:         GetPaths(e, "script"), // js 路径信息
		Img:        GetPaths(e, "img"),    //  图片路径信息
		Forms:      GetForms(e),           // form 表单提取
		Mobiles:    GetInfo(r1.Mobile, e),
		Emails:     GetInfo(r1.Email, e),
		Landline:   GetInfo(r1.Landline, e),
		StatusCode: e.Response.StatusCode,
	}
	for _, k := range []string{"Content-Type", "Content-Encoding", "Connection", "Cache-Control", "Expires", "Content-Length"} {
		delete(*doc.Header, k)
	}

	GetAllA(r, e, doc)
	doc.IsVue = 0 == len(doc.Img) && 0 == len(doc.AChild)

	id := GetUrlId(szUrl)
	//if strings.Contains(doc.Title, "MIME") {
	//	//log.Println(e.Response.Headers)
	//}
	if "" != doc.Title {
		r.Doc <- &IndexData{
			Index: DefaulIndexName,
			Id:    id,
			Doc:   doc,
			FnCbk: func() {
				r.RmCcUrls(szUrl)
				fmt.Printf("is ok %s %s\n", szUrl, doc.Title)
				SetCC(szUrl)
				r.RmOkSite(r1, szUrl)
			},
			FnEnd: func() {},
		}
	} else {
		r.doErrHref(e.Request.URL)
	}
	for _, x := range doc.AChild {
		// 必须是http开头的协议，不是起点 url ，添加到 允许范围
		//isStart := r.IsStart(x.Href)
		adStart := AddDomain(r, x.Href, r1, false)
		if strings.HasPrefix(x.Href, "http") && adStart {
			//r.VisitStart(x.Href) // 这里如果走start，就会出现很多重复，好在有缓存，从缓存中获取
			r.Visit(x.Href)
			//if strings.Contains(x.Href, "gov.cn") || strings.Contains(x.Href, "edu.cn") || 0 < len(ipReg.FindAllString(x.Href, -1)) {
			//	r.Visit(x.Href)
			//} else {
			//	log.Println(x.Href)
			//}
		} else if !strings.HasPrefix(x.Href, "http") && strings.Contains(strings.ToLower(x.Href), "login") {
			r.Visit(x.Href)
		} else {
			//fmt.Println(adStart, x.Href)
		}
	}
}

/*
,"http://0734abc.com/pc/", "http://1.85.55.147:8002/auth_front/", "http://103.203.218.250:8041/scrc/", "http://113.140.18.123/xawt/", "http://113.140.18.123:8100/wsfw/", "http://113.57.190.150:8088/rcfw/", "http://114.255.111.115/bjwtqt/", "http://118.112.188.108:8602/misyh/", "http://118.112.188.108:9700/console/", "http://118.122.124.71:6099/fdams/", "http://118.122.251.6:8088/gzqw/", "http://118.122.8.171:8003/lsui/", "http://119.6.50.201:9999/surfane/", "http://125.64.60.11:8000/gywssb/", "http://125.69.70.49:8088/scmonitor/", "http://182.150.40.151/fwwd/", "http://183.203.216.206:8800/tywscx/", "http://183.203.216.206:8801/tyxsjc/", "http://218.6.145.141:7000/yhcms/", "http://219.138.246.147/tmwt/", "http://219.140.166.16:39080/hbjzjz/", "http://222.169.170.135:7003/yhwtqt/", "http://222.82.215.217:10339/ssologin/", "http://222.83.228.143:7003/ggapp/", "http://222.85.128.66:8007/ssologin/", "http://59.175.218.199:8080/hbjzaz/", "http://59.175.218.202:7005/hbwssb/", "http://59.175.218.203:8081/zcsb/", "http://59.175.230.248:8009/whylxt/", "http://59.175.230.248:8011/whjzjz/", "http://61.177.252.165/netface/", "http://bhzfgjj.gjj.beihai.gov.cn/netface/", "http://czt.sc.gov.cn/hmhn/", "http://gjjwt.jiyuan.gov.cn/netface/", "http://gjjwt.tianshui.gov.cn/netface/", "http://gryh.ycgjj.com/ycwt/", "http://gzzzfgjj.com/gznt/", "http://hgwsbs.hg12333.com/hgwt/", "http://jtt.hubei.gov.cn/zyzgzx/", "http://jyb.hrss.tj.gov.cn:7081/tjsyapp/", "http://ncgjj.nc.gov.cn/nczfgjj/", "http://rswb.gx12333.net/member/", "http://sfrz.xzzwfw.gov.cn:8081/sfrz/", "http://wsdt.ybj.gxzf.gov.cn/member/", "http://wt.gjj.shangqiu.gov.cn/netface/", "http://wt.rsj.taiyuan.gov.cn/service/", "http://wt.zfgjj.xuchang.gov.cn/netface/", "http://www.gdsi.gov.cn:7003/gyww/", "http://www.gxwzgjj.gov.cn/netface/", "http://www.hbxn.hrss.gov.cn/xnrswt/", "http://www.lyzfgjj.com:7011/surtalk/", "http://www.msgjj.com/netface/", "http://www.pzhgjj.com/pzhnt/", "http://www.sc.hrss.gov.cn/scsbwt/", "http://www.scdy.lss.gov.cn/ta3admin/", "http://www.scdy.lss.gov.cn:8000/SBK/", "http://www.scgy.lss.gov.cn:8000/gywssb/", "http://www.snhrm.com/member/", "http://www.xygjj.cn:8890/surfane/", "http://www.yinhaiyun.com/casserver/", "http://www.yinhaiyun.com/pc/", "http://www.yjzfgjj.cn/netface/", "http://wx.sxty12320.com/tywscx/", "http://wx.tlgjjwx.com:8005/netface/", "http://xzgjjapp.sxxz.gov.cn/netface/", "http://zfgjjwt.guilin.gov.cn/netface/", "https://61.185.220.174:7316/userinfo/", "https://aabbccd.com/gjj/", "https://asgjj.anshun.gov.cn:8097/netface/", "https://corp-office.cqgjj.cn/netface/", "https://erp.yinhai.com:8443/portal/", "https://es.cdhrss.chengdu.gov.cn:442/cdwsjb/", "https://fzsylfw.com/psp/", "https://gjj.dt.gov.cn/netface/", "https://gjj.dt.gov.cn:10/netface/", "https://gjj.leshan.gov.cn/netface/", "https://gjjwt.zhanjiang.gov.cn:1443/netface/", "https://gjjwt.zhanjiang.gov.cn:5443/surtalk/", "https://jzt.xzggjyzpw.com/app/", "https://m.duxue.org/a/", "https://network.fzgjj.com.cn/netface/", "https://odzfb.cdhrss.gov.cn:7777/cbc_server/", "https://odzfb.cdhrss.gov.cn:8443/yhjypt/", "https://old.jlgjj.gov.cn:7009/netface/", "https://person-office.cqgjj.cn:9443/netface/", "https://weixin.ncgjj.com.cn:7980/netface/", "https://weixin.ncgjj.com.cn:7990/netface/", "https://wt.gjj.shangqiu.gov.cn/netface/", "https://wt.scsjgjj.com/netface/", "https://wt.smxgjj.com/netface/", "https://www.cqgjj.cn/info/", "https://www.csgjj.com.cn/netface/", "https://www.fokyl.com/news/", "https://www.gzxinde.com/a/", "https://www.jsdfpsj.com/xzgjj_/", "https://www.jzdyb.com/log/", "https://www.msgjj.com/netface/", "https://www.msha3.com/news/", "https://www.tdedb.com/blog/", "https://www.xuankw.com/info/", "https://www.xxgjj.com:8011/netface/", "https://www.yaszfgjjzx.org.cn/netface/", "https://yinhai.webex.com.cn/yinhai/", "https://ywdt.xygjj.cn/netface/", "https://zfgjj.leshan.gov.cn/netface/","https://ndgtfy.fjcourt.gov.cn/index.shtml",",http://nnsa.mee.gov.cn/", "http://nrra.gov.cn/", "http://www.ah.gov.cn/", "http://www.audit.gov.cn/", "http://www.beian.gov.cn/portal/registerSystemInfo?recordcode=11010202000001", "http://www.beijing.gov.cn/", "http://www.caac.gov.cn/index.html", "http://www.cac.gov.cn/", "http://www.caea.gov.cn/", "http://www.cbirc.gov.cn/cn/view/pages/index/index.html", "http://www.ccdi.gov.cn/", "http://www.ccps.gov.cn/", "http://www.chinatax.gov.cn/", "http://www.cidca.gov.cn/", "http://www.cma.gov.cn/", "http://www.cnca.gov.cn/", "http://www.cnipa.gov.cn", "http://www.cnsa.gov.cn/", "http://www.counsellor.gov.cn/", "http://www.court.gov.cn/", "http://www.cppcc.gov.cn/", "http://www.cq.gov.cn/", "http://www.csrc.gov.cn/pub/newsite/", "http://www.customs.gov.cn/", "http://www.drc.gov.cn/", "http://www.fmprc.gov.cn/web/", "http://www.fmprc.gov.cn/web/zwjg_674741/zwsg_674743/yz_674745/", "http://www.fmprc.gov.cn/web/zwjg_674741/zwtc_674771/", "http://www.fmprc.gov.cn/web/zwjg_674741/zwzlg_674757/yz_674759/", "http://www.forestry.gov.cn/", "http://www.fujian.gov.cn", "http://www.gansu.gov.cn/", "http://www.gd.gov.cn/", "http://www.ggj.gov.cn/", "http://www.gjbmj.gov.cn", "http://www.gjxfj.gov.cn/", "http://www.gqb.gov.cn/", "http://www.guizhou.gov.cn/", "http://www.gwytb.gov.cn/", "http://www.gxzf.gov.cn/", "http://www.hainan.gov.cn/", "http://www.hebei.gov.cn/", "http://www.henan.gov.cn/", "http://www.hlj.gov.cn", "http://www.hmo.gov.cn/", "http://www.hubei.gov.cn/", "http://www.hunan.gov.cn/", "http://www.jiangsu.gov.cn/", "http://www.jiangxi.gov.cn", "http://www.jl.gov.cn", "http://www.ln.gov.cn", "http://www.locpg.gov.cn/", "http://www.lswz.gov.cn/", "http://www.mca.gov.cn/", "http://www.mct.gov.cn/", "http://www.mee.gov.cn/", "http://www.mem.gov.cn", "http://www.miit.gov.cn/", "http://www.mnr.gov.cn/", "http://www.moa.gov.cn/", "http://www.moa.gov.cn/ztzl/szcpxx/", "http://www.mod.gov.cn/", "http://www.moe.gov.cn/", "http://www.moe.gov.cn/jyb_sy/China_Language/", "http://www.mof.gov.cn/index.htm", "http://www.mofcom.gov.cn/", "http://www.mohrss.gov.cn/", "http://www.mohurd.gov.cn/", "http://www.moj.gov.cn/", "http://www.most.gov.cn/", "http://www.mot.gov.cn/", "http://www.mps.gov.cn/", "http://www.mva.gov.cn/", "http://www.mwr.gov.cn/", "http://www.ncac.gov.cn/", "http://www.ncha.gov.cn/", "http://www.ndrc.gov.cn/", "http://www.nea.gov.cn/", "http://www.neac.gov.cn", "http://www.nhc.gov.cn/", "http://www.nhsa.gov.cn/", "http://www.nmg.gov.cn/", "http://www.npc.gov.cn/", "http://www.nra.gov.cn/", "http://www.nrta.gov.cn/", "http://www.nx.gov.cn/", "http://www.oscca.gov.cn/", "http://www.pbc.gov.cn/", "http://www.qinghai.gov.cn/", "http://www.saac.gov.cn/", "http://www.sac.gov.cn/", "http://www.safe.gov.cn/", "http://www.samr.gov.cn/", "http://www.sara.gov.cn/gjzjswjhtml/index.html", "http://www.sasac.gov.cn/", "http://www.sastind.gov.cn/", "http://www.satcm.gov.cn/", "http://www.sc.gov.cn", "http://www.scio.gov.cn/index.htm", "http://www.scs.gov.cn/", "http://www.shaanxi.gov.cn/", "http://www.shandong.gov.cn/", "http://www.shanghai.gov.cn/", "http://www.shanxi.gov.cn/", "http://www.spb.gov.cn/", "http://www.sport.gov.cn/", "http://www.spp.gov.cn/", "http://www.stats.gov.cn/", "http://www.tj.gov.cn/", "http://www.tobacco.gov.cn/html/", "http://www.xinjiang.gov.cn/", "http://www.xizang.gov.cn/", "http://www.xjbt.gov.cn/", "http://www.yn.gov.cn/", "http://www.zj.gov.cn/", "http://www.zlb.gov.cn/", "https://beian.miit.gov.cn/#/Integrated/index", "https://www.chinamine-safety.gov.cn/", "https://www.nia.gov.cn", "https://www.nmpa.gov.cn/","http://hngy.hunancourt.gov.cn/index.shtml"
*/
