package main

import (
	"bufio"
	"cfw-updater/platform"
	"fmt"
	"github.com/cheggaaa/pb/v3"
	"github.com/eiannone/keyboard"
	"github.com/gen2brain/go-unarr"
	"github.com/mholt/archiver/v3"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"
	"runtime"
	"strconv"
	"strings"
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
	if runtime.GOOS == "windows" {
		fmt.Printf("\nPress any key to exit...")
		b := make([]byte, 1)
		os.Stdin.Read(b)
	}
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

func downloadFile(url, downLoadPath string) {
	var (
		count int
		err   error
		resp  *http.Response
		bar   *pb.ProgressBar
		out   *os.File
	)
	closeDownload := func() {
		if bar != nil {
			bar.Finish()
		}
		if resp != nil {
			resp.Body.Close()
		}
		if out != nil {
			out.Close()
		}
	}
	fmt.Println(fmt.Sprintf("正在下载: %s", url))
	for {
		if resp, err = http.Get(url); err != nil {
			if count > 3 {
				closeDownload()
				exit(err.Error())
			} else {
				count++
				fmt.Println("正在重试中(http get)..")
				continue
			}
		}
		if resp.StatusCode == 404 {
			exit(fmt.Sprintf("%s: 404 Not Found", url))
		}
		if downLoadPath == "" {
			downLoadPath = fullPath(path.Base(url))
		}
		out, err = os.Create(downLoadPath)
		if err != nil {
			closeDownload()
			exit(err.Error())
		}

		bar = pb.StartNew(int(resp.ContentLength))
		bar.Set(pb.Bytes, true)
		bar.SetTemplateString(`{{counters .}} {{bar .}} {{percent .}} {{speed . "%s/s"}} {{rtime .}}`)
		counter := &WriteCounter{bar}
		if _, err = io.Copy(out, io.TeeReader(resp.Body, counter)); err != nil {
			closeDownload()
			if count > 3 {
				exit(err.Error())
			} else {
				count++
				fmt.Println("正在重试中(io copy)..")
				continue
			}
		}
		closeDownload()
		break
	}
}

func searchText(r io.Reader, key string) string {
	scanner := bufio.NewScanner(r)
	var findStr strings.Builder
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), key) {
			findStr.WriteString(scanner.Text())
		}
	}
	return findStr.String()
}

func httpGet(url string) *http.Response {
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
	return resp
}

func extExists(key string, exts ...string) bool {
	for _, v := range exts {
		if path.Ext(key) == v {
			return true
		}
	}
	return false
}

func webSearch(url, key string) string {
	resp := httpGet(url)
	defer resp.Body.Close()
	return searchText(resp.Body, key)
}

func webFindUrl(url string, keys ...string) string {
	var (
		err  error
		resp *http.Response
	)
	for _, v := range keys {
		downloadUrl := fmt.Sprintf("%s/%s", url, v)
		resp, err = http.Get(downloadUrl)
		if err == nil && resp.StatusCode != 404 {
			return downloadUrl
		}
	}
	return ""
}

func recentlyTag(url string) []string {
	var tagStr string
	searchText := webSearch(url, "archive/refs")
	if searchText == "" {
		exit(fmt.Sprintf("获取%s最新版本号失败!", url))
	}
	result := regexp.MustCompile(`[\d.]{2,}\d`).FindAllStringSubmatch(searchText, -1)
	if result == nil {
		exit(fmt.Sprintf("获取%s最新版本号失败!", url))
	}
	for _, v := range result {
		if !strings.Contains(tagStr, v[0]) {
			tagStr = tagStr + " " + v[0]
		}
	}
	return strings.Split(strings.TrimPrefix(tagStr, " "), " ")
}

func extractFile(name string) {
	stopCh := make(chan struct{})
	closeChan := func() {
		close(stopCh)
		fmt.Println("")
	}
	go showProgress(fmt.Sprintf("解压%s中", name), stopCh)
	extractPath := fullPath(strings.TrimSuffix(name, path.Ext(name)))
	if !IsExists(extractPath) {
		if extExists(name, ".7z", ".zip") {
			a, err := unarr.NewArchive(fullPath(name))
			if err != nil {
				closeChan()
				exit(err.Error())
			}
			defer a.Close()
			_, err = a.Extract(extractPath)
			if err != nil {
				closeChan()
				exit(err.Error())
			}
		} else if path.Ext(name) == ".rar" {
			if err := archiver.Unarchive(fullPath(name), extractPath); err != nil {
				closeChan()
				exit(err.Error())
			}
		}
	}
	closeChan()
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
		Path:        os.Args[0],
		Args:        os.Args,
		Env:         append(os.Environ(), fmt.Sprintf("%s=%d", "CFW_DAEMON_IDX", runIdx)),
		SysProcAttr: platform.NewSysProcAttr(),
	}
	if err := cmd.Start(); err != nil {
		exit(err.Error())
	}
}

func showProgress(tip string, stopCh chan struct{}) {
	count := 1
	for {
		select {
		case <-stopCh:
			return
		default:
			if count > 3 {
				count = 1
			}
			fmt.Printf("\r%s%-3s", tip, strings.Repeat(".", count))
			time.Sleep(time.Millisecond * 500)
			count++
		}
	}
}

// execCommand 运行命令并实时查看运行结果
func execCommand(command string) error {
	cmd := exec.Command("zsh", "-c", command)

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		fmt.Println("Error:The command is err: ", err.Error())
		return err
	}
	ch := make(chan string, 100)
	stdoutScan := bufio.NewScanner(stdout)
	stderrScan := bufio.NewScanner(stderr)
	go func() {
		for stdoutScan.Scan() {
			line := stdoutScan.Text()
			ch <- line
		}
	}()
	go func() {
		for stderrScan.Scan() {
			line := stderrScan.Text()
			ch <- line
		}
	}()
	var err error
	go func() {
		err = cmd.Wait()
		if err != nil && !strings.Contains(err.Error(), "exit status") {
			fmt.Println("wait:", err.Error())
		}
		close(ch)
	}()
	for line := range ch {
		fmt.Println(line)
	}
	return err
}
