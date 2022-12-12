package blevExp

import (
	"bytes"
	"github.com/hktalent/colly"
	"github.com/mmcdole/gofeed/rss"
	"strings"
)

// 解析 RSS 链接
func ParseRSSReader(res *colly.Response, doc chan *IndexData, r1 *Site, reTry chan string) {
	i := bytes.NewReader(res.Body)
	fp := rss.Parser{}
	if rf, err := fp.Parse(i); nil == err {
		for _, r1 := range rf.Items {
			if s := strings.TrimSpace(r1.Link); "" != s {
				reTry <- s
			}
		}
	}
}
