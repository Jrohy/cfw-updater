package main

import (
	"cfw-updater/platform"
	"fmt"
	"github.com/shirou/gopsutil/v3/process"
	"os"
	"path"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

type packType int

const (
	Win7z packType = iota
	WinExe
	MacDmg
)

type cfwInfo struct {
	rootPath, mixPort, version string
	portableData               bool
	installType                packType
	process                    *process.Process
}

type downloadInfo struct {
	url, fileFullName, fileName string
}

func newDI(url string) *downloadInfo {
	di := downloadInfo{url: url}
	di.fileFullName = path.Base(url)
	di.fileName = strings.TrimSuffix(di.fileFullName, path.Ext(di.fileFullName))
	return &di
}

func readCfwPort(path string) string {
	var port string
	if fi, err := os.Open(path); err != nil {
		exit(err.Error())
	} else {
		temp := searchText(fi, "mixed-port")
		valid := regexp.MustCompile(`[0-9.]+`)
		port = valid.FindAllStringSubmatch(temp, -1)[0][0]
	}
	return port
}

func parseProcessInfo(ci *cfwInfo, item *process.Process) {
	info, _ := item.Cmdline()
	if runtime.GOOS == "darwin" {
		ci.installType = MacDmg
		ci.rootPath = strings.TrimRight(info, "/Contents/MacOS/Clash for Windows")
		info = fmt.Sprintf("%s/Contents/Info.plist", ci.rootPath)
	} else {
		ci.rootPath = strings.Trim(path.Dir(strings.Replace(info, "\\", "/", -1)), "\"")
		info = strings.Replace(info, "\"", "", -1)
		if !IsExists(info) {
			exit("无法获取cfw信息, 请以管理员身份运行此程序")
		}
	}
	ci.process = item
	if v, err := platform.FileVersion(info); err == nil {
		ci.version = v
	} else {
		exit(err.Error())
	}
	if IsExists(fmt.Sprintf("%s/Uninstall Clash for Windows.exe", ci.rootPath)) {
		if f, err := os.Create(path.Join(ci.rootPath, "test")); err != nil {
			exit(fmt.Sprintf("%s目录无权限写入, 请以管理员身份运行此程序", ci.rootPath))
		} else {
			f.Close()
			os.Remove(path.Join(ci.rootPath, "test"))
		}
		ci.installType = WinExe
	}
}

func checkCfw() *cfwInfo {
	ci := &cfwInfo{}
	processList, _ := process.Processes()
	for _, item := range processList {
		name, _ := item.Name()
		if strings.Contains(name, "Clash for Windows") {
			child, _ := item.Children()
			if len(child) > 1 {
				parseProcessInfo(ci, item)
				break
			}
		}
	}
	if ci.rootPath == "" {
		return nil
	}
	cfwConfigPath := ci.rootPath + "/data/config.yaml"
	if runtime.GOOS == "darwin" {
		cfwConfigPath = ci.rootPath + "/Contents/MacOS/data/config.yaml"
	}
	if IsExists(cfwConfigPath) {
		ci.portableData = true
	} else {
		home, _ := os.UserHomeDir()
		cfwConfigPath = home + "/.config/clash/config.yaml"
		if !IsExists(cfwConfigPath) {
			exit("找不到cfw的配置文件!")
		}
	}
	ci.mixPort = readCfwPort(cfwConfigPath)
	return ci
}

func transDownloadUrl() string {
	if transWay == "" {
		return ""
	}
	tagMap := make(map[string]string)
	tagMap["BoyceLig/Clash_Chinese_Patch"] = cfwVersion
	tagMap["ender-zhao/Clash-for-Windows_Chinese"] = fmt.Sprintf("CFW-V%s_CN", cfwVersion)
	exchange := func() {
		if transWay == "BoyceLig/Clash_Chinese_Patch" {
			transWay = "ender-zhao/Clash-for-Windows_Chinese"
		} else {
			transWay = "BoyceLig/Clash_Chinese_Patch"
		}
	}
	findFileName := func() string {
		return webFirstMatchKey(fmt.Sprintf("https://github.com/%s/releases/tag/%s", transWay, tagMap[transWay]),
			path.Join(tagMap[transWay], "app.7z"), path.Join(tagMap[transWay], "app.asar"), path.Join(tagMap[transWay], "app.rar"))
	}
	tagExists := func() bool {
		return webSearch(fmt.Sprintf("https://github.com/%s/tags", transWay), cfwVersion) == ""
	}

	fmt.Println(fmt.Sprintf("正在获取%s的%s版本汉化包...", transWay, cfwVersion))
	if cfwVersion == cfwVersionList[0] && tagExists() {
		fmt.Println(fmt.Sprintf("%s的%s汉化补丁尚未发布, 正在切换到另一种汉化补丁..", transWay, cfwVersion))
		exchange()
		if tagExists() {
			fmt.Println(fmt.Sprintf("%s的汉化补丁尚未发布, 若要汉化等后续补丁发布后重新运行工具来更新即可\n", cfwVersion))
			return ""
		}
	}
	fileName := findFileName()
	if fileName == "" {
		fmt.Println(fmt.Sprintf("%s的%s汉化补丁不存在, 正在切换到另一种汉化补丁..", transWay, cfwVersion))
		exchange()
		fileName = findFileName()
		if fileName == "" {
			fmt.Println(fmt.Sprintf("%s版本的汉化补丁不存在\n", cfwVersion))
			return ""
		}
	}
	updateTrans = true
	return fmt.Sprintf("https://github.com/%s/releases/download/%s", transWay, fileName)
}

func tranSelect() {
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
}

func cfwInput() {
	fmt.Println("当前cfw版本: " + ci.version)
	fmt.Println()
	for k, v := range cfwVersionList {
		fmt.Printf("%2d. %s\n", k+1, v)
	}
	for {
		choice := ""
		fmt.Print("\n请输入版本序号或者其他版本号(若要更新最新版直接回车即可):")
		_, _ = fmt.Scanln(&choice)
		if choice == "" {
			cfwVersion = cfwVersionList[0]
			fmt.Println("cfw最新版: " + cfwVersion)
		} else {
			if strings.Contains(choice, ".") {
				cfwVersion = choice
			} else if !IsNumeric(choice) {
				fmt.Println("输入有误,请重新输入")
				continue
			} else {
				number, _ := strconv.Atoi(choice)
				if number > len(cfwVersionList) || number < 1 {
					fmt.Println("输入数字越界,请重新输入")
					continue
				}
				cfwVersion = cfwVersionList[number-1]
				if number == 1 {
					*forceUpdate = true
				}
			}
			if strings.Contains(cfwVersion, "v") {
				cfwVersion = strings.Replace(cfwVersion, "v", "", -1)
			}
			fmt.Println("cfw指定安装版本: " + cfwVersion)
		}
		break
	}
}

func cfwSelect() {
	cfwInput()
	// 通过github tag页面是否为404来判断tag是否存在
	searchText := webSearch("https://github.com/Fndroid/clash_for_windows_pkg/releases/tag/"+cfwVersion, "This is not the web page you are looking for")
	if searchText != "" {
		exit(fmt.Sprintf("cfw %s 版本不存在!", cfwVersion))
	}
	fmt.Println()
	if !*forceUpdate {
		if strings.Contains(ci.version, cfwVersion) {
			if cfwVersion == cfwVersionList[0] {
				fmt.Println("当前cfw版本已为最新!")
			} else {
				fmt.Println(fmt.Sprintf("当前cfw已经是%s版本!", cfwVersion))
			}
			fmt.Println()
			updateCore = false
		}
	}
}
