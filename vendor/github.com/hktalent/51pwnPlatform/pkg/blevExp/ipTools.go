package blevExp

import (
	_ "embed"
	"github.com/bogdanovich/dns_resolver"
	util "github.com/hktalent/go-utils"
	"net"
	"net/url"
	"strings"
)

// 关键词识别、标识
//go:embed dict/CN.txt
var cnIps string

var aCidrs = []*net.IPNet{}

func init() {
	util.RegInitFunc(func() {
		if a := strings.Split(strings.TrimSpace(cnIps), "\n"); 0 < len(a) {
			for _, x := range a {
				if ip1, ir1, err := net.ParseCIDR(x); nil == err {
					if !ip1.IsPrivate() {
						aCidrs = append(aCidrs, ir1)
					}
				}
			}
		}
	})
}

// 判断是否为中国ip
func IsChinaIp(ip string) bool {
	ip1 := net.ParseIP(ip)
	for _, x := range aCidrs {
		if x.Contains(ip1) {
			return true
		}
	}
	return false
}

func getDomainIps(s string) []string {
	if 0 < len(ipReg.FindAllString(s, -1)) && ipCl.ReplaceAllString(s, "") == "" {
		return []string{s}
	}
	resolver := dns_resolver.New([]string{"8.8.8.8", "8.8.4.4"})
	resolver.RetryTimes = 5

	ip, err := resolver.LookupHost(s)
	if err != nil {
		//logrus.Error(err)
		return nil
	}
	if 0 < len(ip) {
		var aR []string
		for _, x := range ip {
			if !x.IsPrivate() {
				aR = append(aR, x.String())
			}
		}
		return aR

	}
	return nil
}
func IsChinaDomain(u string) bool {
	k := u + "IsChinaDomain"
	if r1, err := util.GetAny[string](k); nil == err {
		return string(r1) == "true"
	}
	bR := IsChinaDomain1(u)
	util.PutAny[string](k, "true")
	return bR
}
func IsChinaDomain1(u string) bool {
	if ip := getDomainIps(u); nil != ip && 0 < len(ip) {
		for _, x := range ip {
			if IsChinaIp(x) {
				return true
			}
		}
	}
	return false
}
func IsChinaUrl(u string) bool {
	if u1, err := url.Parse(u); nil == err && "" != u1.Hostname() {
		return IsChinaDomain(u1.Hostname())
	}
	return false
}
