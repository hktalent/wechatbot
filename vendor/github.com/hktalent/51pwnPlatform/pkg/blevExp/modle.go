package blevExp

import (
	"github.com/hktalent/colly"
	util "github.com/hktalent/go-utils"
	"regexp"
	"strings"
	"time"
)

type QueryResult struct {
	ID     string                 `json:"id"`
	Fields map[string]interface{} `json:"fields"`
}

// 自定义字段
type Fields struct {
	Des         string `json:"des"`          // 字段中文名，可选
	ExtractRule string `json:"extract_rule"` // 提取选择器
	Type        string `json:"type"`         // date,number，bool
	BitSize     int    `json:"bit_size"`     // float、int的位数，默认16位
	Html        bool   `json:"html"`         // 保存html
}

// 单个用户及项目列表，例如 apache
type GitProjects struct {
	User     string   `json:"user"`
	Projects []string `json:"projects"`
}

// 站配置
type Site struct {
	Des             string            `json:"des"`               // 规则标题、说明
	IndexName       string            `json:"index_name"`        // 默认 osint
	Start           []string          `json:"start"`             // 开始页面
	A               []string          `json:"a"`                 // 需要爬的url正则规则
	Body            string            `json:"Body"`              // Body 图片自动转base64
	Title           string            `json:"Title"`             // 首尾trim 提取规则
	Email           string            `json:"email"`             // 收集email
	Mobile          string            `json:"mobile"`            // 手机
	Landline        string            `json:"landline"`          // 座机电话
	Date            string            `json:"Date"`              // 自动统一格式
	GitProjects     []*GitProjects    `json:"git_projects"`      // github 定义
	Authorization   []string          `json:"authorization"`     // header中的认证信息
	Rate            string            `json:"rate"`              // 更新频率, Seconds Minutes Hours Day-of-Month Month Day-of-Week Year (optional field)
	SaveCheckTags   bool              `json:"save_check_tags"`   // 检查，有tags才保存，例如github，没有识别到敏感关键词，不保存
	DateFormat      []string          `json:"date_format"`       // 日期格式
	DateExtractRule string            `json:"date_extract_rule"` // 日期提取器
	PostData        string            `json:"post_data"`         // post data
	SearchKeys      []string          `json:"search_keys"`       // 搜索的key
	Headers         map[string]string `json:"headers"`           // cookie、特殊头
	IsOverseas      bool              `json:"is_overseas"`       // 境外
	RandomDelay     bool              `json:"random_delay"`      // 随机延时 5 ～ 15 s
	PostParmConfig  *PostParmConfig   `json:"post_parm_config"`  // 参数定义
	Type            string            `json:"type"`              // 设置特定类型
	Customize       map[string]Fields `json:"customize"`         // 自定义模型
	JsonPath        string            `json:"json_path"`
	ExtData         map[string]string `json:"ext_data"` // 附加数据
	aHrefR          []*regexp.Regexp
	CrawlerEngine   *CrawlerEngine
}

func (r *Site) RmStart(u string) {
	for i, x := range r.Start {
		if x == u {
			if 0 < len(r.Start) && i+1 < len(r.Start) {
				r.Start = append(r.Start[0:i], r.Start[i+1:]...)
			}
			r.CrawlerEngine.Host2Site.Put(r.Start[0], r)
			return
		}
	}
}

func (r *Site) AddStart(u string) {
	n := 0
	for _, x := range r.Start {
		if x != u {
			n++
		}
	}
	if n == len(r.Start) {
		r.Start = append(r.Start, u)
		r.CrawlerEngine.Host2Site.Put(r.Start[0], r)
	}
}

func (r *Site) NoRptStart() {
	r.Start = util.RemoveDuplication_map(r.Start)
	r.CrawlerEngine.Host2Site.Put(r.Start[0], r)
}

type SiteTypeFunc func(res *colly.Response, doc chan *IndexData, r1 *Site, reTry chan string)

const (
	SiteType_Github = "github"
	SiteType_RSS    = "rss"
	SiteType_CNNVD  = "cnnvd"
	SiteType_CVE    = "cve"
	SiteType_POST   = "postraw"
	SiteType_Shodan = "shodan"

	SiteType_Repository = "repository"
	SiteType_China      = "china"
	SiteType_AsnCode    = "asn"
	SiteType_Oracle     = "oracle"
	SiteType_Butian     = "butian"
	SiteType_Camera     = "camera"
	SiteType_DftPswd    = "dfpswd"
	SiteType_Weixin     = "weixin"
)

// 文章
type ArticleData struct {
	Title        string            `json:"title"`
	Body         string            `json:"body"`
	Date         *time.Time        `json:"date"`
	UrlRaw       string            `json:"url-raw"`
	LastModified *time.Time        `json:"last_modified"`
	Tags         string            `json:"tags"`
	ID           uint64            `json:"id"`       // 请求的id
	ExtData      map[string]string `json:"ext_data"` // 附加数据
}

// post参数定义
type PostParmConfig struct {
	Keys    map[string][]string `json:"keys"`     // 关键词替换，多个循环
	NumKeys map[string][]int    `json:"num_keys"` // 数字 替换，第一个是开始，第二个结束，第三个是步长
}

// 默认的更新周期
var DefaultRate = "0 0 0 1 * * *"
var AllIndex = strings.Split("osint,cnnvd", ",")
var DefaulIndexName = "osint"
var FixUrl = regexp.MustCompile(`(^http://\s*)|([\\r\\n]])`)
var Fix4Snyk = regexp.MustCompile(`.io/.*?/vuln/`)
var ipv4Reg = regexp.MustCompile(`^http[s]?:\/\/(\d{1,3}\.){3}\d{1,3}`)

var ChinaWebUrl = "https://www.gov.cn/"
