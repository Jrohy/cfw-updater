//go:build windows

package platform

import (
	"errors"
	"fmt"
	"github.com/gonutz/w32/v2"
	"syscall"
)

// NewSysProcAttr 进程属性
func NewSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		HideWindow: true,
	}
}

// FileVersion 获取文件版本号
func FileVersion(filePath string) (string, error) {
	size := w32.GetFileVersionInfoSize(filePath)
	if size <= 0 {
		return "", errors.New("获取cfw版本号失败: GetFileVersionInfoSize failed")
	}

	info := make([]byte, size)
	ok := w32.GetFileVersionInfo(filePath, info)
	if !ok {
		return "", errors.New("获取cfw版本号失败: GetFileVersionInfo failed")
	}

	fixed, ok := w32.VerQueryValueRoot(info)
	if !ok {
		return "", errors.New("获取cfw版本号失败: VerQueryValueRoot failed")
	}
	version := fixed.FileVersion()
	return fmt.Sprintf(
		"%d.%d.%d",
		version&0xFFFF000000000000>>48,
		version&0x0000FFFF00000000>>32,
		version&0x00000000FFFF0000>>16,
	), nil
}
