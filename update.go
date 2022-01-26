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
			exec.Command("clear").Run()
		} else {
			c := exec.Command("cmd", "/c", "cls")
			c.Stdout = os.Stdout
			c.Run()
		}
		fmt.Println("cfw-updater " + updaterVersion)
	}
	fmt.Println()
}

func updateInstallVersion(diList []*downloadInfo, stopCh chan struct{}) {
	if updateCore {
		startBackground()
		if runtime.GOOS == "windows" {
			go exec.Command(fullPath(diList[0].fileFullName), "/S").Run()
			for {
				if check := checkCfw(); check != nil {
					check.process.Kill()
					break
				}
			}
		} else {
			execCommand("sudo rm -rf " + ci.rootPath)
			execCommand("hdiutil attach " + fullPath(diList[0].fileFullName))
			execCommand(fmt.Sprintf("cp -rf /Volumes/%s/*.app %s", diList[0].fileName, ci.rootPath))
			execCommand("hdiutil attach  /Volumes/" + diList[0].fileName)
			execCommand(fmt.Sprintf("sudo xattr -r -d com.apple.quarantine %s", ci.rootPath))
		}
	}

	if updateTrans {
		var transPath string
		if runtime.GOOS == "darwin" {
			transPath = path.Join(ci.rootPath, "Contents/Resources/app.asar")
		} else {
			transPath = path.Join(ci.rootPath, "resources/app.asar")
		}
		if err := copy.Copy(fullPath(path.Join(diList[len(diList)-1].fileName, "app.asar")), transPath); err != nil {
			close(stopCh)
			fmt.Printf("\n\n请尝试以管理员身份运行此程序:\n")
			exit(err.Error())
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
