package main

import (
	"fmt"
	"github.com/shirou/gopsutil/v3/process"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
)

type cfwInfo struct {
	rootPath, mixPort, version   string
	portableData, installVersion bool
	process                      *process.Process
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

func checkCfw() *cfwInfo {
	ci := &cfwInfo{}
	processList, _ := process.Processes()
	for _, item := range processList {
		name, _ := item.Name()
		if strings.Contains(name, "Clash for Windows") {
			child, _ := item.Children()
			if len(child) > 1 {
				info, _ := item.Cmdline()
				ci.rootPath = strings.Trim(path.Dir(strings.Replace(info, "\\", "/", -1)), "\"")
				ci.version = getExeVersion(strings.Replace(info, "\"", "", -1))
				ci.process = item
				if IsExists(fmt.Sprintf("%s/Uninstall Clash for Windows.exe", ci.rootPath)) {
					ci.installVersion = true
					updateCore = false
				}
				break
			}
		}
	}
	if ci.rootPath == "" {
		return nil
	}
	cfwConfigPath := ci.rootPath + "/data/config.yaml"
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
	var url string
	if transWay == "" {
		return ""
	} else if specialVersion == "" {
		fmt.Println(fmt.Sprintf("正在获取%s汉化包最新版本号...", transWay))
		searchText := webSearch(fmt.Sprintf("https://github.com/%s/tags", transWay), cfwVersion)
		if searchText == "" {
			fmt.Println(fmt.Sprintf("%s的汉化补丁尚未发布, 若要汉化等后续补丁发布后重新运行工具来更新即可\n", cfwVersion))
			return ""
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
			return ""
		}
		url = fmt.Sprintf("https://github.com/%s/releases/download/%s/app.7z", transWay, dTag)
	}
	updateTrans = true
	return url
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
	fmt.Println("当前cfw版本: " + currentVersion)
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
				specialVersion = choice
			} else if !IsNumeric(choice) {
				fmt.Println("输入有误,请重新输入")
				continue
			} else {
				number, _ := strconv.Atoi(choice)
				if number > len(cfwVersionList) || number < 1 {
					fmt.Println("输入数字越界,请重新输入")
					continue
				}
				specialVersion = cfwVersionList[number-1]
				if number == 1 {
					*forceUpdate = true
				}
			}
		}
		break
	}
}

func cfwSelect() {
	if specialVersion == "" {
		cfwInput()
	}
	if specialVersion != "" {
		if strings.Contains(specialVersion, "v") {
			specialVersion = strings.Replace(specialVersion, "v", "", -1)
		}
		cfwVersion = specialVersion
		fmt.Println("cfw指定安装版本: " + cfwVersion)
		// 通过github tag页面是否为404来判断tag是否存在
		searchText := webSearch("https://github.com/Fndroid/clash_for_windows_pkg/releases/tag/"+specialVersion, "This is not the web page you are looking for")
		if searchText != "" {
			exit(fmt.Sprintf("cfw %s 版本不存在!", specialVersion))
		}
	}
	fmt.Println()
	if !*forceUpdate {
		if strings.Contains(currentVersion, cfwVersion) {
			if specialVersion == "" {
				fmt.Println("当前cfw版本已为最新!")
			} else {
				fmt.Println(fmt.Sprintf("当前cfw已经是%s版本!", specialVersion))
			}
			fmt.Println()
			updateCore = false
		}
	}
}
