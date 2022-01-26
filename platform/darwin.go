//go:build darwin

package platform

import (
	"howett.net/plist"
	"io/ioutil"
	"syscall"
)

// NewSysProcAttr 进程属性
func NewSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setsid: true,
	}
}

// FileVersion 获取文件版本号
func FileVersion(filePath string) (string, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	var result map[string]interface{}
	if _, err = plist.Unmarshal(data, &result); err != nil {
		return "", err
	}
	return result["CFBundleShortVersionString"].(string), nil
}
