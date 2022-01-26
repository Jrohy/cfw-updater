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
				downloadFileName = "cfw-updater_darwin_arm64"
			} else {
				downloadFileName = "cfw-updater_darwin_amd64"
			}
		}
		oldName := strings.ReplaceAll(downloadFileName, "cfw-updater", "old")
		newName := strings.ReplaceAll(downloadFileName, "cfw-updater", "new")
		downloadFile(fmt.Sprintf("https://github.com/Jrohy/cfw-updater/releases/download/%s/%s", updaterVersion, downloadFileName), newName)
		startBackground()
		os.Rename(execName, oldName)
		os.Rename(newName, execName)
		fmt.Println()
		exit("updater更新完成, 请手动重新运行!")
	} else {
		if runtime.GOOS == "darwin" {
			execCommand("clear")
		} else {
			c := exec.Command("cmd", "/c", "cls")
			c.Stdout = os.Stdout
			c.Run()
		}
		fmt.Println("cfw-updater " + updaterVersion)
	}
	fmt.Println()
}

func updateDmgShell(execPath, dmgPath string) string {
	return fmt.Sprintf(`
sudo rm -rf "%s"
VOLUME=$(hdiutil attach "%s" | grep Volumes | awk -F " " '{for (i=3;i<=NF;i++)printf("%%s ", $i);print ""}'|awk '$1=$1')
sudo cp -rf "$VOLUME"/*.app /Applications/
hdiutil detach "$VOLUME" >/dev/null
sudo xattr -r -d com.apple.quarantine "%s"
`, execPath, dmgPath, execPath)
}

func updateInstallVersion(diList []*downloadInfo, stopCh chan struct{}) {
	if updateCore {
		startBackground()
		if runtime.GOOS == "darwin" {
			execCommand(updateDmgShell(ci.rootPath, fullPath(diList[0].fileFullName)))
		} else {
			go exec.Command(fullPath(diList[0].fileFullName), "/S").Run()
			for {
				if check := checkCfw(); check != nil {
					check.process.Kill()
					break
				}
			}
		}
	}

	if updateTrans {
		if runtime.GOOS == "darwin" {
			execCommand(fmt.Sprintf("sudo cp -rp \"%s\" \"%s\"", fullPath(path.Join(diList[len(diList)-1].fileName, "app.asar")), path.Join(ci.rootPath, "Contents/Resources/app.asar")))
		} else {
			if err := copy.Copy(fullPath(path.Join(diList[len(diList)-1].fileName, "app.asar")), path.Join(ci.rootPath, "resources/app.asar")); err != nil {
				close(stopCh)
				fmt.Printf("\n\n请尝试以管理员身份运行此程序:\n")
				exit(err.Error())
			}
		}
	}
}

func updatePortableVersion(diList []*downloadInfo, stopCh chan struct{}) {
	if updateTrans {
		if err := copy.Copy(fullPath(path.Join(diList[len(diList)-1].fileName, "app.asar")), fullPath(path.Join(diList[0].fileName, "resources/app.asar"))); err != nil {
			close(stopCh)
			fmt.Printf("\n\n请尝试以管理员身份运行此程序:\n")
			exit(err.Error())
		}
	}
	if updateCore {
		if ci.portableData {
			if err := copy.Copy(ci.rootPath+"/data", fullPath(diList[0].fileName+"/data")); err != nil {
				close(stopCh)
				fmt.Printf("\n\n")
				exit(err.Error())
			}
		}
		for {
			if err := os.RemoveAll(ci.rootPath); err == nil {
				break
			}
		}
		if err := copy.Copy(fullPath(diList[0].fileName), ci.rootPath); err != nil {
			close(stopCh)
			fmt.Printf("\n\n")
			exit(err.Error())
		}
	}
}

func updateCfw(diList []*downloadInfo) {
	stopCh := make(chan struct{})
	if updateCore || updateTrans {
		if runtime.GOOS == "darwin" {
			// 提前获取mac os用于更新的密码
			fmt.Println("请输入更新所需的密码(有权限无需密码的会跳过):")
			execCommand("sudo echo >/dev/null")
			fmt.Println()
		}
		stopCh = make(chan struct{})
		go showProgress("更新cfw中", stopCh)
		ci.process.Kill()
	}
	if ci.installVersion {
		updateInstallVersion(diList, stopCh)
	} else {
		updatePortableVersion(diList, stopCh)
	}
	if updateCore || updateTrans {
		if !ci.installVersion {
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
		fmt.Printf("\n\n更新成功!\n\n")
	}
}
