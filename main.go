package main

import (
	"flag"
	"fmt"
	"github.com/otiai10/copy"
	"os"
	"os/exec"
	"path"
	"time"
)

var (
	version, buildDate, goVersion, gitVersion                      string
	cfwVersion, currentVersion, tempPath, transWay, specialVersion string
	cfwVersionList                                                 []string
	updateTrans                                                    bool
	updateCore                                                     = true
	v                                                              = flag.Bool("V", false, "显示版本号")
	forceUpdate                                                    = flag.Bool("f", false, "强制更新cfw(默认和已存在版本相同则不更新)")
)

func init() {
	flag.StringVar(&specialVersion, "v", "", "指定cfw安装版本")
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

func updateUpdater() {
	fmt.Println("正在获取updater最新版本号..")
	updaterVersion := "v" + recentlyTag("https://github.com/Jrohy/cfw-updater/tags")[0]
	if version != updaterVersion {
		fmt.Println("发现新版updater: " + updaterVersion)
		downloadFile(fmt.Sprintf("https://github.com/Jrohy/cfw-updater/releases/download/%s/cfw-updater.exe", updaterVersion), "new.exe")
		startBackground()
		os.Rename("cfw-updater.exe", "old.exe")
		os.Rename("new.exe", "cfw-updater.exe")
		fmt.Println()
		exit("updater更新完成, 请手动重新运行!")
	} else {
		c := exec.Command("cmd", "/c", "cls")
		c.Stdout = os.Stdout
		c.Run()
		fmt.Println("cfw-updater " + updaterVersion)
	}
	fmt.Println()
}

func getCfw() *downloadInfo {
	cfwUrl := fmt.Sprintf("https://github.com/Fndroid/clash_for_windows_pkg/releases/download/%s/Clash.for.Windows-%s-win.7z", cfwVersion, cfwVersion)
	downloadFile(cfwUrl, "")
	di := newDI(cfwUrl)
	extract7z(di.fileFullName)
	fmt.Println()
	return di
}

func getTrans(url string) *downloadInfo {
	di := newDI(url)
	downloadFile(url, "")
	extract7z(di.fileFullName)
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

func updateCfw(cfw *cfwInfo, diList []*downloadInfo) {
	if updateCore || updateTrans {
		fmt.Println("更新cfw中...")
		cfw.process.Kill()
	}
	if updateTrans {
		var err error
		if updateCore {
			err = copy.Copy(fullPath(path.Join(diList[1].fileName, "app.asar")), fullPath(path.Join(diList[0].fileName, "resources/app.asar")))
		} else {
			err = copy.Copy(fullPath(path.Join(diList[0].fileName, "app.asar")), path.Join(cfw.rootPath, "resources/app.asar"))
		}
		if err != nil {
			fmt.Println("请尝试以管理员身份运行此程序:")
			exit(err.Error())
		}
	}
	if updateCore {
		if cfw.portableData {
			if err := copy.Copy(cfw.rootPath+"/data", fullPath(diList[0].fileName+"/data")); err != nil {
				exit(err.Error())
			}
		}
		for {
			if err := os.RemoveAll(cfw.rootPath); err == nil {
				break
			}
		}
		if err := copy.Copy(fullPath(diList[0].fileName), cfw.rootPath); err != nil {
			exit(err.Error())
		}
	}
	if updateCore || updateTrans {
		startBackground()
		go exec.Command(path.Join(cfw.rootPath, "Clash for Windows.exe")).Run()
		for {
			if checkCfw() != nil {
				break
			}
		}
		fmt.Printf("更新成功!\n\n")
	}
}

func checkEnv() *cfwInfo {
	var cfwInfo *cfwInfo
	fmt.Println("正在获取本机cfw信息..")
	if cfwInfo = checkCfw(); cfwInfo == nil {
		exit("请先运行Clash for Windows再来更新!")
	}
	currentVersion = cfwInfo.version
	proxyUrl := fmt.Sprintf("127.0.0.1:%s", cfwInfo.mixPort)
	os.Setenv("HTTP_PROXY", proxyUrl)
	os.Setenv("HTTPS_PROXY", proxyUrl)
	fmt.Println("正在获取cfw最新的版本号..")
	cfwVersionList = recentlyTag("https://github.com/Fndroid/clash_for_windows_pkg/tags")
	updateUpdater()
	if !cfwInfo.installVersion {
		cfwSelect()
	}
	tranSelect()
	return cfwInfo
}

func task() {
	cfwInfo := checkEnv()
	defer timeCost(time.Now())
	tempPath = "temp_" + time.Now().Format("200601021504")
	os.Mkdir(tempPath, os.ModePerm)
	diList := downloadPack()
	updateCfw(cfwInfo, diList)
}

func main() {
	task()
	exit("")
}
