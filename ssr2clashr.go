package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)


func getSubs(link string) (sub []string, err error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", link, nil)
	if err!=nil {
		return
	}
	req.Header.Add("Accept","text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3")
	req.Header.Add("Accept-Encoding","gzip, deflate")
	req.Header.Add("Accept-Language","zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Add("Cache-Control","no-cache")
	req.Header.Add("User-Agent","Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/77.0.3865.75 Safari/537.36")
	resp, err := client.Do(req)
	if err!=nil {
		log.Println(err)
		return
	}
	log.Println(resp.ContentLength)
	defer resp.Body.Close()
	bodybytes, err := ioutil.ReadAll(resp.Body)
	if err!=nil {
		fmt.Println(err)
		return
	}
	subs := base64decode_urlsafe(string(bodybytes))
	subs = strings.ReplaceAll(subs, "\r", "")
	sub = strings.Split(subs, "\n")
	return
}

func base64decode_urlsafe(src string) (dst string) {
	if i:=len(src)%4; i!=0 {
		for k:=0; k<i; k++ {
			src += string(base64.StdPadding)
		}
	}
	src = strings.ReplaceAll(src, "-", "+")
	src = strings.ReplaceAll(src, "_", "/")
	dstbytes,_ := base64.URLEncoding.DecodeString(src)
	dst = string(dstbytes)
	return
}

type SSR struct {
	Name          string 	`yaml:"name"`
	Server        string	`yaml:"server"`
	Port          int		`yaml:"port"`
	Password      string	`yaml:"password"`
	Cipher        string	`yaml:"cipher"`
	Protocol      string	`yaml:"protocol"`
	ProtocolParam string	`yaml:"protocolparam"`
	Obfs          string	`yaml:"obfs"`
	ObfsParam     string	`yaml:"obfsparam"`
}

func ssrlink2confstr(ssrlink string) (conftxt string, name string, group string) {
	if !strings.HasPrefix(ssrlink, "ssr") {
		return
	}
	ssrmix := strings.SplitN(ssrlink,"://", 2)
	if len(ssrmix)<2 {
		return
	}
	ssrmi := ssrmix[1]
	ssrming := base64decode_urlsafe(ssrmi)
	ssrmingsp := strings.SplitN(ssrming, "/?", 2)
	ssrpath := strings.Split(ssrmingsp[0], ":")
	ssrquery := ssrmingsp[1]
	u,_ := url.ParseQuery(ssrquery)
	port,_ := strconv.Atoi(ssrpath[1])
	ssrins := SSR{
		Name:          base64decode_urlsafe(u.Get("remarks")),
		Server:        ssrpath[0],
		Port:          port,
		Password:      base64decode_urlsafe(ssrpath[5]),
		Cipher:        ssrpath[3],
		Protocol:      ssrpath[2],
		ProtocolParam: base64decode_urlsafe(u.Get("protoparam")),
		Obfs:          ssrpath[4],
		ObfsParam:     base64decode_urlsafe(u.Get("obfsparam")),
	}
	//log.Println(ssrins)
	conftxt = `- { name: "`+ssrins.Name+`", type: ssr, server: "`+ssrins.Server+`", port: `+strconv.Itoa(ssrins.Port)+`, password: "`+ssrins.Password+`", cipher: "`+ssrins.Cipher+`", protocol: "`+ssrins.Protocol+`", protocolparam: "`+ssrins.ProtocolParam+`", obfs: "`+ssrins.Obfs+`", obfsparam: "`+ssrins.ObfsParam+`" }` +"\r\n"
	name = ssrins.Name
	group = base64decode_urlsafe(u.Get("group"))
	return
}

func gengroup(allgroups []string) (groupconf string) {
	groupconf += "\r\n\r\nProxy Group:\r\n\r\n"
	proxylistbytes,_ := json.Marshal(allgroups)
	proxylist := string(proxylistbytes)
	groupconf += `- { name: "延迟最低", type: url-test, proxies: `+proxylist+`, url: "http://www.gstatic.com/generate_204", interval: 300 }` +"\r\n"
	groupconf += `- { name: "故障切换", type: fallback, proxies: `+proxylist+`, url: "http://www.gstatic.com/generate_204", interval: 300 }` +"\r\n"
	groupconf += `- { name: "负载均衡", type: load-balance, proxies: `+proxylist+`, url: "http://www.gstatic.com/generate_204", interval: 300 }` +"\r\n"
	groupconf += `- { name: "手动切换", type: select, proxies: `+proxylist+` }` +"\r\n"
	groupconf += `- { name: "选择模式", type: select, proxies: ["手动切换", "延迟最低", "负载均衡", "故障切换", "DIRECT"] }` + "\r\n"
	groupconf += `- { name: "Apple服务", type: select, proxies: ["DIRECT", "手动切换"] }` + "\r\n"
	groupconf += `- { name: "国际媒体", type: select, proxies: ["手动切换"] }` + "\r\n"
	groupconf += `- { name: "国内媒体", type: select, proxies: ["DIRECT"] }` + "\r\n"
	groupconf += `- { name: "屏蔽网站", type: select, proxies: ["REJECT", "DIRECT"] }` + "\r\n"
	return
}




func ssrsub2clashconf(sub string) (data string, group string, err error) {
	subs,err := getSubs(sub)
	if err!=nil {
		return "","",err
	}
	ssrnames := []string{}
	agroup := "Peekfun"
	ssrproxystr := "\r\n\r\nProxy:\r\n\r\n"
	for idx,sub := range subs {
		if sub=="" {
			continue
		}
		ssrtxt,ssrname, sgroup := ssrlink2confstr(sub)
		if ssrtxt==""{
			continue
		}
		if !in(&ssrnames, ssrname) {
			ssrnames = append(ssrnames, ssrname)
			ssrproxystr += ssrtxt
		}
		if idx==3{
			agroup = sgroup
		}
	}
	proxygroup := gengroup(ssrnames)
	data = clashconf1+ssrproxystr+proxygroup+clashconf2
	group = agroup
	return
}

func in(slicee *[]string, itemm string) bool {
	if strings.Contains(itemm, "剩余流量")||strings.Contains(itemm, "过期时间"){
		return true
	}
	for _,i := range *slicee {
		if itemm==i {
			return true
		}
	}
	return false
}




var clashconf1 = `
# telegram频道： https://t.me/peekfun
# telegram交流群： https://t.me/joinchat/K5hKwle9NveVqnzucnuxQw
# clashR 新版内核将在群里更新
# 新版 clashR for windows客户端 进群自取
# 如有使用问题，截图log进群反馈

# HTTP 代理端口
port: 7890

# SOCKS5 代理端口
socks-port: 7891

# Linux 和 macOS 的 redir 代理端口 (如需使用此功能，请取消注释)
# redir-port: 7892

# 允许局域网的连接（可用来共享代理）
allow-lan: false

# 规则模式：Rule（规则） / Global（全局代理）/ Direct（全局直连）
mode: Rule

# 设置日志输出级别 (默认级别：silent，即不输出任何内容，以避免因日志内容过大而导致程序内存溢出）。
# 5 个级别：silent / info / warning / error / debug。级别越高日志输出量越大，越倾向于调试，若需要请自行开启。
log-level: info

# clash 的 RESTful API
external-controller: 127.0.0.1:9090
# external-ui: folder
# external-ui: "/home/king/clashconf/dist"

# RESTful API 的口令 (可选)
# secret: ""

# DNS 设置
#dns:
#  enable: true
#  listen: 0.0.0.0:53
#  enhanced-mode: fake-ip
#  nameserver:
#   - 119.29.29.29
#   - 223.5.5.5

# Clash for Windows
cfw-bypass:
  - qq.com
  - music.163.com
  - '*.music.126.net'
  - localhost
  - 127.*
  - 10.*
  - 172.16.*
  - 172.17.*
  - 172.18.*
  - 172.19.*
  - 172.20.*
  - 172.21.*
  - 172.22.*
  - 172.23.*
  - 172.24.*
  - 172.25.*
  - 172.26.*
  - 172.27.*
  - 172.28.*
  - 172.29.*
  - 172.30.*
  - 172.31.*
  - 192.168.*
  - <local>
cfw-latency-timeout: 5000

`

var clashconf2 = `


# 规则
Rule:
# Feeb
- DOMAIN-KEYWORD,epochtimes,REJECT
- DOMAIN-SUFFIX,881903.com,REJECT
- DOMAIN-SUFFIX,aboluowang.com,REJECT
- DOMAIN-SUFFIX,bannedbook.org,REJECT
- DOMAIN-SUFFIX,bldaily.com,REJECT
- DOMAIN-SUFFIX,china21.org,REJECT
- DOMAIN-SUFFIX,chinaaffairs.org,REJECT
- DOMAIN-SUFFIX,dajiyuan.com,REJECT
- DOMAIN-SUFFIX,dalianmeng.org,REJECT
- DOMAIN-SUFFIX,dkn.tv,REJECT
- DOMAIN-SUFFIX,dongtaiwang.com,REJECT
- DOMAIN-SUFFIX,edoors.com,REJECT
- DOMAIN-SUFFIX,epochweekly.com,REJECT
- DOMAIN-SUFFIX,falundafa.org,REJECT
- DOMAIN-SUFFIX,fgmtv.org,REJECT
- DOMAIN-SUFFIX,gardennetworks.com,REJECT
- DOMAIN-SUFFIX,gongyiluntan.org,REJECT
- DOMAIN-SUFFIX,gpass1.com,REJECT
- DOMAIN-SUFFIX,hrichina.org,REJECT
- DOMAIN-SUFFIX,huanghuagang.org,REJECT
- DOMAIN-SUFFIX,internetfreedom.org,REJECT
- DOMAIN-SUFFIX,kanzhongguo.com,REJECT
- DOMAIN-SUFFIX,lagranepoca.com,REJECT
- DOMAIN-SUFFIX,mh4u.org,REJECT
- DOMAIN-SUFFIX,mhradio.org,REJECT
- DOMAIN-SUFFIX,minghui.org,REJECT
- DOMAIN-SUFFIX,newrealmstudios.ca,REJECT
- DOMAIN-SUFFIX,ntdtv.com,REJECT
- DOMAIN-SUFFIX,ogate.org,REJECT
- DOMAIN-SUFFIX,open.com.hk,REJECT
- DOMAIN-SUFFIX,organcare.org.tw,REJECT
- DOMAIN-SUFFIX,qxbbs.org,REJECT
- DOMAIN-SUFFIX,renminbao.com,REJECT
- DOMAIN-SUFFIX,secretchina.com,REJECT
- DOMAIN-SUFFIX,shenyun.com,REJECT
- DOMAIN-SUFFIX,shenyunperformingarts.org,REJECT
- DOMAIN-SUFFIX,shenzhoufilm.com,REJECT
- DOMAIN-SUFFIX,soundofhope.org,REJECT
- DOMAIN-SUFFIX,theepochtimes.com,REJECT
- DOMAIN-SUFFIX,tiandixing.org,REJECT
- DOMAIN-SUFFIX,tuidang.org,REJECT
- DOMAIN-SUFFIX,velkaepocha.sk,REJECT
- DOMAIN-SUFFIX,watchinese.com,REJECT
- DOMAIN-SUFFIX,wixsite.com,REJECT
- DOMAIN-SUFFIX,wujie.net,REJECT
- DOMAIN-SUFFIX,wujieliulan.com,REJECT
- DOMAIN-SUFFIX,xinsheng.net,REJECT
- DOMAIN-SUFFIX,zhengjian.org,REJECT
- DOMAIN-SUFFIX,zhuichaguoji.org,REJECT

# > Unbreak
# >> Firebase Cloud Messaging
- DOMAIN,mtalk.google.com,DIRECT

# Internet Service Provider 屏蔽网站
- DOMAIN-SUFFIX,17gouwuba.com,屏蔽网站
- DOMAIN-SUFFIX,186078.com,屏蔽网站
- DOMAIN-SUFFIX,189zj.cn,屏蔽网站
- DOMAIN-SUFFIX,285680.com,屏蔽网站
- DOMAIN-SUFFIX,3721zh.com,屏蔽网站
- DOMAIN-SUFFIX,4336wang.cn,屏蔽网站
- DOMAIN-SUFFIX,51chumoping.com,屏蔽网站
- DOMAIN-SUFFIX,51mld.cn,屏蔽网站
- DOMAIN-SUFFIX,51mypc.cn,屏蔽网站
- DOMAIN-SUFFIX,58mingri.cn,屏蔽网站
- DOMAIN-SUFFIX,58mingtian.cn,屏蔽网站
- DOMAIN-SUFFIX,5vl58stm.com,屏蔽网站
- DOMAIN-SUFFIX,6d63d3.com,屏蔽网站
- DOMAIN-SUFFIX,7gg.cc,屏蔽网站
- DOMAIN-SUFFIX,91veg.com,屏蔽网站
- DOMAIN-SUFFIX,9s6q.cn,屏蔽网站
- DOMAIN-SUFFIX,adsame.com,屏蔽网站
- DOMAIN-SUFFIX,aiclk.com,屏蔽网站
- DOMAIN-SUFFIX,akuai.top,屏蔽网站
- DOMAIN-SUFFIX,atplay.cn,屏蔽网站
- DOMAIN-SUFFIX,baiwanchuangyi.com,屏蔽网站
- DOMAIN-SUFFIX,bayimob.com,屏蔽网站
- DOMAIN-SUFFIX,beerto.cn,屏蔽网站
- DOMAIN-SUFFIX,beilamusi.com,屏蔽网站
- DOMAIN-SUFFIX,benshiw.net,屏蔽网站
- DOMAIN-SUFFIX,bianxianmao.com,屏蔽网站
- DOMAIN-SUFFIX,bryonypie.com,屏蔽网站
- DOMAIN-SUFFIX,cishantao.com,屏蔽网站
- DOMAIN-SUFFIX,cszlks.com,屏蔽网站
- DOMAIN-SUFFIX,cudaojia.com,屏蔽网站
- DOMAIN-SUFFIX,dafapromo.com,屏蔽网站
- DOMAIN-SUFFIX,daitdai.com,屏蔽网站
- DOMAIN-SUFFIX,dsaeerf.com,屏蔽网站
- DOMAIN-SUFFIX,dugesheying.com,屏蔽网站
- DOMAIN-SUFFIX,dv8c1t.cn,屏蔽网站
- DOMAIN-SUFFIX,echatu.com,屏蔽网站
- DOMAIN-SUFFIX,erdoscs.com,屏蔽网站
- DOMAIN-SUFFIX,fan-yong.com,屏蔽网站
- DOMAIN-SUFFIX,feih.com.cn,屏蔽网站
- DOMAIN-SUFFIX,fjlqqc.com,屏蔽网站
- DOMAIN-SUFFIX,fkku194.com,屏蔽网站
- DOMAIN-SUFFIX,freedrive.cn,屏蔽网站
- DOMAIN-SUFFIX,gclick.cn,屏蔽网站
- DOMAIN-SUFFIX,goufanli100.com,屏蔽网站
- DOMAIN-SUFFIX,goupaoerdai.com,屏蔽网站
- DOMAIN-SUFFIX,gouwubang.com,屏蔽网站
- DOMAIN-SUFFIX,gzxnlk.com,屏蔽网站
- DOMAIN-SUFFIX,haoshengtoys.com,屏蔽网站
- DOMAIN-SUFFIX,ichaosheng.com,屏蔽网站
- DOMAIN-SUFFIX,ishop789.com,屏蔽网站
- DOMAIN-SUFFIX,jdkic.com,屏蔽网站
- DOMAIN-SUFFIX,jiubuhua.com,屏蔽网站
- DOMAIN-SUFFIX,jwg365.cn,屏蔽网站
- DOMAIN-SUFFIX,kawo77.com,屏蔽网站
- DOMAIN-SUFFIX,kualianyingxiao.cn,屏蔽网站
- DOMAIN-SUFFIX,kumihua.com,屏蔽网站
- DOMAIN-SUFFIX,ltheanine.cn,屏蔽网站
- DOMAIN-SUFFIX,maipinshangmao.com,屏蔽网站
- DOMAIN-SUFFIX,minisplat.cn,屏蔽网站
- DOMAIN-SUFFIX,mkitgfs.com,屏蔽网站
- DOMAIN-SUFFIX,mlnbike.com,屏蔽网站
- DOMAIN-SUFFIX,mobjump.com,屏蔽网站
- DOMAIN-SUFFIX,nbkbgd.cn,屏蔽网站
- DOMAIN-SUFFIX,newapi.com,屏蔽网站
- DOMAIN-SUFFIX,pinzhitmall.com,屏蔽网站
- DOMAIN-SUFFIX,poppyta.com,屏蔽网站
- DOMAIN-SUFFIX,qianchuanghr.com,屏蔽网站
- DOMAIN-SUFFIX,qichexin.com,屏蔽网站
- DOMAIN-SUFFIX,qinchugudao.com,屏蔽网站
- DOMAIN-SUFFIX,quanliyouxi.cn,屏蔽网站
- DOMAIN-SUFFIX,qutaobi.com,屏蔽网站
- DOMAIN-SUFFIX,ry51w.cn,屏蔽网站
- DOMAIN-SUFFIX,sg536.cn,屏蔽网站
- DOMAIN-SUFFIX,sifubo.cn,屏蔽网站
- DOMAIN-SUFFIX,sifuce.cn,屏蔽网站
- DOMAIN-SUFFIX,sifuda.cn,屏蔽网站
- DOMAIN-SUFFIX,sifufu.cn,屏蔽网站
- DOMAIN-SUFFIX,sifuge.cn,屏蔽网站
- DOMAIN-SUFFIX,sifugu.cn,屏蔽网站
- DOMAIN-SUFFIX,sifuhe.cn,屏蔽网站
- DOMAIN-SUFFIX,sifuhu.cn,屏蔽网站
- DOMAIN-SUFFIX,sifuji.cn,屏蔽网站
- DOMAIN-SUFFIX,sifuka.cn,屏蔽网站
- DOMAIN-SUFFIX,smgru.net,屏蔽网站
- DOMAIN-SUFFIX,taoggou.com,屏蔽网站
- DOMAIN-SUFFIX,tcxshop.com,屏蔽网站
- DOMAIN-SUFFIX,tjqonline.cn,屏蔽网站
- DOMAIN-SUFFIX,topitme.com,屏蔽网站
- DOMAIN-SUFFIX,tt3sm4.cn,屏蔽网站
- DOMAIN-SUFFIX,tuia.cn,屏蔽网站
- DOMAIN-SUFFIX,tuipenguin.com,屏蔽网站
- DOMAIN-SUFFIX,tuitiger.com,屏蔽网站
- DOMAIN-SUFFIX,websd8.com,屏蔽网站
- DOMAIN-SUFFIX,wx16999.com,屏蔽网站
- DOMAIN-SUFFIX,xiaohuau.xyz,屏蔽网站
- DOMAIN-SUFFIX,yinmong.com,屏蔽网站
- DOMAIN-SUFFIX,yiqifa.com,屏蔽网站
- DOMAIN-SUFFIX,yitaopt.com,屏蔽网站
- DOMAIN-SUFFIX,yjqiqi.com,屏蔽网站
- DOMAIN-SUFFIX,yukhj.com,屏蔽网站
- DOMAIN-SUFFIX,zhaozecheng.cn,屏蔽网站
- DOMAIN-SUFFIX,zhenxinet.com,屏蔽网站
- DOMAIN-SUFFIX,zlne800.com,屏蔽网站
- DOMAIN-SUFFIX,zunmi.cn,屏蔽网站
- DOMAIN-SUFFIX,zzd6.com,屏蔽网站
- IP-CIDR,39.107.15.115/32,屏蔽网站
- IP-CIDR,47.89.59.182/32,屏蔽网站
- IP-CIDR,103.49.209.27/32,屏蔽网站
- IP-CIDR,123.56.152.96/32,屏蔽网站
# > ChinaNet
- IP-CIDR,61.160.200.223/32,屏蔽网站
- IP-CIDR,61.160.200.242/32,屏蔽网站
- IP-CIDR,61.160.200.252/32,屏蔽网站
- IP-CIDR,61.174.50.214/32,屏蔽网站
- IP-CIDR,111.175.220.163/32,屏蔽网站
- IP-CIDR,111.175.220.164/32,屏蔽网站
- IP-CIDR,124.232.160.178/32,屏蔽网站
- IP-CIDR,175.6.223.15/32,屏蔽网站
- IP-CIDR,183.59.53.237/32,屏蔽网站
- IP-CIDR,218.93.127.37/32,屏蔽网站
- IP-CIDR,221.228.17.152/32,屏蔽网站
- IP-CIDR,221.231.6.79/32,屏蔽网站
- IP-CIDR,222.186.61.91/32,屏蔽网站
- IP-CIDR,222.186.61.95/32,屏蔽网站
- IP-CIDR,222.186.61.96/32,屏蔽网站
- IP-CIDR,222.186.61.97/32,屏蔽网站
# > ChinaUnicom
- IP-CIDR,106.75.231.48/32,屏蔽网站
- IP-CIDR,119.4.249.166/32,屏蔽网站
- IP-CIDR,220.196.52.141/32,屏蔽网站
- IP-CIDR,221.6.4.148/32,屏蔽网站
# > ChinaMobile
- IP-CIDR,114.247.28.96/32,屏蔽网站
- IP-CIDR,221.179.131.72/32,屏蔽网站
- IP-CIDR,221.179.140.145/32,屏蔽网站
# > Dr.Peng
- IP-CIDR,10.72.25.0/24,屏蔽网站
- IP-CIDR,115.182.16.79/32,屏蔽网站
- IP-CIDR,118.144.88.126/32,屏蔽网站
- IP-CIDR,118.144.88.215/32,屏蔽网站
- IP-CIDR,118.144.88.216/32,屏蔽网站
- IP-CIDR,120.76.189.132/32,屏蔽网站
- IP-CIDR,124.14.21.147/32,屏蔽网站
- IP-CIDR,124.14.21.151/32,屏蔽网站
- IP-CIDR,180.166.52.24/32,屏蔽网站
- IP-CIDR,211.161.101.106/32,屏蔽网站
- IP-CIDR,220.115.251.25/32,屏蔽网站
- IP-CIDR,222.73.156.235/32,屏蔽网站

# Infamous 声名狼藉
- DOMAIN-SUFFIX,kuaizip.com,屏蔽网站
- DOMAIN-SUFFIX,mackeeper.com,屏蔽网站
# > Adobe
- DOMAIN-SUFFIX,flash.cn,屏蔽网站
- DOMAIN,geo2.adobe.com,屏蔽网站
# > CJ Marketing
- DOMAIN-SUFFIX,4009997658.com,屏蔽网站
- DOMAIN-SUFFIX,abbyychina.com,屏蔽网站
- DOMAIN-SUFFIX,bartender.cc,屏蔽网站
- DOMAIN-SUFFIX,betterzip.net,屏蔽网站
- DOMAIN-SUFFIX,beyondcompare.cc,屏蔽网站
- DOMAIN-SUFFIX,bingdianhuanyuan.cn,屏蔽网站
- DOMAIN-SUFFIX,chemdraw.com.cn,屏蔽网站
- DOMAIN-SUFFIX,cjmakeding.com,屏蔽网站
- DOMAIN-SUFFIX,cjmkt.com,屏蔽网站
- DOMAIN-SUFFIX,codesoftchina.com,屏蔽网站
- DOMAIN-SUFFIX,coreldrawchina.com,屏蔽网站
- DOMAIN-SUFFIX,crossoverchina.com,屏蔽网站
- DOMAIN-SUFFIX,easyrecoverychina.com,屏蔽网站
- DOMAIN-SUFFIX,ediuschina.com,屏蔽网站
- DOMAIN-SUFFIX,flstudiochina.com,屏蔽网站
- DOMAIN-SUFFIX,formysql.com,屏蔽网站
- DOMAIN-SUFFIX,guitarpro.cc,屏蔽网站
- DOMAIN-SUFFIX,huishenghuiying.com.cn,屏蔽网站
- DOMAIN-SUFFIX,hypersnap.net,屏蔽网站
- DOMAIN-SUFFIX,iconworkshop.cn,屏蔽网站
- DOMAIN-SUFFIX,imindmap.cc,屏蔽网站
- DOMAIN-SUFFIX,jihehuaban.com.cn,屏蔽网站
- DOMAIN-SUFFIX,keyshot.cc,屏蔽网站
- DOMAIN-SUFFIX,kingdeecn.cn,屏蔽网站
- DOMAIN-SUFFIX,logoshejishi.com,屏蔽网站
- DOMAIN-SUFFIX,mairuan.cn,屏蔽网站
- DOMAIN-SUFFIX,mairuan.com,屏蔽网站
- DOMAIN-SUFFIX,mairuan.com.cn,屏蔽网站
- DOMAIN-SUFFIX,mairuan.net,屏蔽网站
- DOMAIN-SUFFIX,mairuanwang.com,屏蔽网站
- DOMAIN-SUFFIX,makeding.com,屏蔽网站
- DOMAIN-SUFFIX,mathtype.cn,屏蔽网站
- DOMAIN-SUFFIX,mindmanager.cc,屏蔽网站
- DOMAIN-SUFFIX,mindmapper.cc,屏蔽网站
- DOMAIN-SUFFIX,mycleanmymac.com,屏蔽网站
- DOMAIN-SUFFIX,nicelabel.cc,屏蔽网站
- DOMAIN-SUFFIX,ntfsformac.cc,屏蔽网站
- DOMAIN-SUFFIX,ntfsformac.cn,屏蔽网站
- DOMAIN-SUFFIX,overturechina.com,屏蔽网站
- DOMAIN-SUFFIX,passwordrecovery.cn,屏蔽网站
- DOMAIN-SUFFIX,pdfexpert.cc,屏蔽网站
- DOMAIN-SUFFIX,shankejingling.com,屏蔽网站
- DOMAIN-SUFFIX,ultraiso.net,屏蔽网站
- DOMAIN-SUFFIX,vegaschina.cn,屏蔽网站
- DOMAIN-SUFFIX,xmindchina.net,屏蔽网站
- DOMAIN-SUFFIX,xshellcn.com,屏蔽网站
- DOMAIN-SUFFIX,yihuifu.cn,屏蔽网站
- DOMAIN-SUFFIX,yuanchengxiezuo.com,屏蔽网站
- DOMAIN-SUFFIX,zbrushcn.com,屏蔽网站
- DOMAIN-SUFFIX,zhzzx.com,屏蔽网站

# Global Area Network
# (国际媒体)
# (Video)
# > All4
# USER-AGENT,All4*,国际媒体
- DOMAIN-SUFFIX,c4assets.com,国际媒体
- DOMAIN-SUFFIX,channel4.com,国际媒体
# > AbemaTV
# USER-AGENT,AbemaTV*,国际媒体
- DOMAIN-SUFFIX,abema.io,国际媒体
- DOMAIN-SUFFIX,ameba.jp,国际媒体
- DOMAIN-SUFFIX,hayabusa.io,国际媒体
- DOMAIN,abematv.akamaized.net,国际媒体
- DOMAIN,ds-linear-abematv.akamaized.net,国际媒体
- DOMAIN,ds-vod-abematv.akamaized.net,国际媒体
- DOMAIN,linear-abematv.akamaized.net,国际媒体
# > Amazon Prime Video
# USER-AGENT,InstantVideo.US*,国际媒体
# USER-AGENT,Prime%20Video*,国际媒体
- DOMAIN-SUFFIX,primevideo.com,国际媒体
# > Bahamut
# USER-AGENT,Anime*,国际媒体
- DOMAIN-SUFFIX,bahamut.com.tw,国际媒体
- DOMAIN-SUFFIX,gamer.com.tw,国际媒体
- DOMAIN,gamer-cds.cdn.hinet.net,国际媒体
- DOMAIN,gamer2-cds.cdn.hinet.net,国际媒体
# > BBC iPlayer
# USER-AGENT,BBCiPlayer*,国际媒体
- DOMAIN-SUFFIX,bbc.co.uk,国际媒体
- DOMAIN-SUFFIX,bbci.co.uk,国际媒体
- DOMAIN-KEYWORD,bbcfmt,国际媒体
- DOMAIN-KEYWORD,uk-live,国际媒体
# > DAZN
- DOMAIN-SUFFIX,dazn.com,国际媒体
# > encoreTVB
# USER-AGENT,encoreTVB*,国际媒体
- DOMAIN-SUFFIX,encoretvb.com,国际媒体
- DOMAIN,content.jwplatform.com,国际媒体
- DOMAIN,videos-f.jwpsrv.com,国际媒体
# > Fox+ & Fox Now
# USER-AGENT,FOX%20NOW*,国际媒体
# USER-AGENT,FOXPlus*,国际媒体
- DOMAIN-SUFFIX,dashasiafox.akamaized.net,国际媒体
- DOMAIN-SUFFIX,fox.com,国际媒体
- DOMAIN-SUFFIX,foxdcg.com,国际媒体
- DOMAIN-SUFFIX,foxplus.com,国际媒体
- DOMAIN-SUFFIX,staticasiafox.akamaized.net,国际媒体
- DOMAIN-SUFFIX,theplatform.com,国际媒体
- DOMAIN-SUFFIX,uplynk.com,国际媒体
# > HBO Now & HBO GO
# USER-AGENT,HBO%20NOW*,国际媒体
# USER-AGENT,HBO%20GO*,国际媒体
# USER-AGENT,HBOAsia*,国际媒体
- DOMAIN-SUFFIX,hbo.com,国际媒体
- DOMAIN-SUFFIX,hbogo.com,国际媒体
- DOMAIN-SUFFIX,hboasia.com,国际媒体
- DOMAIN-SUFFIX,hbogo.com,国际媒体
- DOMAIN-SUFFIX,hbogoasia.hk,国际媒体
- DOMAIN,44wilhpljf.execute-api.ap-southeast-1.amazonaws.com,国际媒体
- DOMAIN,bcbolthboa-a.akamaihd.net,国际媒体
- DOMAIN,cf-images.ap-southeast-1.prod.boltdns.net,国际媒体
- DOMAIN,manifest.prod.boltdns.net,国际媒体
- DOMAIN,s3-ap-southeast-1.amazonaws.com,国际媒体
# > Hulu
- DOMAIN-SUFFIX,hulu.com,国际媒体
- DOMAIN-SUFFIX,huluim.com,国际媒体
- DOMAIN-SUFFIX,hulustream.com,国际媒体
# > KKTV
# USER-AGENT,KKTV*,国际媒体
# USER-AGENT,com.kktv.ios.kktv*,国际媒体
- DOMAIN-SUFFIX,kktv.com.tw,国际媒体
- DOMAIN-SUFFIX,kktv.me,国际媒体
- DOMAIN,kktv-theater.kk.stream,国际媒体
# > Line TV
# USER-AGENT,LINE%20TV*,国际媒体
- DOMAIN-SUFFIX,linetv.tw,国际媒体
- DOMAIN,d3c7rimkq79yfu.cloudfront.net,国际媒体
# > Hulu(フールー)
- DOMAIN-SUFFIX,happyon.jp,国际媒体
- DOMAIN-SUFFIX,hulu.jp,国际媒体
# > LiTV
- DOMAIN-SUFFIX,litv.tv,国际媒体
- DOMAIN,litvfreemobile-hichannel.cdn.hinet.net,国际媒体
# > myTV_SUPER
# USER-AGENT,mytv*,国际媒体
- DOMAIN-SUFFIX,mytvsuper.com,国际媒体
- DOMAIN-SUFFIX,tvb.com,国际媒体
# > Netflix
# USER-AGENT,Argo*,国际媒体
- DOMAIN-SUFFIX,netflix.com,国际媒体
- DOMAIN-SUFFIX,netflix.net,国际媒体
- DOMAIN-SUFFIX,nflxext.com,国际媒体
- DOMAIN-SUFFIX,nflximg.com,国际媒体
- DOMAIN-SUFFIX,nflximg.net,国际媒体
- DOMAIN-SUFFIX,nflxso.net,国际媒体
- DOMAIN-SUFFIX,nflxvideo.net,国际媒体
- IP-CIDR,23.246.0.0/18,国际媒体
- IP-CIDR,37.77.184.0/21,国际媒体
- IP-CIDR,45.57.0.0/17,国际媒体
- IP-CIDR,64.120.128.0/17,国际媒体
- IP-CIDR,66.197.128.0/17,国际媒体
- IP-CIDR,108.175.32.0/20,国际媒体
- IP-CIDR,192.173.64.0/18,国际媒体
- IP-CIDR,198.38.96.0/19,国际媒体
- IP-CIDR,198.45.48.0/20,国际媒体
# > PBS
# USER-AGENT,PBS*,国际媒体
- DOMAIN-SUFFIX,pbs.org,国际媒体
# > Pornhub
- DOMAIN-SUFFIX,phncdn.com,国际媒体
- DOMAIN-SUFFIX,pornhub.com,国际媒体
- DOMAIN-SUFFIX,pornhubpremium.com,国际媒体
# > Twitch
- DOMAIN-SUFFIX,twitch.tv,国际媒体
- DOMAIN-SUFFIX,twitchcdn.net,国际媒体
- DOMAIN-SUFFIX,ttvnw.net,国际媒体
# > Viu(TV)
# USER-AGENT,Viu*,国际媒体
# USER-AGENT,ViuTV*,国际媒体
- DOMAIN-SUFFIX,viu.com,国际媒体
- DOMAIN-SUFFIX,viu.tv,国际媒体
- DOMAIN,api.viu.now.com,国际媒体
- DOMAIN,d1k2us671qcoau.cloudfront.net,国际媒体
- DOMAIN,d2anahhhmp1ffz.cloudfront.net,国际媒体
- DOMAIN,dfp6rglgjqszk.cloudfront.net,国际媒体
# > Youtube
# USER-AGENT,com.google.ios.youtube*,国际媒体
# USER-AGENT,YouTube*,国际媒体
- DOMAIN-SUFFIX,googlevideo.com,国际媒体
- DOMAIN-SUFFIX,youtube.com,国际媒体
- DOMAIN,youtubei.googleapis.com,国际媒体

# (Music)
# > Deezer
# USER-AGENT,Deezer*,国际媒体
- DOMAIN-SUFFIX,deezer.com,国际媒体
- DOMAIN-SUFFIX,dzcdn.net,国际媒体
# > KKBOX
- DOMAIN-SUFFIX,kkbox.com,国际媒体
- DOMAIN-SUFFIX,kkbox.com.tw,国际媒体
- DOMAIN-SUFFIX,kfs.io,国际媒体
# > JOOX
# USER-AGENT,WeMusic*,国际媒体
# USER-AGENT,JOOX*,国际媒体
- DOMAIN-SUFFIX,joox.com,国际媒体
# > Pandora
# USER-AGENT,Pandora*,国际媒体
- DOMAIN-SUFFIX,pandora.com,国际媒体
# > Spotify
# USER-AGENT,Spotify*,国际媒体
- DOMAIN-SUFFIX,pscdn.co,国际媒体
- DOMAIN-SUFFIX,scdn.co,国际媒体
- DOMAIN-SUFFIX,spotify.com,国际媒体
- DOMAIN-SUFFIX,spoti.fi,国际媒体
- IP-CIDR,35.186.224.47/32,国际媒体
# > TIDAL
# USER-AGENT,TIDAL*,国际媒体
- DOMAIN-SUFFIX,tidal.com,国际媒体

# (国内媒体)
# > 愛奇藝台灣站
- DOMAIN-SUFFIX,iqiyi.com,国内媒体
- DOMAIN-SUFFIX,71.am,国内媒体
# > bilibili
- DOMAIN-SUFFIX,bilibili.com,国内媒体
- DOMAIN,upos-hz-mirrorakam.akamaized.net,国内媒体

# (DNS Cache Pollution Protection)
# > Google
- DOMAIN-SUFFIX,appspot.com,选择模式
- DOMAIN-SUFFIX,blogger.com,选择模式
- DOMAIN-SUFFIX,getoutline.org,选择模式
- DOMAIN-SUFFIX,gvt0.com,选择模式
- DOMAIN-SUFFIX,gvt1.com,选择模式
- DOMAIN-SUFFIX,gvt3.com,选择模式
- DOMAIN-SUFFIX,xn--ngstr-lra8j.com,选择模式
- DOMAIN-KEYWORD,google,选择模式
- DOMAIN-KEYWORD,blogspot,选择模式
# > Microsoft
- DOMAIN-SUFFIX,onedrive.live.com,选择模式
- DOMAIN-SUFFIX,xboxlive.com,选择模式
# > Facebook
- DOMAIN-SUFFIX,cdninstagram.com,选择模式
- DOMAIN-SUFFIX,fb.com,选择模式
- DOMAIN-SUFFIX,fb.me,选择模式
- DOMAIN-SUFFIX,fbaddins.com,选择模式
- DOMAIN-SUFFIX,fbcdn.net,选择模式
- DOMAIN-SUFFIX,fbsbx.com,选择模式
- DOMAIN-SUFFIX,fbworkmail.com,选择模式
- DOMAIN-SUFFIX,instagram.com,选择模式
- DOMAIN-SUFFIX,m.me,选择模式
- DOMAIN-SUFFIX,messenger.com,选择模式
- DOMAIN-SUFFIX,oculus.com,选择模式
- DOMAIN-SUFFIX,oculuscdn.com,选择模式
- DOMAIN-SUFFIX,rocksdb.org,选择模式
- DOMAIN-SUFFIX,whatsapp.com,选择模式
- DOMAIN-SUFFIX,whatsapp.net,选择模式
- DOMAIN-KEYWORD,facebook,选择模式
- IP-CIDR,3.123.36.126/32,选择模式
- IP-CIDR,52.58.209.134/32,选择模式
- IP-CIDR,54.93.124.31/32,选择模式
- IP-CIDR,54.173.34.141/32,选择模式
- IP-CIDR,54.235.23.242/32,选择模式
- IP-CIDR,169.45.248.118/32,选择模式
# > Twitter
- DOMAIN-SUFFIX,pscp.tv,选择模式
- DOMAIN-SUFFIX,periscope.tv,选择模式
- DOMAIN-SUFFIX,t.co,选择模式
- DOMAIN-SUFFIX,twimg.co,选择模式
- DOMAIN-SUFFIX,twimg.com,选择模式
- DOMAIN-SUFFIX,twitpic.com,选择模式
- DOMAIN-SUFFIX,vine.co,选择模式
- DOMAIN-KEYWORD,twitter,选择模式
# > Telegram
- DOMAIN-SUFFIX,t.me,选择模式
- DOMAIN-SUFFIX,tdesktop.com,选择模式
- DOMAIN-SUFFIX,telegra.ph,选择模式
- DOMAIN-SUFFIX,telegram.me,选择模式
- DOMAIN-SUFFIX,telegram.org,选择模式
- IP-CIDR,67.198.55.0/24,选择模式
- IP-CIDR,91.108.4.0/22,选择模式
- IP-CIDR,91.108.8.0/22,选择模式
- IP-CIDR,91.108.12.0/22,选择模式
- IP-CIDR,91.108.16.0/22,选择模式
- IP-CIDR,91.108.56.0/22,选择模式
- IP-CIDR,109.239.140.0/24,选择模式
- IP-CIDR,149.154.160.0/20,选择模式
- IP-CIDR,205.172.60.0/22,选择模式
# > Line
- DOMAIN-SUFFIX,line.me,选择模式
- DOMAIN-SUFFIX,line-apps.com,选择模式
- DOMAIN-SUFFIX,line-scdn.net,选择模式
- DOMAIN-SUFFIX,naver.jp,选择模式
- IP-CIDR,103.2.30.0/23,选择模式
- IP-CIDR,125.209.208.0/20,选择模式
- IP-CIDR,147.92.128.0/17,选择模式
- IP-CIDR,203.104.144.0/21,选择模式
# > Other
- DOMAIN-SUFFIX,4shared.com,选择模式
- DOMAIN-SUFFIX,520cc.cc,选择模式
- DOMAIN-SUFFIX,881903.com,选择模式
- DOMAIN-SUFFIX,9cache.com,选择模式
- DOMAIN-SUFFIX,9gag.com,选择模式
- DOMAIN-SUFFIX,abc.com,选择模式
- DOMAIN-SUFFIX,abc.net.au,选择模式
- DOMAIN-SUFFIX,abebooks.com,选择模式
- DOMAIN-SUFFIX,amazon.co.jp,选择模式
- DOMAIN-SUFFIX,apigee.com,选择模式
- DOMAIN-SUFFIX,apk-dl.com,选择模式
- DOMAIN-SUFFIX,apkfind.com,选择模式
- DOMAIN-SUFFIX,apkmirror.com,选择模式
- DOMAIN-SUFFIX,apkmonk.com,选择模式
- DOMAIN-SUFFIX,apkpure.com,选择模式
- DOMAIN-SUFFIX,aptoide.com,选择模式
- DOMAIN-SUFFIX,archive.is,选择模式
- DOMAIN-SUFFIX,archive.org,选择模式
- DOMAIN-SUFFIX,arte.tv,选择模式
- DOMAIN-SUFFIX,artstation.com,选择模式
- DOMAIN-SUFFIX,arukas.io,选择模式
- DOMAIN-SUFFIX,ask.com,选择模式
- DOMAIN-SUFFIX,avgle.com,选择模式
- DOMAIN-SUFFIX,badoo.com,选择模式
- DOMAIN-SUFFIX,bandwagonhost.com,选择模式
- DOMAIN-SUFFIX,bbc.com,选择模式
- DOMAIN-SUFFIX,behance.net,选择模式
- DOMAIN-SUFFIX,bibox.com,选择模式
- DOMAIN-SUFFIX,biggo.com.tw,选择模式
- DOMAIN-SUFFIX,binance.com,选择模式
- DOMAIN-SUFFIX,bitcointalk.org,选择模式
- DOMAIN-SUFFIX,bitfinex.com,选择模式
- DOMAIN-SUFFIX,bitmex.com,选择模式
- DOMAIN-SUFFIX,bit-z.com,选择模式
- DOMAIN-SUFFIX,bloglovin.com,选择模式
- DOMAIN-SUFFIX,bloomberg.cn,选择模式
- DOMAIN-SUFFIX,bloomberg.com,选择模式
- DOMAIN-SUFFIX,blubrry.com,选择模式
- DOMAIN-SUFFIX,book.com.tw,选择模式
- DOMAIN-SUFFIX,booklive.jp,选择模式
- DOMAIN-SUFFIX,books.com.tw,选择模式
- DOMAIN-SUFFIX,box.com,选择模式
- DOMAIN-SUFFIX,businessinsider.com,选择模式
- DOMAIN-SUFFIX,bwh1.net,选择模式
- DOMAIN-SUFFIX,castbox.fm,选择模式
- DOMAIN-SUFFIX,cbc.ca,选择模式
- DOMAIN-SUFFIX,cdw.com,选择模式
- DOMAIN-SUFFIX,change.org,选择模式
- DOMAIN-SUFFIX,ck101.com,选择模式
- DOMAIN-SUFFIX,clarionproject.org,选择模式
- DOMAIN-SUFFIX,clyp.it,选择模式
- DOMAIN-SUFFIX,cna.com.tw,选择模式
- DOMAIN-SUFFIX,comparitech.com,选择模式
- DOMAIN-SUFFIX,conoha.jp,选择模式
- DOMAIN-SUFFIX,crucial.com,选择模式
- DOMAIN-SUFFIX,cts.com.tw,选择模式
- DOMAIN-SUFFIX,cw.com.tw,选择模式
- DOMAIN-SUFFIX,cyberctm.com,选择模式
- DOMAIN-SUFFIX,dailymotion.com,选择模式
- DOMAIN-SUFFIX,dailyview.tw,选择模式
- DOMAIN-SUFFIX,daum.net,选择模式
- DOMAIN-SUFFIX,daumcdn.net,选择模式
- DOMAIN-SUFFIX,dcard.tw,选择模式
- DOMAIN-SUFFIX,deepdiscount.com,选择模式
- DOMAIN-SUFFIX,depositphotos.com,选择模式
- DOMAIN-SUFFIX,deviantart.com,选择模式
- DOMAIN-SUFFIX,disconnect.me,选择模式
- DOMAIN-SUFFIX,discordapp.com,选择模式
- DOMAIN-SUFFIX,discordapp.net,选择模式
- DOMAIN-SUFFIX,disqus.com,选择模式
- DOMAIN-SUFFIX,dns2go.com,选择模式
- DOMAIN-SUFFIX,dowjones.com,选择模式
- DOMAIN-SUFFIX,dropbox.com,选择模式
- DOMAIN-SUFFIX,dropboxusercontent.com,选择模式
- DOMAIN-SUFFIX,duckduckgo.com,选择模式
- DOMAIN-SUFFIX,dw.com,选择模式
- DOMAIN-SUFFIX,dynu.com,选择模式
- DOMAIN-SUFFIX,earthcam.com,选择模式
- DOMAIN-SUFFIX,ebookservice.tw,选择模式
- DOMAIN-SUFFIX,economist.com,选择模式
- DOMAIN-SUFFIX,edgecastcdn.net,选择模式
- DOMAIN-SUFFIX,edu,选择模式
- DOMAIN-SUFFIX,elpais.com,选择模式
- DOMAIN-SUFFIX,enanyang.my,选择模式
- DOMAIN-SUFFIX,encyclopedia.com,选择模式
- DOMAIN-SUFFIX,esoir.be,选择模式
- DOMAIN-SUFFIX,euronews.com,选择模式
- DOMAIN-SUFFIX,feedly.com,选择模式
- DOMAIN-SUFFIX,firech.at,选择模式
- DOMAIN-SUFFIX,flickr.com,选择模式
- DOMAIN-SUFFIX,flitto.com,选择模式
- DOMAIN-SUFFIX,foreignpolicy.com,选择模式
- DOMAIN-SUFFIX,friday.tw,选择模式
- DOMAIN-SUFFIX,ftchinese.com,选择模式
- DOMAIN-SUFFIX,ftimg.net,选择模式
- DOMAIN-SUFFIX,gate.io,选择模式
- DOMAIN-SUFFIX,getlantern.org,选择模式
- DOMAIN-SUFFIX,getsync.com,选择模式
- DOMAIN-SUFFIX,globalvoices.org,选择模式
- DOMAIN-SUFFIX,goo.ne.jp,选择模式
- DOMAIN-SUFFIX,goodreads.com,选择模式
- DOMAIN-SUFFIX,gov,选择模式
- DOMAIN-SUFFIX,gov.tw,选择模式
- DOMAIN-SUFFIX,gumroad.com,选择模式
- DOMAIN-SUFFIX,hbg.com,选择模式
- DOMAIN-SUFFIX,heroku.com,选择模式
- DOMAIN-SUFFIX,hightail.com,选择模式
- DOMAIN-SUFFIX,hk01.com,选择模式
- DOMAIN-SUFFIX,hkbf.org,选择模式
- DOMAIN-SUFFIX,hkbookcity.com,选择模式
- DOMAIN-SUFFIX,hkej.com,选择模式
- DOMAIN-SUFFIX,hket.com,选择模式
- DOMAIN-SUFFIX,hkgolden.com,选择模式
- DOMAIN-SUFFIX,hootsuite.com,选择模式
- DOMAIN-SUFFIX,hudson.org,选择模式
- DOMAIN-SUFFIX,hyread.com.tw,选择模式
- DOMAIN-SUFFIX,ibtimes.com,选择模式
- DOMAIN-SUFFIX,i-cable.com,选择模式
- DOMAIN-SUFFIX,icij.org,选择模式
- DOMAIN-SUFFIX,icoco.com,选择模式
- DOMAIN-SUFFIX,imgur.com,选择模式
- DOMAIN-SUFFIX,initiummall.com,选择模式
- DOMAIN-SUFFIX,insecam.org,选择模式
- DOMAIN-SUFFIX,ipfs.io,选择模式
- DOMAIN-SUFFIX,issuu.com,选择模式
- DOMAIN-SUFFIX,istockphoto.com,选择模式
- DOMAIN-SUFFIX,japantimes.co.jp,选择模式
- DOMAIN-SUFFIX,jiji.com,选择模式
- DOMAIN-SUFFIX,jinx.com,选择模式
- DOMAIN-SUFFIX,jkforum.net,选择模式
- DOMAIN-SUFFIX,joinmastodon.org,选择模式
- DOMAIN-SUFFIX,justpaste.it,选择模式
- DOMAIN-SUFFIX,kakao.com,选择模式
- DOMAIN-SUFFIX,kakaocorp.com,选择模式
- DOMAIN-SUFFIX,kik.com,选择模式
- DOMAIN-SUFFIX,kobo.com,选择模式
- DOMAIN-SUFFIX,kobobooks.com,选择模式
- DOMAIN-SUFFIX,kodingen.com,选择模式
- DOMAIN-SUFFIX,lemonde.fr,选择模式
- DOMAIN-SUFFIX,lepoint.fr,选择模式
- DOMAIN-SUFFIX,lihkg.com,选择模式
- DOMAIN-SUFFIX,listennotes.com,选择模式
- DOMAIN-SUFFIX,livestream.com,选择模式
- DOMAIN-SUFFIX,logmein.com,选择模式
- DOMAIN-SUFFIX,mail.ru,选择模式
- DOMAIN-SUFFIX,mailchimp.com,选择模式
- DOMAIN-SUFFIX,marc.info,选择模式
- DOMAIN-SUFFIX,matters.news,选择模式
- DOMAIN-SUFFIX,medium.com,选择模式
- DOMAIN-SUFFIX,mega.nz,选择模式
- DOMAIN-SUFFIX,mil,选择模式
- DOMAIN-SUFFIX,mingpao.com,选择模式
- DOMAIN-SUFFIX,mobile01.com,选择模式
- DOMAIN-SUFFIX,myspace.com,选择模式
- DOMAIN-SUFFIX,myspacecdn.com,选择模式
- DOMAIN-SUFFIX,nanyang.com,选择模式
- DOMAIN-SUFFIX,naver.com,选择模式
- DOMAIN-SUFFIX,newstapa.org,选择模式
- DOMAIN-SUFFIX,nhk.or.jp,选择模式
- DOMAIN-SUFFIX,nicovideo.jp,选择模式
- DOMAIN-SUFFIX,nii.ac.jp,选择模式
- DOMAIN-SUFFIX,nikkei.com,选择模式
- DOMAIN-SUFFIX,nofile.io,选择模式
- DOMAIN-SUFFIX,now.com,选择模式
- DOMAIN-SUFFIX,nrk.no,选择模式
- DOMAIN-SUFFIX,nyt.com,选择模式
- DOMAIN-SUFFIX,nytchina.com,选择模式
- DOMAIN-SUFFIX,nytcn.me,选择模式
- DOMAIN-SUFFIX,nytco.com,选择模式
- DOMAIN-SUFFIX,nytimes.com,选择模式
- DOMAIN-SUFFIX,nytimg.com,选择模式
- DOMAIN-SUFFIX,nytlog.com,选择模式
- DOMAIN-SUFFIX,nytstyle.com,选择模式
- DOMAIN-SUFFIX,ok.ru,选择模式
- DOMAIN-SUFFIX,okex.com,选择模式
- DOMAIN-SUFFIX,on.cc,选择模式
- DOMAIN-SUFFIX,orientaldaily.com.my,选择模式
- DOMAIN-SUFFIX,overcast.fm,选择模式
- DOMAIN-SUFFIX,paltalk.com,选择模式
- DOMAIN-SUFFIX,pbxes.com,选择模式
- DOMAIN-SUFFIX,pcdvd.com.tw,选择模式
- DOMAIN-SUFFIX,pchome.com.tw,选择模式
- DOMAIN-SUFFIX,pcloud.com,选择模式
- DOMAIN-SUFFIX,picacomic.com,选择模式
- DOMAIN-SUFFIX,pinimg.com,选择模式
- DOMAIN-SUFFIX,pixiv.net,选择模式
- DOMAIN-SUFFIX,player.fm,选择模式
- DOMAIN-SUFFIX,plurk.com,选择模式
- DOMAIN-SUFFIX,po18.tw,选择模式
- DOMAIN-SUFFIX,prism-break.org,选择模式
- DOMAIN-SUFFIX,proxifier.com,选择模式
- DOMAIN-SUFFIX,pts.org.tw,选择模式
- DOMAIN-SUFFIX,pubu.com.tw,选择模式
- DOMAIN-SUFFIX,pubu.tw,选择模式
- DOMAIN-SUFFIX,pureapk.com,选择模式
- DOMAIN-SUFFIX,quora.com,选择模式
- DOMAIN-SUFFIX,quoracdn.net,选择模式
- DOMAIN-SUFFIX,rakuten.co.jp,选择模式
- DOMAIN-SUFFIX,readingtimes.com.tw,选择模式
- DOMAIN-SUFFIX,readmoo.com,选择模式
- DOMAIN-SUFFIX,reddit.com,选择模式
- DOMAIN-SUFFIX,redditmedia.com,选择模式
- DOMAIN-SUFFIX,resilio.com,选择模式
- DOMAIN-SUFFIX,reuters.com,选择模式
- DOMAIN-SUFFIX,rfi.fr,选择模式
- DOMAIN-SUFFIX,roadshow.hk,选择模式
- DOMAIN-SUFFIX,scmp.com,选择模式
- DOMAIN-SUFFIX,scribd.com,选择模式
- DOMAIN-SUFFIX,seatguru.com,选择模式
- DOMAIN-SUFFIX,shadowsocks.org,选择模式
- DOMAIN-SUFFIX,shopee.tw,选择模式
- DOMAIN-SUFFIX,slideshare.net,选择模式
- DOMAIN-SUFFIX,softfamous.com,选择模式
- DOMAIN-SUFFIX,soundcloud.com,选择模式
- DOMAIN-SUFFIX,startpage.com,选择模式
- DOMAIN-SUFFIX,steamcommunity.com,选择模式
- DOMAIN-SUFFIX,steemit.com,选择模式
- DOMAIN-SUFFIX,steemitwallet.com,选择模式
- DOMAIN-SUFFIX,t66y.com,选择模式
- DOMAIN-SUFFIX,tapatalk.com,选择模式
- DOMAIN-SUFFIX,teco-hk.org,选择模式
- DOMAIN-SUFFIX,teco-mo.org,选择模式
- DOMAIN-SUFFIX,teddysun.com,选择模式
- DOMAIN-SUFFIX,theguardian.com,选择模式
- DOMAIN-SUFFIX,theinitium.com,选择模式
- DOMAIN-SUFFIX,tineye.com,选择模式
- DOMAIN-SUFFIX,torproject.org,选择模式
- DOMAIN-SUFFIX,tumblr.com,选择模式
- DOMAIN-SUFFIX,turbobit.net,选择模式
- DOMAIN-SUFFIX,tutanota.com,选择模式
- DOMAIN-SUFFIX,tvboxnow.com,选择模式
- DOMAIN-SUFFIX,udn.com,选择模式
- DOMAIN-SUFFIX,unseen.is,选择模式
- DOMAIN-SUFFIX,upmedia.mg,选择模式
- DOMAIN-SUFFIX,uptodown.com,选择模式
- DOMAIN-SUFFIX,ustream.tv,选择模式
- DOMAIN-SUFFIX,uwants.com,选择模式
- DOMAIN-SUFFIX,v2ray.com,选择模式
- DOMAIN-SUFFIX,viber.com,选择模式
- DOMAIN-SUFFIX,videopress.com,选择模式
- DOMAIN-SUFFIX,vimeo.com,选择模式
- DOMAIN-SUFFIX,voachinese.com,选择模式
- DOMAIN-SUFFIX,voanews.com,选择模式
- DOMAIN-SUFFIX,voxer.com,选择模式
- DOMAIN-SUFFIX,vzw.com,选择模式
- DOMAIN-SUFFIX,w3schools.com,选择模式
- DOMAIN-SUFFIX,washingtonpost.com,选择模式
- DOMAIN-SUFFIX,wattpad.com,选择模式
- DOMAIN-SUFFIX,whoer.net,选择模式
- DOMAIN-SUFFIX,wikimapia.org,选择模式
- DOMAIN-SUFFIX,wikipedia.org,选择模式
- DOMAIN-SUFFIX,winudf.com,选择模式
- DOMAIN-SUFFIX,wire.com,选择模式
- DOMAIN-SUFFIX,wordpress.com,选择模式
- DOMAIN-SUFFIX,workflow.is,选择模式
- DOMAIN-SUFFIX,worldcat.org,选择模式
- DOMAIN-SUFFIX,wsj.com,选择模式
- DOMAIN-SUFFIX,wsj.net,选择模式
- DOMAIN-SUFFIX,xhamster.com,选择模式
- DOMAIN-SUFFIX,xnxx.com,选择模式
- DOMAIN-SUFFIX,xvideos.com,选择模式
- DOMAIN-SUFFIX,yahoo.com,选择模式
- DOMAIN-SUFFIX,yandex.ru,选择模式
- DOMAIN-SUFFIX,ycombinator.com,选择模式
- DOMAIN-SUFFIX,yesasia.com,选择模式
- DOMAIN-SUFFIX,yes-news.com,选择模式
- DOMAIN-SUFFIX,yomiuri.co.jp,选择模式
- DOMAIN-SUFFIX,you-get.org,选择模式
- DOMAIN-SUFFIX,zaobao.com,选择模式
- DOMAIN-SUFFIX,zb.com,选择模式
- DOMAIN-SUFFIX,zello.com,选择模式
- DOMAIN-SUFFIX,zeronet.io,选择模式
- DOMAIN-SUFFIX,zoom.us,选择模式
- DOMAIN-KEYWORD,github,选择模式
- DOMAIN-KEYWORD,jav,选择模式
- DOMAIN-KEYWORD,pinterest,选择模式
- DOMAIN-KEYWORD,porn,选择模式
- DOMAIN-KEYWORD,wikileaks,选择模式

# (Region-Restricted Access Denied)
- DOMAIN-SUFFIX,apartmentratings.com,选择模式
- DOMAIN-SUFFIX,apartments.com,选择模式
- DOMAIN-SUFFIX,bankmobilevibe.com,选择模式
- DOMAIN-SUFFIX,bing.com,选择模式
- DOMAIN-SUFFIX,booktopia.com.au,选择模式
- DOMAIN-SUFFIX,cccat.io,选择模式
- DOMAIN-SUFFIX,centauro.com.br,选择模式
- DOMAIN-SUFFIX,clearsurance.com,选择模式
- DOMAIN-SUFFIX,costco.com,选择模式
- DOMAIN-SUFFIX,crackle.com,选择模式
- DOMAIN-SUFFIX,depositphotos.cn,选择模式
- DOMAIN-SUFFIX,dish.com,选择模式
- DOMAIN-SUFFIX,dmm.co.jp,选择模式
- DOMAIN-SUFFIX,dmm.com,选择模式
- DOMAIN-SUFFIX,dnvod.tv,选择模式
- DOMAIN-SUFFIX,esurance.com,选择模式
- DOMAIN-SUFFIX,extmatrix.com,选择模式
- DOMAIN-SUFFIX,fastpic.ru,选择模式
- DOMAIN-SUFFIX,flipboard.com,选择模式
- DOMAIN-SUFFIX,fnac.be,选择模式
- DOMAIN-SUFFIX,fnac.com,选择模式
- DOMAIN-SUFFIX,funkyimg.com,选择模式
- DOMAIN-SUFFIX,fxnetworks.com,选择模式
- DOMAIN-SUFFIX,gettyimages.com,选择模式
- DOMAIN-SUFFIX,go.com,选择模式
- DOMAIN-SUFFIX,here.com,选择模式
- DOMAIN-SUFFIX,jcpenney.com,选择模式
- DOMAIN-SUFFIX,jiehua.tv,选择模式
- DOMAIN-SUFFIX,mailfence.com,选择模式
- DOMAIN-SUFFIX,nationwide.com,选择模式
- DOMAIN-SUFFIX,nbc.com,选择模式
- DOMAIN-SUFFIX,nexon.com,选择模式
- DOMAIN-SUFFIX,nordstrom.com,选择模式
- DOMAIN-SUFFIX,nordstromimage.com,选择模式
- DOMAIN-SUFFIX,nordstromrack.com,选择模式
- DOMAIN-SUFFIX,superpages.com,选择模式
- DOMAIN-SUFFIX,target.com,选择模式
- DOMAIN-SUFFIX,thinkgeek.com,选择模式
- DOMAIN-SUFFIX,tracfone.com,选择模式
- DOMAIN-SUFFIX,uploader.jp,选择模式
- DOMAIN-SUFFIX,vevo.com,选择模式
- DOMAIN-SUFFIX,viu.tv,选择模式
- DOMAIN-SUFFIX,vk.com,选择模式
- DOMAIN-SUFFIX,vsco.co,选择模式
- DOMAIN-SUFFIX,xfinity.com,选择模式
- DOMAIN-SUFFIX,zattoo.com,选择模式
# USER-AGENT,Roam*,选择模式

# (The Most Popular Sites)
# > Apple服务
# >> TestFlight
- DOMAIN,testflight.Apple服务.com,选择模式
# >> Apple服务 URL Shortener
- DOMAIN-SUFFIX,appsto.re,选择模式
# >> iBooks Store download
- DOMAIN,books.itunes.Apple服务.com,选择模式
# >> iTunes Store Moveis Trailers
- DOMAIN,hls.itunes.Apple服务.com,选择模式
# >> App Store Preview
- DOMAIN,apps.Apple服务.com,选择模式
- DOMAIN,itunes.Apple服务.com,选择模式
# >> Spotlight
- DOMAIN,api-glb-sea.smoot.Apple服务.com,选择模式
# >> Dictionary
- DOMAIN,lookup-api.Apple服务.com,选择模式
# >> Apple服务 News and Apple服务 Map TOMTOM Version
- DOMAIN,gspe1-ssl.ls.Apple服务.com,选择模式
# USER-AGENT,Apple服务News*,选择模式
# USER-AGENT,com.Apple服务.news*,选择模式
- DOMAIN-SUFFIX,Apple服务.news,选择模式
- DOMAIN,news-client.Apple服务.com,选择模式
- DOMAIN,news-edge.Apple服务.com,选择模式
- DOMAIN,news-events.Apple服务.com,选择模式
- DOMAIN,Apple服务.comscoreresearch.com,选择模式
# > Google
- DOMAIN-SUFFIX,abc.xyz,选择模式
- DOMAIN-SUFFIX,android.com,选择模式
- DOMAIN-SUFFIX,androidify.com,选择模式
- DOMAIN-SUFFIX,dialogflow.com,选择模式
- DOMAIN-SUFFIX,延迟最低draw.com,选择模式
- DOMAIN-SUFFIX,capitalg.com,选择模式
- DOMAIN-SUFFIX,certificate-transparency.org,选择模式
- DOMAIN-SUFFIX,chrome.com,选择模式
- DOMAIN-SUFFIX,chromeexperiments.com,选择模式
- DOMAIN-SUFFIX,chromestatus.com,选择模式
- DOMAIN-SUFFIX,chromium.org,选择模式
- DOMAIN-SUFFIX,creativelab5.com,选择模式
- DOMAIN-SUFFIX,debug.com,选择模式
- DOMAIN-SUFFIX,deepmind.com,选择模式
- DOMAIN-SUFFIX,firebaseio.com,选择模式
- DOMAIN-SUFFIX,getmdl.io,选择模式
- DOMAIN-SUFFIX,ggpht.com,选择模式
- DOMAIN-SUFFIX,gmail.com,选择模式
- DOMAIN-SUFFIX,gmodules.com,选择模式
- DOMAIN-SUFFIX,godoc.org,选择模式
- DOMAIN-SUFFIX,golang.org,选择模式
- DOMAIN-SUFFIX,gstatic.com,选择模式
- DOMAIN-SUFFIX,gv.com,选择模式
- DOMAIN-SUFFIX,gwtproject.org,选择模式
- DOMAIN-SUFFIX,itasoftware.com,选择模式
- DOMAIN-SUFFIX,madewithcode.com,选择模式
- DOMAIN-SUFFIX,material.io,选择模式
- DOMAIN-SUFFIX,polymer-project.org,选择模式
- DOMAIN-SUFFIX,admin.recaptcha.net,选择模式
- DOMAIN-SUFFIX,recaptcha.net,选择模式
- DOMAIN-SUFFIX,shattered.io,选择模式
- DOMAIN-SUFFIX,synergyse.com,选择模式
- DOMAIN-SUFFIX,tensorflow.org,选择模式
- DOMAIN-SUFFIX,tfhub.dev,选择模式
- DOMAIN-SUFFIX,tiltbrush.com,选择模式
- DOMAIN-SUFFIX,waveprotocol.org,选择模式
- DOMAIN-SUFFIX,waymo.com,选择模式
- DOMAIN-SUFFIX,webmproject.org,选择模式
- DOMAIN-SUFFIX,webrtc.org,选择模式
- DOMAIN-SUFFIX,whatbrowser.org,选择模式
- DOMAIN-SUFFIX,widevine.com,选择模式
- DOMAIN-SUFFIX,x.company,选择模式
- DOMAIN-SUFFIX,youtu.be,选择模式
- DOMAIN-SUFFIX,yt.be,选择模式
- DOMAIN-SUFFIX,ytimg.com,选择模式
# > Microsoft
# >> Microsoft OneDrive
- DOMAIN-SUFFIX,1drv.com,选择模式
- DOMAIN-SUFFIX,1drv.ms,选择模式
- DOMAIN-SUFFIX,blob.core.windows.net,选择模式
- DOMAIN-SUFFIX,livefilestore.com,选择模式
- DOMAIN-SUFFIX,onedrive.com,选择模式
- DOMAIN-SUFFIX,storage.live.com,选择模式
- DOMAIN-SUFFIX,storage.msn.com,选择模式
- DOMAIN,oneclient.sfx.ms,选择模式
# > Other
- DOMAIN-SUFFIX,0rz.tw,选择模式
- DOMAIN-SUFFIX,4bluestones.biz,选择模式
- DOMAIN-SUFFIX,9bis.net,选择模式
- DOMAIN-SUFFIX,allconnected.co,选择模式
- DOMAIN-SUFFIX,amazonaws.com,选择模式
- DOMAIN-SUFFIX,aol.com,选择模式
- DOMAIN-SUFFIX,bcc.com.tw,选择模式
- DOMAIN-SUFFIX,bit.ly,选择模式
- DOMAIN-SUFFIX,bitshare.com,选择模式
- DOMAIN-SUFFIX,blog.jp,选择模式
- DOMAIN-SUFFIX,blogimg.jp,选择模式
- DOMAIN-SUFFIX,blogtd.org,选择模式
- DOMAIN-SUFFIX,broadcast.co.nz,选择模式
- DOMAIN-SUFFIX,camfrog.com,选择模式
- DOMAIN-SUFFIX,cfos.de,选择模式
- DOMAIN-SUFFIX,citypopulation.de,选择模式
- DOMAIN-SUFFIX,cloudfront.net,选择模式
- DOMAIN-SUFFIX,ctitv.com.tw,选择模式
- DOMAIN-SUFFIX,cuhk.edu.hk,选择模式
- DOMAIN-SUFFIX,cusu.hk,选择模式
- DOMAIN-SUFFIX,discuss.com.hk,选择模式
- DOMAIN-SUFFIX,dropboxapi.com,选择模式
- DOMAIN-SUFFIX,duolingo.cn,选择模式
- DOMAIN-SUFFIX,edditstatic.com,选择模式
- DOMAIN-SUFFIX,flickriver.com,选择模式
- DOMAIN-SUFFIX,focustaiwan.tw,选择模式
- DOMAIN-SUFFIX,free.fr,选择模式
- DOMAIN-SUFFIX,gigacircle.com,选择模式
- DOMAIN-SUFFIX,hk-pub.com,选择模式
- DOMAIN-SUFFIX,hosting.co.uk,选择模式
- DOMAIN-SUFFIX,hwcdn.net,选择模式
- DOMAIN-SUFFIX,ifixit.com,选择模式
- DOMAIN-SUFFIX,iphone4hongkong.com,选择模式
- DOMAIN-SUFFIX,iphonetaiwan.org,选择模式
- DOMAIN-SUFFIX,iptvbin.com,选择模式
- DOMAIN-SUFFIX,jtvnw.net,选择模式
- DOMAIN-SUFFIX,linksalpha.com,选择模式
- DOMAIN-SUFFIX,manyvids.com,选择模式
- DOMAIN-SUFFIX,myactimes.com,选择模式
- DOMAIN-SUFFIX,newsblur.com,选择模式
- DOMAIN-SUFFIX,now.im,选择模式
- DOMAIN-SUFFIX,nowe.com,选择模式
- DOMAIN-SUFFIX,redditlist.com,选择模式
- DOMAIN-SUFFIX,signal.org,选择模式
- DOMAIN-SUFFIX,smartmailcloud.com,选择模式
- DOMAIN-SUFFIX,sparknotes.com,选择模式
- DOMAIN-SUFFIX,streetvoice.com,选择模式
- DOMAIN-SUFFIX,supertop.co,选择模式
- DOMAIN-SUFFIX,tv.com,选择模式
- DOMAIN-SUFFIX,typepad.com,选择模式
- DOMAIN-SUFFIX,udnbkk.com,选择模式
- DOMAIN-SUFFIX,urbanairship.com,选择模式
- DOMAIN-SUFFIX,whispersystems.org,选择模式
- DOMAIN-SUFFIX,wikia.com,选择模式
- DOMAIN-SUFFIX,wn.com,选择模式
- DOMAIN-SUFFIX,wolframalpha.com,选择模式
- DOMAIN-SUFFIX,x-art.com,选择模式
- DOMAIN-SUFFIX,yimg.com,选择模式
- DOMAIN,api.steampowered.com,选择模式
- DOMAIN,store.steampowered.com,选择模式

# China Area Network
# > 360
- DOMAIN-SUFFIX,qhres.com,DIRECT
- DOMAIN-SUFFIX,qhimg.com,DIRECT
# > Akamai
- DOMAIN-SUFFIX,akadns.net,DIRECT
# - DOMAIN-SUFFIX,akamai.net,DIRECT
# - DOMAIN-SUFFIX,akamaiedge.net,DIRECT
# - DOMAIN-SUFFIX,akamaihd.net,DIRECT
# - DOMAIN-SUFFIX,akamaistream.net,DIRECT
# - DOMAIN-SUFFIX,akamaized.net,DIRECT
# > Alibaba
# USER-AGENT,%E4%BC%98%E9%85%B7*,DIRECT
- DOMAIN-SUFFIX,alibaba.com,DIRECT
- DOMAIN-SUFFIX,alicdn.com,DIRECT
- DOMAIN-SUFFIX,alikunlun.com,DIRECT
- DOMAIN-SUFFIX,alipay.com,DIRECT
- DOMAIN-SUFFIX,amap.com,DIRECT
- DOMAIN-SUFFIX,dingtalk.com,DIRECT
- DOMAIN-SUFFIX,mxhichina.com,DIRECT
- DOMAIN-SUFFIX,soku.com,DIRECT
- DOMAIN-SUFFIX,taobao.com,DIRECT
- DOMAIN-SUFFIX,tmall.com,DIRECT
- DOMAIN-SUFFIX,tmall.hk,DIRECT
- DOMAIN-SUFFIX,ykimg.com,DIRECT
- DOMAIN-SUFFIX,youku.com,DIRECT
- DOMAIN-SUFFIX,xiami.com,DIRECT
- DOMAIN-SUFFIX,xiami.net,DIRECT
# > Apple服务
- DOMAIN-SUFFIX,aaplimg.com,Apple服务
- DOMAIN-SUFFIX,Apple服务.co,Apple服务
- DOMAIN-SUFFIX,Apple服务.com,Apple服务
- DOMAIN-SUFFIX,appstore.com,Apple服务
- DOMAIN-SUFFIX,cdn-Apple服务.com,Apple服务
- DOMAIN-SUFFIX,crashlytics.com,Apple服务
- DOMAIN-SUFFIX,icloud.com,Apple服务
- DOMAIN-SUFFIX,icloud-content.com,Apple服务
- DOMAIN-SUFFIX,me.com,Apple服务
- DOMAIN-SUFFIX,mzstatic.com,Apple服务
- DOMAIN,www-cdn.icloud.com.akadns.net,Apple服务
- IP-CIDR,17.0.0.0/8,Apple服务
# > Baidu
- DOMAIN-SUFFIX,baidu.com,DIRECT
- DOMAIN-SUFFIX,baidubcr.com,DIRECT
- DOMAIN-SUFFIX,bdstatic.com,DIRECT
- DOMAIN-SUFFIX,yunjiasu-cdn.net,DIRECT
# > bilibili
- DOMAIN-SUFFIX,acgvideo.com,DIRECT
- DOMAIN-SUFFIX,biliapi.com,DIRECT
- DOMAIN-SUFFIX,biliapi.net,DIRECT
- DOMAIN-SUFFIX,bilibili.com,DIRECT
- DOMAIN-SUFFIX,hdslb.com,DIRECT
# > Blizzard
- DOMAIN-SUFFIX,blizzard.com,DIRECT
- DOMAIN-SUFFIX,battle.net,DIRECT
- DOMAIN,blzddist1-a.akamaihd.net,DIRECT
# > ByteDance
- DOMAIN-SUFFIX,feiliao.com,DIRECT
- DOMAIN-SUFFIX,pstatp.com,DIRECT
- DOMAIN-SUFFIX,snssdk.com,DIRECT
- DOMAIN-SUFFIX,iesdouyin.com,DIRECT
- DOMAIN-SUFFIX,toutiao.com,DIRECT
# > DiDi
- DOMAIN-SUFFIX,didialift.com,DIRECT
- DOMAIN-SUFFIX,didiglobal.com,DIRECT
- DOMAIN-SUFFIX,udache.com,DIRECT
# > 蛋蛋赞
- DOMAIN-SUFFIX,343480.com,DIRECT
- DOMAIN-SUFFIX,baduziyuan.com,DIRECT
- DOMAIN-SUFFIX,com-hs-hkdy.com,DIRECT
- DOMAIN-SUFFIX,czybjz.com,DIRECT
- DOMAIN-SUFFIX,dandanzan.com,DIRECT
- DOMAIN-SUFFIX,fjhps.com,DIRECT
- DOMAIN-SUFFIX,kuyunbo.club,DIRECT
# > ChinaNet
- DOMAIN-SUFFIX,21cn.com,DIRECT
# > HunanTV
- DOMAIN-SUFFIX,hitv.com,DIRECT
- DOMAIN-SUFFIX,mgtv.com,DIRECT
# > iQiyi
- DOMAIN-SUFFIX,iqiyi.com,DIRECT
- DOMAIN-SUFFIX,iqiyipic.com,DIRECT
- DOMAIN-SUFFIX,71.am.com,DIRECT
# > JD
- DOMAIN-SUFFIX,jd.com,DIRECT
- DOMAIN-SUFFIX,jd.hk,DIRECT
- DOMAIN-SUFFIX,360buyimg.com,DIRECT
# > Kingsoft
- DOMAIN-SUFFIX,iciba.com,DIRECT
- DOMAIN-SUFFIX,ksosoft.com,DIRECT
# > Meitu
- DOMAIN-SUFFIX,meitu.com,DIRECT
- DOMAIN-SUFFIX,meitudata.com,DIRECT
- DOMAIN-SUFFIX,meitustat.com,DIRECT
- DOMAIN-SUFFIX,meipai.com,DIRECT
# > MI
- DOMAIN-SUFFIX,duokan.com,DIRECT
- DOMAIN-SUFFIX,mi-img.com,DIRECT
- DOMAIN-SUFFIX,miui.com,DIRECT
- DOMAIN-SUFFIX,miwifi.com,DIRECT
- DOMAIN-SUFFIX,xiaomi.com,DIRECT
# > Microsoft
- DOMAIN-SUFFIX,microsoft.com,DIRECT
- DOMAIN-SUFFIX,msecnd.net,DIRECT
- DOMAIN-SUFFIX,office365.com,DIRECT
- DOMAIN-SUFFIX,outlook.com,DIRECT
- DOMAIN-SUFFIX,visualstudio.com,DIRECT
- DOMAIN-SUFFIX,windows.com,DIRECT
- DOMAIN-SUFFIX,windowsupdate.com,DIRECT
- DOMAIN,officecdn-microsoft-com.akamaized.net,DIRECT
# > NetEase
# USER-AGENT,NeteaseMusic*,DIRECT
# USER-AGENT,%E7%BD%91%E6%98%93%E4%BA%91%E9%9F%B3%E4%B9%90*,DIRECT
- DOMAIN-SUFFIX,163.com,DIRECT
- DOMAIN-SUFFIX,126.net,DIRECT
- DOMAIN-SUFFIX,127.net,DIRECT
- DOMAIN-SUFFIX,163yun.com,DIRECT
- DOMAIN-SUFFIX,lofter.com,DIRECT
- DOMAIN-SUFFIX,netease.com,DIRECT
- DOMAIN-SUFFIX,ydstatic.com,DIRECT
# > Sina
- DOMAIN-SUFFIX,sina.com,DIRECT
- DOMAIN-SUFFIX,weibo.com,DIRECT
- DOMAIN-SUFFIX,weibocdn.com,DIRECT
# > Sohu
- DOMAIN-SUFFIX,sohu.com,DIRECT
- DOMAIN-SUFFIX,sohucs.com,DIRECT
- DOMAIN-SUFFIX,sohu-inc.com,DIRECT
- DOMAIN-SUFFIX,v-56.com,DIRECT
# > Sogo
- DOMAIN-SUFFIX,sogo.com,DIRECT
- DOMAIN-SUFFIX,sogou.com,DIRECT
- DOMAIN-SUFFIX,sogoucdn.com,DIRECT
# > Steam
- DOMAIN-SUFFIX,steampowered.com,DIRECT
- DOMAIN-SUFFIX,steam-chat.com,DIRECT
- DOMAIN-SUFFIX,steamgames.com,DIRECT
- DOMAIN-SUFFIX,steamusercontent.com,DIRECT
- DOMAIN-SUFFIX,steamcontent.com,DIRECT
- DOMAIN-SUFFIX,steamstatic.com,DIRECT
- DOMAIN-SUFFIX,steamstat.us,DIRECT
# > Tencent
# USER-AGENT,MicroMessenger%20Client,DIRECT
# USER-AGENT,WeChat*,DIRECT
- DOMAIN-SUFFIX,gtimg.com,DIRECT
- DOMAIN-SUFFIX,myqcloud.com,DIRECT
- DOMAIN-SUFFIX,qq.com,DIRECT
- DOMAIN-SUFFIX,tencent.com,DIRECT
# > YYeTs
# USER-AGENT,YYeTs*,DIRECT
- DOMAIN-SUFFIX,jstucdn.com,DIRECT
- DOMAIN-SUFFIX,zimuzu.io,DIRECT
- DOMAIN-SUFFIX,zimuzu.tv,DIRECT
- DOMAIN-SUFFIX,zmz2019.com,DIRECT
- DOMAIN-SUFFIX,zmzapi.com,DIRECT
- DOMAIN-SUFFIX,zmzapi.net,DIRECT
- DOMAIN-SUFFIX,zmzfile.com,DIRECT
# > Content Delivery Network
- DOMAIN-SUFFIX,ccgslb.com,DIRECT
- DOMAIN-SUFFIX,ccgslb.net,DIRECT
- DOMAIN-SUFFIX,chinanetcenter.com,DIRECT
- DOMAIN-SUFFIX,meixincdn.com,DIRECT
- DOMAIN-SUFFIX,ourdvs.com,DIRECT
- DOMAIN-SUFFIX,wangsu.com,DIRECT
# > IP Query
- DOMAIN-SUFFIX,ipip.net,DIRECT
- DOMAIN-SUFFIX,ip.la,DIRECT
- DOMAIN-SUFFIX,ip-cdn.com,DIRECT
- DOMAIN-SUFFIX,ipv6-test.com,DIRECT
- DOMAIN-SUFFIX,test-ipv6.com,DIRECT
- DOMAIN-SUFFIX,whatismyip.com,DIRECT
# > Speed Test
# - DOMAIN-SUFFIX,speedtest.net,DIRECT
- DOMAIN-SUFFIX,netspeedtestmaster.com,DIRECT
- DOMAIN,speedtest.macpaw.com,DIRECT
# > Private Tracker
- DOMAIN-SUFFIX,awesome-hd.me,DIRECT
- DOMAIN-SUFFIX,broadcasthe.net,DIRECT
- DOMAIN-SUFFIX,chdbits.co,DIRECT
- DOMAIN-SUFFIX,classix-unlimited.co.uk,DIRECT
- DOMAIN-SUFFIX,empornium.me,DIRECT
- DOMAIN-SUFFIX,gazellegames.net,DIRECT
- DOMAIN-SUFFIX,hdchina.org,DIRECT
- DOMAIN-SUFFIX,hdsky.me,DIRECT
- DOMAIN-SUFFIX,jpopsuki.eu,DIRECT
- DOMAIN-SUFFIX,keepfrds.com,DIRECT
- DOMAIN-SUFFIX,m-team.cc,DIRECT
- DOMAIN-SUFFIX,nanyangpt.com,DIRECT
- DOMAIN-SUFFIX,ncore.cc,DIRECT
- DOMAIN-SUFFIX,open.cd,DIRECT
- DOMAIN-SUFFIX,ourbits.club,DIRECT
- DOMAIN-SUFFIX,passthepopcorn.me,DIRECT
- DOMAIN-SUFFIX,privatehd.to,DIRECT
- DOMAIN-SUFFIX,redacted.ch,DIRECT
- DOMAIN-SUFFIX,springsunday.net,DIRECT
- DOMAIN-SUFFIX,tjupt.org,DIRECT
- DOMAIN-SUFFIX,totheglory.im,DIRECT
# > Other
- DOMAIN-SUFFIX,cn,DIRECT
- DOMAIN-SUFFIX,360in.com,DIRECT
- DOMAIN-SUFFIX,51ym.me,DIRECT
- DOMAIN-SUFFIX,8686c.com,DIRECT
- DOMAIN-SUFFIX,abchina.com,DIRECT
- DOMAIN-SUFFIX,accuweather.com,DIRECT
- DOMAIN-SUFFIX,air-matters.com,DIRECT
- DOMAIN-SUFFIX,air-matters.io,DIRECT
- DOMAIN-SUFFIX,aixifan.com,DIRECT
- DOMAIN-SUFFIX,amd.com,DIRECT
- DOMAIN-SUFFIX,b612.net,DIRECT
- DOMAIN-SUFFIX,bdatu.com,DIRECT
- DOMAIN-SUFFIX,beitaichufang.com,DIRECT
- DOMAIN-SUFFIX,bjango.com,DIRECT
- DOMAIN-SUFFIX,booking.com,DIRECT
- DOMAIN-SUFFIX,bstatic.com,DIRECT
- DOMAIN-SUFFIX,cailianpress.com,DIRECT
- DOMAIN-SUFFIX,chinaso.com,DIRECT
- DOMAIN-SUFFIX,chunyu.mobi,DIRECT
- DOMAIN-SUFFIX,chushou.tv,DIRECT
- DOMAIN-SUFFIX,cmbchina.com,DIRECT
- DOMAIN-SUFFIX,cmbimg.com,DIRECT
- DOMAIN-SUFFIX,ctrip.com,DIRECT
- DOMAIN-SUFFIX,dfcfw.com,DIRECT
- DOMAIN-SUFFIX,docschina.org,DIRECT
- DOMAIN-SUFFIX,douban.com,DIRECT
- DOMAIN-SUFFIX,doubanio.com,DIRECT
- DOMAIN-SUFFIX,douyu.com,DIRECT
- DOMAIN-SUFFIX,dxycdn.com,DIRECT
- DOMAIN-SUFFIX,dytt8.net,DIRECT
- DOMAIN-SUFFIX,eastmoney.com,DIRECT
- DOMAIN-SUFFIX,eudic.net,DIRECT
- DOMAIN-SUFFIX,feng.com,DIRECT
- DOMAIN-SUFFIX,frdic.com,DIRECT
- DOMAIN-SUFFIX,futu5.com,DIRECT
- DOMAIN-SUFFIX,futunn.com,DIRECT
- DOMAIN-SUFFIX,geilicdn.com,DIRECT
- DOMAIN-SUFFIX,gifshow.com,DIRECT
- DOMAIN-SUFFIX,godic.net,DIRECT
- DOMAIN-SUFFIX,hicloud.com,DIRECT
- DOMAIN-SUFFIX,hongxiu.com,DIRECT
- DOMAIN-SUFFIX,hostbuf.com,DIRECT
- DOMAIN-SUFFIX,huxiucdn.com,DIRECT
- DOMAIN-SUFFIX,huya.com,DIRECT
- DOMAIN-SUFFIX,infinitynewtab.com,DIRECT
- DOMAIN-SUFFIX,ithome.com,DIRECT
- DOMAIN-SUFFIX,java.com,DIRECT
- DOMAIN-SUFFIX,keepcdn.com,DIRECT
- DOMAIN-SUFFIX,kkmh.com,DIRECT
- DOMAIN-SUFFIX,licdn.com,DIRECT
- DOMAIN-SUFFIX,linkedin.com,DIRECT
- DOMAIN-SUFFIX,loli.net,DIRECT
- DOMAIN-SUFFIX,luojilab.com,DIRECT
- DOMAIN-SUFFIX,maoyun.tv,DIRECT
- DOMAIN-SUFFIX,meituan.net,DIRECT
- DOMAIN-SUFFIX,mobike.com,DIRECT
- DOMAIN-SUFFIX,mubu.com,DIRECT
- DOMAIN-SUFFIX,myzaker.com,DIRECT
- DOMAIN-SUFFIX,nim-lang-cn.org,DIRECT
- DOMAIN-SUFFIX,nvidia.com,DIRECT
- DOMAIN-SUFFIX,oracle.com,DIRECT
- DOMAIN-SUFFIX,paypal.com,DIRECT
- DOMAIN-SUFFIX,paypalobjects.com,DIRECT
- DOMAIN-SUFFIX,qdaily.com,DIRECT
- DOMAIN-SUFFIX,qidian.com,DIRECT
- DOMAIN-SUFFIX,qyer.com,DIRECT
- DOMAIN-SUFFIX,qyerstatic.com,DIRECT
- DOMAIN-SUFFIX,ronghub.com,DIRECT
- DOMAIN-SUFFIX,ruguoapp.com,DIRECT
- DOMAIN-SUFFIX,sm.ms,DIRECT
- DOMAIN-SUFFIX,smzdm.com,DIRECT
- DOMAIN-SUFFIX,snapdrop.net,DIRECT
- DOMAIN-SUFFIX,snwx.com,DIRECT
- DOMAIN-SUFFIX,s-reader.com,DIRECT
- DOMAIN-SUFFIX,sspai.com,DIRECT
- DOMAIN-SUFFIX,teamviewer.com,DIRECT
- DOMAIN-SUFFIX,tianyancha.com,DIRECT
- DOMAIN-SUFFIX,udacity.com,DIRECT
- DOMAIN-SUFFIX,uning.com,DIRECT
- DOMAIN-SUFFIX,weather.com,DIRECT
- DOMAIN-SUFFIX,weico.cc,DIRECT
- DOMAIN-SUFFIX,weidian.com,DIRECT
- DOMAIN-SUFFIX,ximalaya.com,DIRECT
- DOMAIN-SUFFIX,xinhuanet.com,DIRECT
- DOMAIN-SUFFIX,xmcdn.com,DIRECT
- DOMAIN-SUFFIX,yangkeduo.com,DIRECT
- DOMAIN-SUFFIX,zhangzishi.cc,DIRECT
- DOMAIN-SUFFIX,zhihu.com,DIRECT
- DOMAIN-SUFFIX,zhimg.com,DIRECT
- DOMAIN-SUFFIX,zhuihd.com,DIRECT
- DOMAIN,download.jetbrains.com,DIRECT

# Local Area Network
- DOMAIN-SUFFIX,local,DIRECT
- IP-CIDR,192.168.0.0/16,DIRECT
- IP-CIDR,10.0.0.0/8,DIRECT
- IP-CIDR,172.16.0.0/12,DIRECT
- IP-CIDR,127.0.0.0/8,DIRECT
- IP-CIDR,100.64.0.0/10,DIRECT

# GeoIP China
- GEOIP,CN,DIRECT

- MATCH,选择模式`
