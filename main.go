package main

import (
	"bytes"
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

//go:embed lib
var embededFiles embed.FS

//go:embed static/*
var staticFiles embed.FS

var api = ""
var vpnname = "vvvvvvpn"
var httpClient = http.DefaultClient

type VPNInfo struct {
	Account string `json:"account"`
	Server  string `json:"server"`
	Pwd     string `json:"pwd"`
	Secret  string `json:"secret"`
}

func fetchVpnInfo() VPNInfo {
	fmt.Println("get vpn account")

	mac := Base64Encoding(NewRandomMac())
	fmt.Println("mac:" + mac)
	url := api + mac

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "eVPN/1.0 (Mac OS X Version 10.16 (Build 21F79))")

	resp, _ := httpClient.Do(req)
	bytes, _ := ioutil.ReadAll(resp.Body)

	defer func() {
		resp.Body.Close()
	}()

	var vpnInfo VPNInfo
	json.Unmarshal(bytes, &vpnInfo)

	fmt.Println(&vpnInfo)
	return vpnInfo
}

func NewRandomMac() string {
	var m [6]byte

	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 6; i++ {
		mac_byte := rand.Intn(256)
		m[i] = byte(mac_byte)
		rand.Seed(int64(mac_byte))
	}

	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", m[0], m[1], m[2], m[3], m[4], m[5])
}

func getHelperPath() string {
	tempdir := os.TempDir()
	filepath := tempdir + "/vvvvvvpnhelper"
	file, _ := os.Create(filepath)
	defer func() {
		file.Close()
	}()

	helper, _ := embededFiles.Open("lib/helper")
	io.Copy(file, helper)
	return filepath
}

func setVPNStatus(vpnname, status string) {
	exec.Command("scutil", "--nc", status, vpnname).Run()
}

func getVPNStatus(vpnname string) bool {
	cmd := exec.Command("scutil", "--nc", "status", vpnname)

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr

	cmd.Run()

	return strings.Contains(out.String(), "Connected")
}

func createVPN(helper, vpnname string, vpn *VPNInfo) {
	exec.Command(helper, "create", "--cisco", vpnname, "--endpoint", vpn.Server, "--username", vpn.Account, "--password", vpn.Pwd, "--sharedsecret", vpn.Secret).Run()
}

func deleteVPN(helper, vpnname string) {
	exec.Command(helper, "delete", "--name", vpnname).Run()
}

func sleep(second int) {
	time.Sleep(time.Duration(second) * time.Second)
}

func createGin(port int) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	tmpl := template.Must(template.New("").ParseFS(staticFiles, "static/*"))
	r.SetHTMLTemplate(tmpl)

	r.GET("/", func(c *gin.Context) {
		fmt.Println(tmpl.DefinedTemplates())
		c.HTML(http.StatusOK, "index.html", nil)
	})

	r.GET("status", func(c *gin.Context) {
		connected := getVPNStatus(vpnname)
		if connected {
			c.String(http.StatusOK, "1")
		} else {
			c.String(http.StatusOK, "0")
		}
	})

	r.POST("/status/connect", func(c *gin.Context) {
		Do()
		c.String(http.StatusOK, "ok")
	})
	r.POST("/status/disconnect", func(c *gin.Context) {
		fmt.Println("stop vpn :" + vpnname)
		setVPNStatus(vpnname, "stop")
		c.String(http.StatusOK, "ok")
	})

	r.Any("/static/*filepath", func(c *gin.Context) {
		staticServer := http.FileServer(http.FS(staticFiles))
		staticServer.ServeHTTP(c.Writer, c.Request)
	})

	r.Run(":" + strconv.Itoa(port))
}

func Do() {
	var wg sync.WaitGroup
	wg.Add(2)

	var vpn VPNInfo
	go func() {
		defer wg.Done()
		vpn = fetchVpnInfo()
	}()

	var helperpath string
	go func() {
		defer wg.Done()
		helperpath = getHelperPath()
		exec.Command("chmod", "+x", helperpath).Run()
	}()

	wg.Wait()

	// vpn_name := "vvvvvvpn-" + strconv.Itoa(int(time.Now().Unix()))

	fmt.Println("stop vpn :" + vpnname)
	setVPNStatus(vpnname, "stop")
	sleep(2)

	fmt.Println("delete vpn :" + vpnname)
	deleteVPN(helperpath, vpnname)
	sleep(2)

	fmt.Println("create vpn :" + vpnname)
	createVPN(helperpath, vpnname, &vpn)
	sleep(2)

	fmt.Println("start vpn :" + vpnname)
	setVPNStatus(vpnname, "start")

	fmt.Println("Success!")
}

func Base64Encoding(str string) string { //Base64编码
	src := []byte(str)
	res := base64.StdEncoding.EncodeToString(src) //将编码变成字符串
	return res
}

func Base64Decoding(str string) string { //Base64解码
	res, _ := base64.StdEncoding.DecodeString(str)
	return string(res)
}

func main() {
	// Do()
	// Test()

	go exec.Command("open", "http://localhost:8866").Run()
	createGin(8866)

}
