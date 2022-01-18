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
	version, buildDate, goVersion, gitVersion string
	cfwVersion, tempPath, transWay            string
	cfwVersionList                            []string
	updateTrans                               bool
	updateCore                                = true
	ci                                        *cfwInfo
	v                                         = flag.Bool("V", false, "显示版本号")
	forceUpdate                               = flag.Bool("f", false, "强制更新cfw(默认和已存在版本相同则不更新)")
)

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
	var cfwFileName string
	if ci.installVersion {
		cfwFileName = fmt.Sprintf("Clash.for.Windows.Setup.%s.exe", cfwVersion)
	} else {
		cfwFileName = fmt.Sprintf("Clash.for.Windows-%s-win.7z", cfwVersion)
	}
	cfwUrl := fmt.Sprintf("https://github.com/Fndroid/clash_for_windows_pkg/releases/download/%s/%s", cfwVersion, cfwFileName)
	downloadFile(cfwUrl, "")
	di := newDI(cfwUrl)
	if !ci.installVersion {
		extract7z(di.fileFullName)
	}
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

func transUpdate(diList []*downloadInfo, stopCh chan struct{}) {
	if updateTrans {
		var err error
		if ci.installVersion {
			err = copy.Copy(fullPath(path.Join(diList[len(diList)-1].fileName, "app.asar")), path.Join(ci.rootPath, "resources/app.asar"))
		} else {
			err = copy.Copy(fullPath(path.Join(diList[len(diList)-1].fileName, "app.asar")), fullPath(path.Join(diList[0].fileName, "resources/app.asar")))
		}
		if err != nil {
			close(stopCh)
			fmt.Printf("\n\n请尝试以管理员身份运行此程序:\n")
			exit(err.Error())
		}
	}
}

func updateProcess(diList []*downloadInfo, stopCh chan struct{}) {
	closeChan := func() {
		close(stopCh)
		fmt.Printf("\n\n")
	}
	if ci.installVersion {
		if updateCore {
			exec.Command(fullPath(diList[0].fileFullName), "/S").Run()
		}
		var checkInfo *cfwInfo
		for {
			if checkInfo = checkCfw(); checkInfo != nil {
				break
			}
		}
		checkInfo.process.Kill()
		transUpdate(diList, stopCh)
	} else {
		transUpdate(diList, stopCh)
		if updateCore {
			if ci.portableData {
				if err := copy.Copy(ci.rootPath+"/data", fullPath(diList[0].fileName+"/data")); err != nil {
					closeChan()
					exit(err.Error())
				}
			}
			for {
				if err := os.RemoveAll(ci.rootPath); err == nil {
					break
				}
			}
			if err := copy.Copy(fullPath(diList[0].fileName), ci.rootPath); err != nil {
				closeChan()
				exit(err.Error())
			}
		}
	}
}

func updateCfw(diList []*downloadInfo) {
	var stopCh chan struct{}
	if updateCore || updateTrans {
		stopCh = make(chan struct{})
		go showProgress("更新cfw中", stopCh)
		ci.process.Kill()
	}
	updateProcess(diList, stopCh)
	if updateCore || updateTrans {
		startBackground()
		go exec.Command(path.Join(ci.rootPath, "Clash for Windows.exe")).Run()
		for {
			if checkCfw() != nil {
				break
			}
		}
		close(stopCh)
		fmt.Printf("\n\n更新成功!\n\n")
	}
}

func checkEnv() {
	fmt.Println("正在获取本机cfw信息..")
	if ci = checkCfw(); ci == nil {
		exit("请先运行Clash for Windows再来更新!")
	}
	proxyUrl := fmt.Sprintf("127.0.0.1:%s", ci.mixPort)
	os.Setenv("HTTP_PROXY", proxyUrl)
	os.Setenv("HTTPS_PROXY", proxyUrl)
	fmt.Println("正在获取cfw最新的版本号..")
	cfwVersionList = recentlyTag("https://github.com/Fndroid/clash_for_windows_pkg/tags")
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
