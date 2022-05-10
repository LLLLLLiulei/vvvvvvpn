package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

//go:embed lib
var embededFiles embed.FS

//go:embed static/*
var staticFiles embed.FS

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

	resp, _ := httpClient.Get("http://xxxx")
	bytes, _ := ioutil.ReadAll(resp.Body)
	defer func() {
		resp.Body.Close()
	}()

	var vpnInfo VPNInfo
	json.Unmarshal(bytes, &vpnInfo)

	fmt.Println(vpnInfo)
	return vpnInfo
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

func createGin() {
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

	// r.Run()
	r.Run(":8866")
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
		cmd := exec.Command("chmod", "+x", helperpath)
		cmd.Start()
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

func main() {
	// Do()
	// Test()
	createGin()

}
