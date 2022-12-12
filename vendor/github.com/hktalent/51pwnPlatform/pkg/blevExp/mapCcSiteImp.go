package blevExp

import (
	"fmt"
	"github.com/syndtr/goleveldb/leveldb/util"
	"sync"
)

type MapCcSiteImp struct {
	Host2Site sync.Map
}

func NewMapCcSiteImp() *MapCcSiteImp {
	return &MapCcSiteImp{Host2Site: sync.Map{}}
}

var lcrw sync.RWMutex

func (r *MapCcSiteImp) Get(fnCbk func(*Site), a ...any) {
	lcrw.Lock()
	defer lcrw.Unlock()
	for _, x := range a {
		s := fmt.Sprintf("%v", x)
		if o, ok := r.Host2Site.Load(s); ok {
			if x, ok := o.(*Site); ok {
				fnCbk(x)
			}
		}
	}
}

func (r *MapCcSiteImp) Delete(a ...any) bool {
	lcrw.Lock()
	defer lcrw.Unlock()
	for _, x := range a {
		r.Host2Site.Delete(x)
	}
	return true
}

func (r *MapCcSiteImp) Put(a ...any) bool {
	lcrw.Lock()
	defer lcrw.Unlock()
	for i := 0; i < len(a); i += 2 {
		r.Host2Site.Store(a[i], a[i+1])
	}
	return true
}

func (r *MapCcSiteImp) Iterator(fnCbk func(a ...any) bool, slice *util.Range) {
	lcrw.Lock()
	defer lcrw.Unlock()
	r.Host2Site.Range(func(key, value any) bool {
		return fnCbk(key, value)
	})
}

func (r *MapCcSiteImp) Close() {}
