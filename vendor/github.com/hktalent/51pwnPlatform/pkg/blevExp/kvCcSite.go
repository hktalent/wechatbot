package blevExp

import (
	"github.com/hktalent/kvDb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type kvCcSiteImp struct {
	Host2Site *kvDb.KvDb `json:"Host2Site"`
}

func NewkvCcSiteImp() *kvCcSiteImp {
	return &kvCcSiteImp{Host2Site: kvDb.NewKvDb("db/site", nil)}
}

func (r *kvCcSiteImp) Get(out chan interface{}, fnCbk func([]byte), a ...any) {
	r.Host2Site.Get(out, fnCbk, a...)
}

func (r *kvCcSiteImp) Delete(a ...any) bool {
	return r.Host2Site.Delete(a...)
}

func (r *kvCcSiteImp) Put(a ...any) bool {
	return r.Host2Site.Put(a...)
}

func (r *kvCcSiteImp) Iterator(fnCbk func(iterator.Iterator) bool, slice *util.Range) {
	r.Iterator(fnCbk, slice)
}
