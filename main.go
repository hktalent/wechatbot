package main

import (
	"github.com/hktalent/51pwnPlatform/pkg/blevExp"
	util "github.com/hktalent/go-utils"
	"github.com/hktalent/wechatbot/bootstrap"
	"github.com/hktalent/wechatbot/gtp"
	"os"
)

func main() {
	util.DoInitAll()
	os.MkdirAll("data", os.ModePerm)
	blevExp.InitIndexDb()
	blevExp.CreateIndex(gtp.DefaultIndexName, `{"default_mapping":{"enabled":true,"display_order":"0"},"type_field":"_type","default_type":"_default","default_analyzer":"standard","default_datetime_parser":"dateTimeOptional","default_field":"_all","byte_array_converter":"json","store_dynamic":true,"index_dynamic":true}`)
	bootstrap.Run()
	util.Wg.Wait()
	util.CloseAll()

}
