//go:build darwin

package platform

import "syscall"

// NewSysProcAttr 进程属性
func NewSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setsid: true,
	}
}

// FileVersion 获取文件版本号
func FileVersion(filePath string) (string, error) {
	return "", nil
}
