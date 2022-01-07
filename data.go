package main

import (
	"github.com/shirou/gopsutil/v3/process"
	"path"
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
