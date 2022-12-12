package blevExp

import (
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

/*
github公开信息，具备基本的poc
https://securitylab.github.com/advisories/
墨菲信息
https://www.zerodayinitiative.com/advisories/upcoming/
https://www.zerodayinitiative.com/advisories/published/

    {
      "start": ["https://seclists.org/rss/fulldisclosure.rss"],
      "title": "#nst-content > h1",
      "body": "#nst-content > pre",
      "date": "//*[@id=\"nst-content\"]/text()[2]",
      "date_format": ["Mon, 02 Jan 2006 15:04:05 -0300","Mon, 02 Jan 2006 15:04:05 MST"],
      "is_overseas":true,
      "type": "rss",
      "headers": {}
    },

 {
      "start": ["https://sploitus.com/search"],
      "title": "#search-results > div.accordion > label > div.tile-content > h1",
      "body": "#code-FEA5EE0D-5909-558F-8947-9AE371C51A08",
      "date": "//*[@id=\"search-results\"]/div[1]/label/div[2]/div",
      "date_extract_rule":"(\\d{4}-\\d{2}-\\d{2})",
      "post_data": "{\"type\":\"exploits\",\"sort\":\"default\",\"query\":\"poc\",\"title\":false,\"offset\":10}",
      "date_format": ["2006-01-02","Mon, 02 Jan 2006 15:04:05 MST"],
      "is_overseas":true,
      "type": "rss",
      "headers": {
        "Referer": "https://sploitus.com/?query=poc",
        "Origin": "https://sploitus.com",
        "Accept": "application/json",
        "Content-Type": "application/json",
        "User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.1 Safari/605.1.15",
      }
    },

{
      "start": ["https://sploitus.com/rss"],
      "title": "#search-results > div.accordion > label > div.tile-content > h1",
      "body": "#code-FEA5EE0D-5909-558F-8947-9AE371C51A08",
      "date": "//*[@id=\"search-results\"]/div[1]/label/div[2]/div",
      "date_extract_rule":"(\\d{4}-\\d{2}-\\d{2})",
      "date_format": ["2006-01-02","Mon, 02 Jan 2006 15:04:05 MST"],
      "is_overseas":true,
      "type": "rss",
      "headers": {}
    },


https://github.com/mempool/mempool/commits/master
/[^\/]+/[^\/]+/commit/[a-f0-9]{40}$

#repo-content-pjax-container > div > div.commit.full-commit.mt-0.px-2.pt-2 > div.commit-title.markdown-title
{
      "start": ["https://github.com/centreon/centreon"],
      "type": "pull&issues",
      "body": "/html/body/div[4]/div/main/turbo-frame/div/div[2]/div[3]/div/div[1]/div/div[1]/div[1]/div[2]/div/div[2]/div",
      "title": "#partial-discussion-header > div.gh-header-show > div > h1 > span.js-issue-title.markdown-title",
      "date": "/html/body/div[4]/div/main/turbo-frame/div/div[2]/div[3]/div/div[1]/div/div[1]/div[1]/div[2]/div/div[1]/h3/a/relative-time",
      "date_format": ["2006-01-02T15:04:05Z07:00","Mon, 02 Jan 2006 15:04:05 MST"],
      "is_overseas":true,
      "random_delay": true,
      "a": ["\\/[^\\/]+\\/[^\\/]+\\/(pull|issues)\\/\\d+$","\\/[^\\/]+\\/[^\\/]+\\/(issues|pulls)\\?page=\\d{1,}&q=.*"]
    },
*/

// 匿名 每个ip 每小时 可以访问60次
// 加上认证信息，每个账号每小时可以访问5000次
const (
	DefaultGithubApiUrl = "https://api.github.com/repos/"
)

// 更新github key 计数器
var nPosHd = 0

// 自动切换认证信息
func GetHeader4Git(r1 *Site) *map[string]string {
	if nPosHd >= len(r1.Authorization) {
		nPosHd = 0
	}
	szCurlKey := r1.Authorization[nPosHd]
	if 1 >= X_Ratelimit_Remaining {
		nPosHd++
	}
	// 延时1小时后再继续
	if nPosHd > len(r1.Authorization) {
		time.Sleep(3600 * time.Second)
	}
	return &map[string]string{"Authorization": "Bearer " + szCurlKey}
}

var X_Ratelimit_Remaining int64 = math.MaxInt64

/*/ 更新限制信息
< x-ratelimit-limit: 5000
< x-ratelimit-remaining: 4999
< x-ratelimit-reset: 1666682825
< x-ratelimit-used: 1
//////////////////////////*/
func UpLimit(r1 *http.Response) {
	if s := r1.Header.Get("X-Ratelimit-Remaining"); "" != s {
		if n, err := strconv.ParseInt(s, 10, 32); nil == err {
			if n < X_Ratelimit_Remaining {
				atomic.StoreInt64(&X_Ratelimit_Remaining, n)
			}
		}
	}
	//log.Println(r1.Header)
}

func GetObjFromUrl(szUrl string, r1 *Site, r *CrawlerEngine) interface{} {
	if r1, err := DoGet(szUrl, *GetHeader4Git(r1)); nil == err {
		UpLimit(r1)
		var m1 = []map[string]interface{}{}
		if data, err := ioutil.ReadAll(r1.Body); nil == err {
			if err := json.Unmarshal(data, &m1); nil == err {
				return m1
			} else {
				var m1 = map[string]interface{}{}
				if err := json.Unmarshal(data, &m1); nil == err {
					return m1
				}
				if nil != data {
					log.Println(string(data))
				}
				log.Println(err)
			}
		}
	} else {
		log.Println("DoGet ", err)
	}
	return nil
}
func GetDocFromUrl(szUrl, indexName string, r1 *Site, r *CrawlerEngine) interface{} {
	if m1 := GetObjFromUrl(szUrl, r1, r); nil != m1 {
		r.Doc <- &IndexData{Doc: m1, Id: GetUrlId(szUrl), Index: indexName}
		return m1
	}
	return nil
}

// 获取项目的 所有分支
//  每个项目的分支信息只取一次
// curl -H "Authorization: Bearer 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" https://api.github.com/repos/apache/logging-log4j2/branches
func GetBranches(user, project string, r1 *Site, r *CrawlerEngine) interface{} {
	szUrl := DefaultGithubApiUrl + user + "/" + project + "/branches"
	return GetObjFromUrl(szUrl, r1, r)
}

// 获取提交信息
func GetCommitDoc(u string, r *CrawlerEngine, r1 *Site) interface{} {
	id := GetUrlId(u)
	if doc := GetDoc(r1.IndexName, id); nil != doc {
		return doc
	}
	return GetObjFromUrl(u, r1, r)
}

// 遍历每个节点进行处理
/*
{
    "name": "LOG4J2-609",
    "commit": {
      "sha": "dead971fd0e92615209e0084d237b145407663d8",
      "url": "https://api.github.com/repos/apache/logging-log4j2/commits/dead971fd0e92615209e0084d237b145407663d8"
    },
    "protected": false
  },
*/
func EacheBranchesItem(o interface{}, r *CrawlerEngine, r1 *Site) {
	if nil != o {
		if a, ok := o.([]map[string]interface{}); ok {
			for _, x := range a {
				if szUrl := fmt.Sprintf("%v", GetJson4Query(x, ".commit.url")); "" != szUrl {
					if cmd := GetCommitDoc(szUrl, r, r1); nil != cmd {
						if m3, ok := cmd.(map[string]interface{}); ok {
							DoOneCommit(m3, r, r1)
						}
					}
				}
			}
		}
	}
}

func SendDoc2Buf(r *CrawlerEngine, id, indexName string, doc interface{}) {
	r.Doc <- &IndexData{
		Index: indexName,
		Id:    id,
		Doc:   doc,
		FnCbk: func() {

		},
		FnEnd: func() {

		},
	}
}

func GetKeys2Obj(ms map[string]interface{}, m map[string]interface{}, s string) map[string]interface{} {
	a := strings.Split(s, ",")
	for _, x := range a {
		if v, ok := ms[x]; ok {
			m[x] = v
		}
	}
	return m
}

func DoOneCommit(m3 map[string]interface{}, r *CrawlerEngine, r1 *Site) {
	var m5 = map[string]interface{}{}
	var cmt interface{}
	var ok bool
	if cmt, ok = m3["commit"]; !ok {
		if cmt, ok = m3["workflow_runs"]; !ok {
			cmt = m3
		}
	}
	if m6, ok := cmt.(map[string]interface{}); ok {
		m5 = GetKeys2Obj(m6, m5, "url,message,title,body,html_url,head_branch")
		// 清洗后保存
		SendDoc2Buf(r, GetUrlId(fmt.Sprintf("%v", m6["url"])), r1.IndexName, m5)
	}
}

// 项目的提交信息
//  curl -H "Authorization: Bearer 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" "https://api.github.com/repos/mempool/mempool/commits?page=1&per_page=100"
func GetCommitFromApi(user, project string, r1 *Site, r *CrawlerEngine) {
	szUrl := DefaultGithubApiUrl + user + "/" + project + "/commits?page=0&per_page=100000"
	if m1 := GetObjFromUrl(szUrl, r1, r); nil != m1 {
		if a, ok := m1.([]map[string]interface{}); ok {
			for _, m3 := range a {
				DoOneCommit(m3, r, r1)
			}
		}
	}
}

func GetIssueFromApi(user, project string, r1 *Site, r *CrawlerEngine) {
	szUrl := DefaultGithubApiUrl + user + "/" + project + "/issues?page=0&per_page=100000&state=all"
	if m1 := GetObjFromUrl(szUrl, r1, r); nil != m1 {
		if a, ok := m1.([]map[string]interface{}); ok {
			for _, m3 := range a {
				DoOneCommit(m3, r, r1)
			}
		}
	}
}

func GetRelaseFromApi(user, project string, r1 *Site, r *CrawlerEngine) {
	szUrl := DefaultGithubApiUrl + user + "/" + project + "/releases?page=0&per_page=100000"
	if m1 := GetObjFromUrl(szUrl, r1, r); nil != m1 {
		if a, ok := m1.([]map[string]interface{}); ok {
			for _, m3 := range a {
				DoOneCommit(m3, r, r1)
			}
		}
	}
}

func GetPullsFromApi(user, project string, r1 *Site, r *CrawlerEngine) {
	szUrl := DefaultGithubApiUrl + user + "/" + project + "/pulls?page=0&per_page=10000&state=all"
	if m1 := GetObjFromUrl(szUrl, r1, r); nil != m1 {
		if a, ok := m1.([]map[string]interface{}); ok {
			for _, m3 := range a {
				DoOneCommit(m3, r, r1)
			}
		}
	}
}

func GetActionFromApi(user, project string, r1 *Site, r *CrawlerEngine) {
	szUrl := DefaultGithubApiUrl + user + "/" + project + "/actions/runs?page=0&per_page=10000"
	if m1 := GetObjFromUrl(szUrl, r1, r); nil != m1 {
		if a, ok := m1.([]map[string]interface{}); ok {
			for _, m3 := range a {
				DoOneCommit(m3, r, r1)
			}
		}
	}
}

// 添加到项目中
func AddProject(u string, r1 *Site) {
	oU, _ := url.Parse(u)
	a1 := strings.Split(oU.Path, "/")
	var a = r1.GitProjects
	for _, x := range a {
		if x.User == a1[1] {
			x.Projects = append(x.Projects, a1[2])
			return
		}
	}
	r1.GitProjects = append(r1.GitProjects, &GitProjects{User: a1[1], Projects: []string{a1[2]}})
}

func DoSearch(key string, r *CrawlerEngine, r1 *Site) {
	for i := 1; i < 31; i++ {
		szUrl := fmt.Sprintf("https://api.github.com/search/repositories?l=%s&o=desc&q=stars%%3A%%3E500&type=Repositorie&p=%d", key, i)
		if m1 := GetObjFromUrl(szUrl, r1, r); nil != m1 {
			if o := GetJson4Query(m1, ".items"); nil != o {
				if a, ok := o.([]map[string]interface{}); ok {
					for _, x := range a {
						if html_url := GetJson4Query(x, ".html_url"); nil != html_url {
							AddProject(fmt.Sprintf("%v", html_url), r1)
						}
					}
				}
			}
		}
	}
}

// 搜索所有项目并加到爬虫列表中
func DoSearchAll(r *CrawlerEngine, r1 *Site) {
	for _, k := range r1.SearchKeys {
		DoSearch(k, r, r1)
	}
}

// github
//  1-
func DoStartGithub(r *CrawlerEngine, u string, r1 *Site) {
	DoSearchAll(r, r1)
	for _, x := range r1.GitProjects {
		for _, j := range x.Projects {
			GetActionFromApi(x.User, j, r1, r)
			GetPullsFromApi(x.User, j, r1, r)
			GetRelaseFromApi(x.User, j, r1, r)
			GetIssueFromApi(x.User, j, r1, r)
			GetCommitFromApi(x.User, j, r1, r)
			if brcs := GetBranches(x.User, j, r1, r); nil != brcs {
				EacheBranchesItem(brcs, r, r1)
			}
		}
	}
}
