package blevExp

import (
	util "github.com/hktalent/go-utils"
	"time"
)

type IndexDoc4BatchImp struct {
	Doc         chan *IndexData
	fnOk, fnEnd func()
	IndexName   string
}

func NewIndexDoc4BatchImp(szIndex string, fnOk func(), fnEnd func()) *IndexDoc4BatchImp {
	x := &IndexDoc4BatchImp{IndexName: szIndex, fnOk: fnOk, fnEnd: fnEnd, Doc: make(chan *IndexData, 5005)}
	go x.Run()
	return x
}

func (r *IndexDoc4BatchImp) Run() {
	util.DoSyncFunc(func() {
		tk := time.NewTicker(2 * time.Second)
		var n int = 0
		defer func() {
			tk.Stop()
			n = len(r.Doc)
			if 0 < n {
				SaveIndexDoc4Batch(r.IndexName, r.Doc, r.fnOk, r.fnEnd)
				close(r.Doc)
			}
		}()
		var nLst int64 = 0
		//var nFlg = 0
		for {
			select {
			case <-tk.C:
				//if 0 == len(r.Doc) {
				//	nFlg++
				//} else {
				//	nFlg = 0
				//}
				//if nFlg > 2 {
				//	break
				//}

				n = len(r.Doc)
				if 5000 < n {
					nLst = 0
					SaveIndexDoc4Batch(r.IndexName, r.Doc, r.fnOk, r.fnEnd)
				} else if 0 == n { // 1分钟没有任务就退出
					if 0 == nLst {
						nLst = time.Now().UnixMilli()
					} else {
						if (time.Now().UnixMilli() - nLst) > 60000 {
							break
						}
					}
				}
			default:

			}
		}
	})
}
