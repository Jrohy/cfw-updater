package main

import (
	"flag"
	"fmt"
	"github.com/otiai10/copy"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"time"
)

var (
	cfwVersion, tempPath, transWay string
	updateTrans                    bool
	updateCore                     = true
	forceUpdate                    = flag.Bool("f", false, "是否强制更新cfw(默认策略和最新版本相同则不更新)")
)

func init() {
	flag.Parse()
	if *forceUpdate {
		fmt.Println("强制更新cfw模式.")
		fmt.Println()
	}
}

func getCfw(cfw *cfwInfo) *downloadInfo {
	fmt.Println("正在获取cfw最新版本号...")
	searchText := webSearch("https://github.com/Fndroid/clash_for_windows_pkg/tags", "archive/refs")
	valid := regexp.MustCompile(`[0-9.]+`)
	cfwVersion = strings.TrimSuffix(valid.FindAllStringSubmatch(searchText, -1)[0][0], ".")
	fmt.Println("cfw最新版: " + cfwVersion)
	if !*forceUpdate {
		if strings.Contains(cfw.version, cfwVersion) {
			fmt.Println("当前版本已为最新!")
			updateCore = false
			return &downloadInfo{}
		}
	}

	cfwUrl := fmt.Sprintf("https://github.com/Fndroid/clash_for_windows_pkg/releases/download/%s/Clash.for.Windows-%s-win.7z", cfwVersion, cfwVersion)
	downloadFile(cfwUrl)
	di := newDI(cfwUrl)
	extract7z(di.fileFullName)
	return di
}

func getTrans() *downloadInfo {
	fmt.Println(fmt.Sprintf("正在获取%s汉化包最新版本号...", transWay))
	searchText := webSearch(fmt.Sprintf("https://github.com/%s/tags", transWay), cfwVersion)
	if searchText == "" {
		fmt.Println(fmt.Sprintf("%s的汉化补丁尚未发布, 若要汉化等后续补丁发布后重新运行工具来更新即可\n", cfwVersion))
		return &downloadInfo{}
	} else {
		updateTrans = true
	}
	url := fmt.Sprintf("https://github.com/%s/releases/latest/download/app.7z", transWay)
	di := newDI(url)
	downloadFile(url)
	extract7z(di.fileFullName)
	return di
}

func downloadPack(cfw *cfwInfo) []*downloadInfo {
	var diList []*downloadInfo
	proxyUrl := fmt.Sprintf("127.0.0.1:%s", cfw.mixPort)
	os.Setenv("HTTP_PROXY", proxyUrl)
	os.Setenv("HTTPS_PROXY", proxyUrl)
	diList = append(diList, getCfw(cfw))
	fmt.Println()
	if transWay != "" {
		diList = append(diList, getTrans())
	}
	return diList
}

func updateCfw(cfw *cfwInfo, diList []*downloadInfo) {
	if updateCore || updateTrans {
		cfw.process.Kill()
		// 等待cfw完全退出
		for {
			if c := checkCfw(); c == nil {
				break
			}
		}
	}
	if updateTrans {
		if updateCore {
			copy.Copy(fullPath(path.Join(diList[1].fileName, "app.asar")), fullPath(path.Join(diList[0].fileName, "resources/app.asar")))
		} else {
			copy.Copy(fullPath(path.Join(diList[1].fileName, "app.asar")), path.Join(cfw.rootPath, "resources/app.asar"))
		}
	}
	if updateCore {
		if cfw.portable {
			if err := copy.Copy(cfw.rootPath+"/data", fullPath(diList[0].fileName+"/data")); err != nil {
				exit(err.Error())
			}
		}
		if err := os.RemoveAll(cfw.rootPath); err != nil {
			exit(err.Error())
		} else {
			if err := copy.Copy(fullPath(diList[0].fileName), cfw.rootPath); err != nil {
				exit(err.Error())
			}
		}
	}
	if updateCore || updateTrans {
		println("更新完成, 请手动启动服务!")
	}
}

func checkEnv() *cfwInfo {
	var cfwInfo *cfwInfo
	if _, err := exec.LookPath("7z"); err != nil {
		exit("7z命令不存在, 请先安装7z(https://www.7-zip.org)并配置环境变量后再来运行!")
	}
	if cfwInfo = checkCfw(); cfwInfo == nil {
		exit("请先运行Clash for Windows再来更新!")
	}
	fmt.Println("请选择要汉化的方式: ")
	fmt.Println("1. BoyceLig/Clash_Chinese_Patch")
	fmt.Println("2. ender-zhao/Clash-for-Windows_Chinese")
	fmt.Println("3. 不进行汉化")
	choice := LoopInput("\n请输入功能序号(若选择第1项直接回车即可):", 3)
	if choice == -1 || choice == 1 {
		transWay = "BoyceLig/Clash_Chinese_Patch"
	} else if choice == 2 {
		transWay = "ender-zhao/Clash-for-Windows_Chinese"
	}
	fmt.Println()
	return cfwInfo
}

func task() {
	cfwInfo := checkEnv()
	defer timeCost(time.Now())
	tempPath = "temp_" + time.Now().Format("200601021504")
	os.Mkdir(tempPath, os.ModePerm)
	diList := downloadPack(cfwInfo)
	updateCfw(cfwInfo, diList)
}

func main() {
	task()
	exit("")
}
