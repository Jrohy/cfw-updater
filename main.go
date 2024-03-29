package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

var (
	version, buildDate, goVersion, gitVersion string
	cfwVersion, tempPath, transWay            string
	updateTrans                               bool
	updateCore                                = true
	ci                                        *cfwInfo
	v                                         = flag.Bool("v", false, "显示版本号")
	forceUpdate                               = flag.Bool("f", false, "强制更新cfw(默认和已存在版本相同则不更新)")
)

const CfwRepos = "Fndroid/clash_for_windows_pkg"

func init() {
	flag.Parse()
	if *forceUpdate {
		fmt.Println("强制更新cfw模式.")
		fmt.Println()
	}
	if *v {
		fmt.Printf("Version: %s\n\n", version)
		fmt.Printf("BuildDate: %s\n\n", buildDate)
		fmt.Printf("GoVersion: %s\n\n", goVersion)
		fmt.Printf("GitVersion: %s\n\n", gitVersion)
		os.Exit(0)
	}
}

func getCfw() *downloadInfo {
	var cfwFileName string
	if ci.installType == MacDmg {
		if runtime.GOARCH == "arm64" {
			cfwFileName = fmt.Sprintf("Clash.for.Windows-%s-arm64.dmg", cfwVersion)
		} else {
			cfwFileName = fmt.Sprintf("Clash.for.Windows-%s.dmg", cfwVersion)
		}
	} else if ci.installType == WinExe {
		cfwFileName = fmt.Sprintf("Clash.for.Windows.Setup.%s.exe", cfwVersion)
	} else {
		cfwFileName = fmt.Sprintf("Clash.for.Windows-%s-win.7z", cfwVersion)
	}
	cfwUrl := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", CfwRepos, cfwVersion, cfwFileName)
	downloadFile(cfwUrl, "")
	di := newDI(cfwUrl)
	if ci.installType == Win7z {
		extractFile(di.fileFullName)
	}
	fmt.Println()
	return di
}

func getTrans(url string) *downloadInfo {
	di := newDI(url)
	downloadFile(url, "")
	if extExists(di.fileFullName, ".zip", ".7z", ".rar") {
		extractFile(di.fileFullName)
	}
	fmt.Println()
	return di
}

func downloadPack() []*downloadInfo {
	var diList []*downloadInfo
	if updateCore {
		diList = append(diList, getCfw())
	}
	transUrl := transDownloadUrl()
	if updateTrans {
		diList = append(diList, getTrans(transUrl))
	}
	return diList
}

func checkEnv() {
	fmt.Println("正在获取本机cfw信息..")
	if ci = checkCfw(); ci == nil {
		exit("请先运行Clash for Windows再来更新!")
	}
	if runtime.GOOS == "darwin" {
		if exePath, err := os.Executable(); err != nil {
			exit(err.Error())
		} else {
			os.Chdir(filepath.Dir(exePath))
		}
	}
	proxyUrl := fmt.Sprintf("127.0.0.1:%s", ci.mixPort)
	os.Setenv("HTTP_PROXY", proxyUrl)
	os.Setenv("HTTPS_PROXY", proxyUrl)
	updateUpdater()
	cfwSelect()
	tranSelect()
}

func task() {
	checkEnv()
	defer timeCost(time.Now())
	tempPath = "temp_" + time.Now().Format("200601021504")
	os.Mkdir(tempPath, os.ModePerm)
	diList := downloadPack()
	updateCfw(diList)
}

func main() {
	task()
	exit("")
}
