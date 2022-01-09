package main

import (
	"github.com/shirou/gopsutil/v3/process"
	"os"
	"path"
	"regexp"
	"strings"
)

type cfwInfo struct {
	rootPath, mixPort, version string
	portable                   bool
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

func checkCfw() *cfwInfo {
	ci := &cfwInfo{}
	processList, _ := process.Processes()
	for _, item := range processList {
		name, _ := item.Name()
		if strings.Contains(name, "Clash for Windows") {
			info, _ := item.Cmdline()
			if !strings.Contains(info, "user-data-dir") {
				ci.rootPath = strings.Trim(path.Dir(strings.Replace(info, "\\", "/", -1)), "\"")
				ci.version = getExeVersion(strings.Replace(info, "\"", "", -1))
				ci.process = item
				break
			}
		}
	}
	if ci.rootPath == "" {
		return nil
	}
	cfwConfigPath := ci.rootPath + "/data/config.yaml"
	if IsExists(cfwConfigPath) {
		ci.portable = true
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
