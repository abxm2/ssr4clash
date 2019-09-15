package main

import (
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

var port string

func main() {
	gin.SetMode("release")

	port = os.Getenv("PORT")

	if port == "" {
		log.Fatal("$PORT must be set")
	}

	r := gin.Default()
	r.Use(cors.Default())
	r.GET("/", rhelp)
	r.GET("/ping", rping)
	r.GET("/ssrsub2clash", ssr2clash)
	r.Run(":" + port)
}

func rhelp(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, `<html><body>`+
	"example:" +
		"<p>/ping?host=baidu.com&port=80</p>" +
		"<p>/ssrsub2clash?sub=sublink</p>"+
	`</body></html>` )
}

func rping(c *gin.Context) {
	ahost := c.DefaultQuery("host", "127.0.0.1")
	aport, err := strconv.Atoi(c.DefaultQuery("port", port))
	if err != nil {
		aport = 80
	}
	aips, _ := net.LookupIP(ahost)
	tcpt, tsuccess := tcpPing(ahost, aport)
	c.JSON(http.StatusOK, gin.H{
		"host": ahost,
		"port": aport,
		"success": tsuccess,
		"delay":   tcpt,
		"ip": aips,
	})
}


func tcpPing(tcpingaddr string, tcpport int) (float32, bool) {
	startTime := time.Now()
	conn, err := net.DialTimeout("tcp", tcpingaddr+":"+strconv.Itoa(tcpport), time.Second*3)
	endTime := time.Now()
	if err != nil {
		return -1, false
	} else {
		defer conn.Close()
		return float32(float32(endTime.Sub(startTime)) / float32(time.Millisecond)), true
	}
}

func ssr2clash(c *gin.Context) {
	sublink := c.DefaultQuery("sub", "no")
	if sublink=="no" {
		c.String(http.StatusNotFound, "请添加参数 ?sub=sublink  sublink是你自己的ssr订阅链接")
		return
	}
	if sublink=="sublink" {
		c.String(http.StatusNotFound, "让你把sublink改成自己的ssr订阅链接，你不改，直接点开，你是不是傻，我就问问你，你是不是傻")
		return
	}
	txt,groupname,err := ssrsub2clashconf(sublink)
	if err!=nil {
		c.String(http.StatusNotFound, "订阅转化出错")
		return
	}
	filename := groupname+".yml"
	//filenamegbk,_ := UTF82GB2312([]byte(filename))
	//filename = string(filenamegbk)
	filename = url.QueryEscape(filename)
	c.Header("content-disposition", `attachment; filename=`+filename +`; filename*=UTF-8''` +filename )
	c.Header("Accept-Length", fmt.Sprintf("%d", len(txt)))
	c.Data(http.StatusOK, "application/text/plain", []byte(txt))
}


