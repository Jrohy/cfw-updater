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
	version, buildDate, goVersion, gitVersion      string
	cfwVersion, tempPath, transWay, specialVersion string
	updateTrans                                    bool
	updateCore                                     = true
	v                                              = flag.Bool("V", false, "显示版本号")
	forceUpdate                                    = flag.Bool("f", false, "强制更新cfw(默认和已存在版本相同则不更新)")
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

func getCfw(cfw *cfwInfo) *downloadInfo {
	if specialVersion == "" {
		fmt.Println("正在获取cfw最新版本号...")
		searchText := webSearch("https://github.com/Fndroid/clash_for_windows_pkg/tags", "archive/refs")
		if searchText == "" {
			exit("获取cfw最新版本号失败!")
		}
		valid := regexp.MustCompile(`[0-9.]+`)
		cfwVersion = strings.TrimSuffix(valid.FindAllStringSubmatch(searchText, -1)[0][0], ".")
		fmt.Println("cfw最新版: " + cfwVersion)
	}
	if !*forceUpdate {
		if strings.Contains(cfw.version, cfwVersion) {
			if specialVersion == "" {
				fmt.Println("当前版本已为最新!")
			} else {
				fmt.Println(fmt.Sprintf("当前已经是%s版本!", specialVersion))
			}
			updateCore = false
			return &downloadInfo{}
		}
	}

	cfwUrl := fmt.Sprintf("https://github.com/Fndroid/clash_for_windows_pkg/releases/download/%s/Clash.for.Windows-%s-win.7z", cfwVersion, cfwVersion)
	downloadFile(cfwUrl)
	di := newDI(cfwUrl)
	extract7z(di.fileFullName)
	fmt.Println()
	return di
}

func getTrans() *downloadInfo {
	var url string
	if specialVersion == "" {
		fmt.Println(fmt.Sprintf("正在获取%s汉化包最新版本号...", transWay))
		searchText := webSearch(fmt.Sprintf("https://github.com/%s/tags", transWay), cfwVersion)
		if searchText == "" {
			fmt.Println(fmt.Sprintf("%s的汉化补丁尚未发布, 若要汉化等后续补丁发布后重新运行工具来更新即可\n", cfwVersion))
			return &downloadInfo{}
		}
		url = fmt.Sprintf("https://github.com/%s/releases/latest/download/app.7z", transWay)
	} else {
		var dTag string
		fmt.Println(fmt.Sprintf("正在获取%s的%s版本汉化包...", transWay, specialVersion))
		if transWay == "BoyceLig/Clash_Chinese_Patch" {
			dTag = specialVersion
		} else if transWay == "ender-zhao/Clash-for-Windows_Chinese" {
			dTag = fmt.Sprintf("CFW-V%s_CN", specialVersion)
		}
		searchText := webSearch(fmt.Sprintf("https://github.com/%s/releases/tag/%s", transWay, dTag), "app.7z")
		if searchText == "" {
			fmt.Println(fmt.Sprintf("%s的app.7z包不存在\n", transWay))
			return &downloadInfo{}
		}
		url = fmt.Sprintf("https://github.com/%s/releases/download/%s/app.7z", transWay, dTag)
	}
	updateTrans = true
	di := newDI(url)
	downloadFile(url)
	extract7z(di.fileFullName)
	fmt.Println()
	return di
}

func downloadPack(cfw *cfwInfo) []*downloadInfo {
	var diList []*downloadInfo
	diList = append(diList, getCfw(cfw))
	if transWay != "" {
		diList = append(diList, getTrans())
	}
	return diList
}

func updateCfw(cfw *cfwInfo, diList []*downloadInfo) {
	if updateCore || updateTrans {
		fmt.Println("更新cfw中...")
		cfw.process.Kill()
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
		fmt.Printf("更新成功!\n\n")
	}
}

func checkEnv() *cfwInfo {
	var cfwInfo *cfwInfo
	if cfwInfo = checkCfw(); cfwInfo == nil {
		exit("请先运行Clash for Windows再来更新!")
	}
	proxyUrl := fmt.Sprintf("127.0.0.1:%s", cfwInfo.mixPort)
	os.Setenv("HTTP_PROXY", proxyUrl)
	os.Setenv("HTTPS_PROXY", proxyUrl)
	if specialVersion != "" {
		if strings.Contains(specialVersion, "v") {
			specialVersion = strings.Replace(specialVersion, "v", "", -1)
		}
		// 通过github tag页面是否为404来判断tag是否存在
		searchText := webSearch("https://github.com/Fndroid/clash_for_windows_pkg/releases/tag/"+specialVersion, "This is not the web page you are looking for")
		if searchText != "" {
			exit(fmt.Sprintf("cfw %s 版本不存在!", specialVersion))
		}
		cfwVersion = specialVersion
		fmt.Println("cfw指定安装版本: " + cfwVersion)
		fmt.Println()
	}
	fmt.Println("请选择要汉化的方式: ")
	fmt.Println("1. ender-zhao/Clash-for-Windows_Chinese")
	fmt.Println("2. BoyceLig/Clash_Chinese_Patch")
	fmt.Println("3. 不进行汉化")
	choice := LoopInput("\n请输入功能序号(若选择第1项直接回车即可):", 3)
	if choice == -1 || choice == 1 {
		transWay = "ender-zhao/Clash-for-Windows_Chinese"
	} else if choice == 2 {
		transWay = "BoyceLig/Clash_Chinese_Patch"
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
