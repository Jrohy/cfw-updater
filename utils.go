package main

import (
	"bufio"
	"fmt"
	"github.com/cheggaaa/pb/v3"
	"github.com/eiannone/keyboard"
	"github.com/gen2brain/go-unarr"
	"github.com/gonutz/w32/v2"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// 运行时调用startBackground的次数
var runIdx = 0

func timeCost(start time.Time) {
	fmt.Printf("time cost: %v\n", time.Since(start).String())
}

func fullPath(basePath string) string {
	return path.Join(tempPath, basePath)
}

func exit(info string) {
	if IsExists(tempPath) {
		os.RemoveAll(tempPath)
	}
	if info != "" {
		fmt.Println(info)
	}
	fmt.Printf("\nPress any key to exit...")
	b := make([]byte, 1)
	os.Stdin.Read(b)
	os.Exit(0)
}

// IsExists 检测指定路径文件或者文件夹是否存在
func IsExists(path string) bool {
	_, err := os.Stat(path) //os.Stat获取文件信息
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

type WriteCounter struct {
	bar *pb.ProgressBar
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.bar.Add(n)
	return n, nil
}

func downloadFile(url string) {
	var (
		count int
		err   error
		resp  *http.Response
		bar   *pb.ProgressBar
	)
	fmt.Println(fmt.Sprintf("正在下载: %s", url))
	for {
		if resp, err = http.Get(url); err != nil {
			if count > 3 {
				if bar != nil {
					bar.Finish()
					fmt.Println()
				}
				exit(err.Error())
			} else {
				count++
				fmt.Println("正在重试中(http get)..")
				continue
			}
		}
		out, err := os.Create(fullPath(path.Base(url)))
		if err != nil {
			exit(err.Error())
		}

		bar = pb.StartNew(int(resp.ContentLength))
		bar.Set(pb.Bytes, true)
		bar.SetTemplateString(`{{counters .}} {{bar .}} {{percent .}} {{speed . "%s/s"}} {{rtime .}}`)
		counter := &WriteCounter{bar}
		if _, err = io.Copy(out, io.TeeReader(resp.Body, counter)); err != nil {
			if count > 3 {
				exit(err.Error())
			} else {
				count++
				fmt.Println("正在重试中(io copy)..")
				continue
			}
		}
		resp.Body.Close()
		out.Close()
		bar.Finish()
		break
	}
}

func searchText(r io.Reader, key string) string {
	scanner := bufio.NewScanner(r)
	var findStr string
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), key) {
			findStr = scanner.Text()
			break
		}
	}
	return findStr
}

func webSearch(url, key string) string {
	var (
		count int
		err   error
		resp  *http.Response
	)
	for {
		resp, err = http.Get(url)
		if err == nil {
			break
		}
		if count > 3 {
			exit(err.Error())
		} else {
			fmt.Println("正在重试中..")
			count++
		}
	}
	defer resp.Body.Close()
	return searchText(resp.Body, key)
}

func extract7z(name string) {
	fmt.Println(fmt.Sprintf("解压%s中..", name))
	extractPath := fullPath(strings.TrimSuffix(name, path.Ext(name)))
	if !IsExists(extractPath) {
		a, err := unarr.NewArchive(fullPath(name))
		if err != nil {
			exit(err.Error())
		}
		defer a.Close()
		_, err = a.Extract(extractPath)
		if err != nil {
			exit(err.Error())
		}
	}
}

func getExeVersion(exePath string) string {
	size := w32.GetFileVersionInfoSize(exePath)
	if size <= 0 {
		exit("获取cfw版本号失败: GetFileVersionInfoSize failed")
	}

	info := make([]byte, size)
	ok := w32.GetFileVersionInfo(exePath, info)
	if !ok {
		exit("获取cfw版本号失败: GetFileVersionInfo failed")
	}

	fixed, ok := w32.VerQueryValueRoot(info)
	if !ok {
		exit("获取cfw版本号失败: VerQueryValueRoot failed")
	}
	version := fixed.FileVersion()
	return fmt.Sprintf(
		"%d.%d.%d.%d",
		version&0xFFFF000000000000>>48,
		version&0x0000FFFF00000000>>32,
		version&0x00000000FFFF0000>>16,
		version&0x000000000000FFFF>>0,
	)
}

func getChar(str string) string {
	err := keyboard.Open()
	if err != nil {
		panic(err)
	}
	defer keyboard.Close()
	fmt.Print(str)
	char, _, err := keyboard.GetKey()
	fmt.Printf("%c\n", char)
	if char == 0 {
		return ""
	} else {
		return string(char)
	}
}

// IsNumeric is_numeric()
// Numeric strings consist of optional sign, any number of digits, optional decimal part and optional exponential part.
// Thus +0123.45e6 is a valid numeric value.
func IsNumeric(val interface{}) bool {
	switch val.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return true
	case float32, float64, complex64, complex128:
		return true
	case string:
		str := val.(string)
		if str == "" {
			return false
		}
		// Trim any whitespace
		str = strings.TrimSpace(str)
		if str[0] == '-' || str[0] == '+' {
			if len(str) == 1 {
				return false
			}
			str = str[1:]
		}
		// hex
		if len(str) > 2 && str[0] == '0' && (str[1] == 'x' || str[1] == 'X') {
			for _, h := range str[2:] {
				if !((h >= '0' && h <= '9') || (h >= 'a' && h <= 'f') || (h >= 'A' && h <= 'F')) {
					return false
				}
			}
			return true
		}
		// 0-9, Point, Scientific
		p, s, l := 0, 0, len(str)
		for i, v := range str {
			if v == '.' { // Point
				if p > 0 || s > 0 || i+1 == l {
					return false
				}
				p = i
			} else if v == 'e' || v == 'E' { // Scientific
				if i == 0 || s > 0 || i+1 == l {
					return false
				}
				s = i
			} else if v < '0' || v > '9' {
				return false
			}
		}
		return true
	}

	return false
}

func LoopInput(tip string, len int) int {
	for {
		inputString := ""
		if len < 10 {
			inputString = getChar(tip)
		} else {
			fmt.Print(tip)
			_, _ = fmt.Scanln(&inputString)
		}
		if inputString == "" {
			return -1
		} else if !IsNumeric(inputString) {
			fmt.Println("输入有误,请重新输入")
			continue
		}
		number, _ := strconv.Atoi(inputString)
		if number <= len && number > 0 {
			return number
		} else {
			fmt.Println("输入数字越界,请重新输入")
		}
	}
}

// startBackground 把本身程序转化为后台运行
func startBackground() {
	//判断子进程还是父进程
	runIdx++
	envIdx, err := strconv.Atoi(os.Getenv("CFW_DAEMON_IDX"))
	if err != nil {
		envIdx = 0
	}
	if runIdx <= envIdx { //子进程, 退出
		return
	}
	//启动子进程
	cmd := &exec.Cmd{
		Path: os.Args[0],
		Args: os.Args,
		Env:  append(os.Environ(), fmt.Sprintf("%s=%d", "CFW_DAEMON_IDX", runIdx)),
		SysProcAttr: &syscall.SysProcAttr{
			HideWindow: true,
		},
	}
	if err := cmd.Start(); err != nil {
		exit(err.Error())
	}
}
