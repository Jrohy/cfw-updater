package main

import (
	"fmt"
	"github.com/otiai10/copy"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

func updateUpdater() {
	fmt.Println("正在获取updater最新版本号..")
	updaterVersion := "v" + recentlyTag("https://github.com/Jrohy/cfw-updater/tags")[0]
	if version != updaterVersion {
		currentPath, _ := os.Executable()
		_, execName := filepath.Split(currentPath)
		fmt.Println("发现新版updater: " + updaterVersion)
		downloadFileName := "cfw-updater.exe"
		if runtime.GOOS == "darwin" {
			if runtime.GOARCH == "arm64" {
				downloadFileName = "cfw-updater_mac_arm64"
			} else {
				downloadFileName = "cfw-updater_mac_amd64"
			}
		}
		oldName := strings.ReplaceAll(downloadFileName, "cfw-updater", "old")
		newName := strings.ReplaceAll(downloadFileName, "cfw-updater", "new")
		downloadFile(fmt.Sprintf("https://github.com/Jrohy/cfw-updater/releases/download/%s/%s", updaterVersion, downloadFileName), newName)
		if runtime.GOOS == "darwin" {
			exec.Command("chmod", "+x", newName).Run()
		}
		startBackground()
		os.Rename(execName, oldName)
		os.Rename(newName, execName)
		fmt.Println()
		exit("updater更新完成, 请手动重新运行!")
	} else {
		var c *exec.Cmd
		if runtime.GOOS == "darwin" {
			c = exec.Command("zsh", "-c", "clear")
		} else {
			c = exec.Command("cmd", "/c", "cls")
		}
		c.Stdout = os.Stdout
		c.Run()
		fmt.Println("cfw-updater " + updaterVersion)
	}
	fmt.Println()
}

func updateDmgShell(updatePath, dmgPath string) string {
	return fmt.Sprintf(`
DMG_PATH="%s"
UPDATE_PATH="%s"
TEMP_PATH="%s"
VOLUME=$(hdiutil attach "$DMG_PATH" | grep Volumes | awk -F " " '{for (i=3;i<=NF;i++)printf("%%s ", $i);print ""}'|awk '$1=$1')
[[ -e "$UPDATE_PATH"/Contents/MacOS/data ]] && sudo cp -rp "$UPDATE_PATH"/Contents/MacOS/data "$TEMP_PATH"/
sudo rm -rf "$UPDATE_PATH"
sudo cp -rf "$VOLUME"/*.app "$UPDATE_PATH"
[[ -e "$TEMP_PATH"/data ]] && sudo mv "$TEMP_PATH"/data "$UPDATE_PATH"/Contents/MacOS/
hdiutil detach "$VOLUME" >/dev/null
sudo xattr -r -d com.apple.quarantine "$UPDATE_PATH"
`, dmgPath, updatePath, tempPath)
}

func updateTransFile(diList []*downloadInfo, stopCh chan struct{}) {
	var srcFile string
	if strings.Contains(diList[len(diList)-1].fileFullName, "asar") {
		srcFile = fullPath("app.asar")
	} else {
		srcFile = fullPath(path.Join(diList[len(diList)-1].fileName, "app.asar"))
	}
	if runtime.GOOS == "darwin" {
		execCommand(fmt.Sprintf("sudo cp -rp \"%s\" \"%s\"", srcFile, path.Join(ci.rootPath, "Contents/Resources/app.asar")))
	} else {
		if err := copy.Copy(srcFile, path.Join(ci.rootPath, "resources/app.asar")); err != nil {
			close(stopCh)
			fmt.Printf("\n\n请尝试以管理员身份运行此程序:\n")
			exit(err.Error())
		}
	}
}

func updateCoreFile(diList []*downloadInfo, stopCh chan struct{}) {
	if ci.portableData && runtime.GOOS == "windows" {
		if err := copy.Copy(ci.rootPath+"/data", fullPath(diList[0].fileName+"/data")); err != nil {
			close(stopCh)
			fmt.Printf("\n\n")
			exit(err.Error())
		}
	}
	if ci.installType == MacDmg {
		startBackground()
		execCommand(updateDmgShell(ci.rootPath, fullPath(diList[0].fileFullName)))
	} else if ci.installType == WinExe {
		startBackground()
		go exec.Command(fullPath(diList[0].fileFullName), "/S").Run()
		for {
			if check := checkCfw(); check != nil {
				check.process.Kill()
				break
			}
		}
	} else {
		for {
			if err := copy.Copy(fullPath(diList[0].fileName), ci.rootPath); err == nil {
				break
			}
		}
	}
}

func stopCfw(stopCh chan struct{}) {
	if runtime.GOOS == "darwin" {
		// 提前获取macOS用于更新的密码
		fmt.Println("请输入更新所需的密码(有权限无需密码的会跳过):")
		execCommand("sudo echo >/dev/null")
		fmt.Println()
	}
	go showProgress("更新cfw中", stopCh)
	ci.process.Kill()
	if ci.portableData && len(ci.otherProcess) > 1 {
		for _, item := range ci.otherProcess {
			item.Kill()
		}
	}
}

func startCfw(stopCh chan struct{}) {
	if ci.installType == Win7z {
		startBackground()
	}
	if runtime.GOOS == "darwin" {
		go exec.Command(path.Join(ci.rootPath, "Contents/MacOS/Clash for Windows")).Run()
	} else {
		go exec.Command(path.Join(ci.rootPath, "Clash for Windows.exe")).Run()
	}
	for {
		if checkCfw() != nil {
			break
		}
	}
	close(stopCh)
}

func updateCfw(diList []*downloadInfo) {
	stopCh := make(chan struct{})
	if updateCore || updateTrans {
		stopCfw(stopCh)
	}
	if updateCore {
		updateCoreFile(diList, stopCh)
	}
	if updateTrans {
		updateTransFile(diList, stopCh)
	}
	if updateCore || updateTrans {
		startCfw(stopCh)
		fmt.Printf("\n\n更新成功!\n\n")
	}
}
