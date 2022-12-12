package blevExp

import (
	"fmt"
	util "github.com/hktalent/go-utils"
	"github.com/shomali11/util/xhashes"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type PswdImp struct {
	User   string `json:"user"`
	Passwd string `json:"passwd"`
}

var c chan struct{}

func init() {
	util.RegInitFunc(func() {
		c = make(chan struct{}, 4)
	})
}

var (
	nTcnt   int64 = 0
	NSubmit       = 10000
	DocPsd        = make(chan *IndexData, NSubmit+50)
	pF            = message.NewPrinter(language.English)
	DocDef        = make(chan *IndexData, NSubmit+50)
)

var fnSaveOk = func(n int, s string) func() {
	return func() {
		fmt.Printf("save %s pswd ok: %8d \r", s, n)
	}
}
var Nodo = func() {}

var doIt1 = func(s string) {
	n3 := len(DocPsd)
	if 0 < n3 {
		SaveIndexDoc4Batch("passwd", DocPsd, fnSaveOk(n3, s), func() {
			util.PutAny[string](s, "1")
		})
	}
}

func DoFile(s string) {
	c <- struct{}{}
	go func() {
		defer func() {
			<-c
		}()
		//if y, err := util.GetAny[string](s); err == nil && "1" == y {
		//	log.Println(s, " skip")
		//	return
		//}

		defer doIt1(s)
		if data, err := ioutil.ReadFile(s); nil == err {
			if a := strings.Split(strings.TrimSpace(string(data)), "\n"); 0 < len(a) {
				wg := sync.WaitGroup{}
				fmt.Printf("\nstart %s pswd: %d\n", s, len(a))
				for _, x := range a {
					x = strings.TrimSpace(x)
					n := strings.Index(x, ":")
					if "" == x || -1 == n {
						continue
					}
					nTcnt++
					//fmt.Printf(pF.Sprintf("start %8f   ", nTcnt))
					//fmt.Printf("start %8d\r", nTcnt)
					wg.Add(1)
					go func(x1 string) {
						defer wg.Done()
						szID := xhashes.SHA1(x1)
						if nil != GetDoc("passwd", szID) {
							return
						}
						DocPsd <- &IndexData{
							Index: "passwd",
							Doc:   &PswdImp{User: strings.TrimSpace(x1[0:n]), Passwd: strings.TrimSpace(x1[n+1:])},
							FnEnd: Nodo,
							FnCbk: Nodo,
						}
						n3 := len(DocPsd)
						if n3 >= NSubmit {
							SaveIndexDoc4Batch("passwd", DocPsd, fnSaveOk(n3, s), Nodo)
						}
					}(x)
				}
				wg.Wait()
				doIt1(s)
			}
		}
	}()
}

func Visit(path string, di fs.DirEntry, err error) error {
	if strings.Contains(path, "/data/") && !strings.HasSuffix(path, "/symbols") && !strings.HasSuffix(path, "/.DS_Store") {
		//fmt.Printf("start: %s\n", path)
		DoFile(path)
	}
	return nil
}

// 构建 41G 密码社工库
func DoPassWd() {
	os.Setenv("CacheName", ".pswdCc")
	//util.DoInitAll()
	//util.NewKvCachedb()
	if err := filepath.WalkDir(util.GetVal("BreachCompilation"), Visit); nil != err {
		//if err := filepath.WalkDir("/volume1/home/admin/MyWork/sgk/BreachCompilation/data", Visit); nil != err {
		fmt.Printf("filepath.WalkDir() returned %v\n", err)
	}
	//util.Wg.Wait()
	//util.CloseAll()
}
