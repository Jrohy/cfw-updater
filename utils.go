package main

import (
	"bufio"
	"fmt"
	"github.com/cheggaaa/pb/v3"
	"github.com/eiannone/keyboard"
	"github.com/gen2brain/go-unarr"
	"github.com/gonutz/w32/v2"
	"github.com/shirou/gopsutil/v3/process"
	"io"
	"net/http"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
)

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
	fmt.Println(fmt.Sprintf("正在下载: %s", url))
	resp, _ := http.Get(url)
	defer resp.Body.Close()
	out, err := os.Create(fullPath(path.Base(url)))
	if err != nil {
		exit(err.Error())
	}
	defer out.Close()
	bar := pb.StartNew(int(resp.ContentLength))
	bar.Set(pb.Bytes, true)
	counter := &WriteCounter{bar}
	_, err = io.Copy(out, io.TeeReader(resp.Body, counter))
	if err != nil {
		exit(err.Error())
	}
	bar.Finish()
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
	resp, err := http.Get(url)
	if err != nil {
		exit(err.Error())
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
