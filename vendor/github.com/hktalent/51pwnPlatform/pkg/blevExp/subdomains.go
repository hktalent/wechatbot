package blevExp

import (
	"fmt"
	util "github.com/hktalent/go-utils"
	"io/fs"
	"io/ioutil"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

/*
wget -c 'https://chaos-data.projectdiscovery.io/index.json'
cat chaos.json|jq ".[].URL"|sed 's/"//g'|sort -u|xargs -I % wget -c %
*/
var tc = make(chan struct{}, 64)
var docs = make(chan *IndexData, 5000)
var doIt = make(chan struct{}, 5)
var fnOk = func() {}

func VisitSubDomains(path string, di fs.DirEntry, err error) error {
	if y, err := util.GetAny[string](path); err == nil && "1" == y {
		fmt.Println(path, " skip")
		return nil
	}
	if strings.HasSuffix(path, ".txt") {
		a := strings.Split(path, "/")
		szRoot := a[len(a)-1:][0]
		szRoot = szRoot[0 : len(szRoot)-4]
		if data, err := ioutil.ReadFile(path); nil == err {
			a = strings.Split(strings.TrimSpace(string(data)), "\n")
			var wg sync.WaitGroup
			fmt.Printf("\nstart subdomain: %d\n", len(a))
			for _, x := range a {
				x = strings.TrimSpace(x)
				if 3 < len(x) {
					tc <- struct{}{}
					wg.Add(1)
					go func(x string) {
						defer func() {
							<-tc
							wg.Done()
						}()
						if aIp := getDomainIps(x); 0 < len(aIp) {
							//fmt.Printf("start %s\r", x)
							var m1 = map[string]interface{}{"domain": x, "root": szRoot, "ips": aIp, "tags": []string{"chaos", "bugbounty"}, "type": "subdomain"}
							docs <- &IndexData{
								Index: DefaulIndexName,
								Id:    x,
								Doc:   m1,
								FnCbk: fnOk, FnEnd: fnOk,
							}
							if 500 < len(docs) {
								doIt <- struct{}{}
							}
							//go SaveIndexDoc(DefaulIndexName, x, m1, func() {
							//	fmt.Printf("%s save ok\r", x)
							//}, func() {
							//	atomic.AddInt32(&n1, 1)
							//	if int32(len(a)) == n1 {
							//		util.PutAny[string](path, "1")
							//		fmt.Printf("\nok: %s\n", path)
							//	}
							//})
						}
					}(x)
				}
			}
			wg.Wait()
			doIt <- struct{}{}
			//SaveIndexDoc4Batch(DefaulIndexName, docs, func() {
			//	fmt.Printf("save ok: %d \r", len(a))
			//}, func() {
			//	util.PutAny[string](path, "1")
			//})
		}
	}
	return nil
}

var doEnd = make(chan struct{}, 1)

func asyncBatch() {
	var fnOk = func() {}
	var fnEnd = fnOk
	tk := time.NewTicker(10 * time.Second)
	defer tk.Stop()
	var fnDoIt = func() {
		n := len(docs)
		if 0 < n {
			SaveIndexDoc4Batch(DefaulIndexName, docs, func() {
				fmt.Printf("save subdomain   ok: %d \r", n)
			}, fnEnd)
		}
	}
	for {
		select {
		case <-doEnd:
			fnDoIt()
		case <-doIt:
			fnDoIt()
		case <-tk.C:
			//if 500 < len(docs)
			{
				fnDoIt()
			}
		}
	}
}

func DoInitSubdomains() {
	go asyncBatch()
	if err := filepath.WalkDir("config/chaos/", VisitSubDomains); nil != err {
		fmt.Printf("filepath.WalkDir() returned %v\n", err)
	}
	doEnd <- struct{}{}
}
