package blevExp

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/document"
	"github.com/gin-gonic/gin"
	"github.com/hktalent/51pwnPlatform/pkg/blevExp/formatImp"
	"github.com/hktalent/PipelineHttp"
	"github.com/hktalent/colly"
	util "github.com/hktalent/go-utils"
	xj "github.com/hktalent/goxml2json"
	"github.com/simonnilsson/ask"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var (
	ppHttp     = PipelineHttp.NewPipelineHttp()
	ServerUrl  = "http://127.0.0.1:8095/api/%s/%v"
	SaveThread chan struct{}
)

func SetCC(id string, args ...string) {
	log.Printf("ok: %s %s", strings.Join(args, " "), id)
	util.PutAny[string](id, "1")
}

func init() {
	util.RegInitFunc(func() {
		ServerUrl = util.GetVal("ServerUrl")
		SaveThread = make(chan struct{}, util.GetValAsInt("thread", 64))
	})
}

// 获取body中的图片，并返回base64的编码
func GetImg2Base64(szUrl string, s string) string {
	szrst := ""
	c01 := ppHttp.GetClient(nil)
	c01.CheckRedirect = nil
	ppHttp.DoGetWithClient4SetHd(c01, szUrl, "GET", nil, func(resp *http.Response, err error, szU string) {
		if nil == err && nil != resp {
			defer resp.Body.Close()
			if data, err := ioutil.ReadAll(resp.Body); nil == err {
				szrst = base64.StdEncoding.EncodeToString(data)
			}
		}
	}, func() map[string]string {
		return map[string]string{}
	}, true)
	return szrst
}

// 保存爬到到文章
//  坑：json的strutc首字母必须大写，且设置json
func SaveIndexData(index, id string, o interface{}, okCbk func(...interface{}), endCbk func()) bool {
	var bRst = false
	SaveThread <- struct{}{}
	util.DoSyncFunc(func() {
		defer func() {
			<-SaveThread
			endCbk()
		}()
		data, err := json.Marshal(o)
		if nil != err || 3 > len(data) {
			log.Printf("json.Marshal index: %s id: %s %v\n", index, id, err)
			return
		}
		ppHttp.ErrLimit = 99999
		ppHttp.ErrCount = 0
		ppHttp.DoGetWithClient4SetHd(ppHttp.Client, fmt.Sprintf("http://127.0.0.1:8095/api/%s/%v", index, id), "PUT", bytes.NewReader(data), func(resp *http.Response, err error, szU string) {
			if nil == err && resp != nil {
				defer resp.Body.Close()
				bRst = true
				//data, _ := ioutil.ReadAll(resp.Body)
				//log.Println(index, id, " is save ok ", string(data))
				okCbk(index, id, o)
			} else if nil != err {
				log.Println("ppHttp.DoGetWithClient4SetHd", err)
			}
		}, func() map[string]string {
			return map[string]string{
				"User-Agent":   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.85 Safari/537.36",
				"Content-Type": "application/json"}
		}, true)
	})
	return bRst
}

// 创建 Handle
func CreateHandle(getIndexHandler ServeHTTPFace) gin.HandlerFunc {
	return func(context *gin.Context) {
		getIndexHandler.ServeHTTP(context.Writer, context.Request)
	}
}

// 入参数不能传指针，否则无法正确获得结果
func GetJson4Query(source interface{}, path string) interface{} {
	res := ask.For(source, path)
	if nil != res {
		return res.Value()
	}
	return nil
}

func GetStrFromObj(o interface{}, path string) string {
	return fmt.Sprintf("%v", GetJson4Query(o, path))
	return ""
}

func CvtData(data interface{}, id string, dates *[]string, boolFeild *[]string, numFeild ...string) interface{} {
	if oR, ok := data.(map[string]interface{}); ok {
		if nil != dates {
			for _, x := range *dates {
				if d, err := formatImp.ParseDateTime(fmt.Sprintf("%v", GetStrFromObj(oR, "."+x))); nil == err {
					oR[x] = d
				}
			}
		}
		if 0 < len(numFeild) && nil != numFeild {
			for _, x := range numFeild {
				if i, err := strconv.Atoi(fmt.Sprintf("%v", GetStrFromObj(oR, "."+x))); nil == err {
					oR[x] = i
				}
			}
		}
		if nil != boolFeild {
			for _, x := range *boolFeild {
				oR[x] = "true" == strings.ToLower(fmt.Sprintf("%v", GetStrFromObj(oR, "."+x)))
			}
		}
		return oR
	}
	return data
}

// 索引数据处理
func CvtData1(data interface{}, id string, dates *[]string, boolFeild *[]string, numFeild ...string) interface{} {
	dtf := bleve.NewDateTimeFieldMapping()
	mapping := bleve.NewIndexMapping()
	if nil != dates {
		for _, x := range *dates {
			mapping.DefaultMapping.AddFieldMappingsAt(x, dtf)
		}
	}
	dt1 := bleve.NewNumericFieldMapping()
	if 0 < len(numFeild) && nil != numFeild {
		for _, x := range numFeild {
			mapping.DefaultMapping.AddFieldMappingsAt(x, dt1)
		}
	}
	dt2 := bleve.NewBooleanFieldMapping()
	if nil != boolFeild {
		for _, x := range *boolFeild {
			mapping.DefaultMapping.AddFieldMappingsAt(x, dt2)
		}
	}
	doc := document.NewDocument(id)
	if err := mapping.MapDocument(doc, data); nil != err {
		log.Println(err)
	}
	if x1, err := json.Marshal(doc); nil == err {
		log.Println(string(x1))
	}
	return doc
}

// 解析额日期
func ParseDate(s string, af []string) *time.Time {
	for _, k := range af {
		if d, err := time.Parse(k, s); nil == err {
			return &d
		}
	}
	return nil
}

func DoZipFile(res *colly.Response, reTry chan string, doFileCbk func(oI io.Reader)) {
	xj.AttrPrefix = ""
	xj.ContentPrefix = ""
	if !strings.HasSuffix(res.Request.URL.String(), ".zip") {
		return
	}
	n11 := int64(len(res.Body))
	if n10 := res.Headers.Get("Content-Length"); "" != n10 {
		if n09, err := strconv.Atoi(n10); nil == err {
			if n11 < int64(n09) {
				reTry <- res.Request.URL.String()
				return
			}
		}
	}
	if zi, err := zip.NewReader(io.NewSectionReader(bytes.NewReader(res.Body), 0, n11), n11); nil == err {
		for _, f := range zi.File {
			if fio, err := f.Open(); nil == err {
				defer fio.Close()
				doFileCbk(fio)
			} else {
				reTry <- res.Request.URL.String()
				return
			}
		}
	} else {
		log.Println(err, res.Request.URL.String(), res.Headers)
		reTry <- res.Request.URL.String()
	}
}

func DoGet(szUrl string, hd map[string]string) (resp *http.Response, err error) {
	ppHttp.DoGetWithClient4SetHd(nil, szUrl, "GET", nil, func(resp1 *http.Response, err1 error, szU string) {
		resp = resp1
		err = err1
	}, func() map[string]string {
		return hd
	}, false)
	return resp, err
}
