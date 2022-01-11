# cfw-updater
![](https://img.shields.io/github/v/release/Jrohy/cfw-updater.svg) 
[![Downloads](https://img.shields.io/github/downloads/Jrohy/cfw-updater/total.svg)](https://img.shields.io/github/downloads/Jrohy/cfw-updater/total.svg) 
[![License](https://img.shields.io/badge/license-GPL%20V3-blue.svg?longCache=true)](https://www.gnu.org/licenses/gpl-3.0.en.html)  

[clash_for_windows](https://github.com/Fndroid/clash_for_windows_pkg)便携版的一键更新工具(仅限windows平台), 支持汉化补丁一起安装, 支持cfw的[便携模式](https://docs.cfw.lbyczf.com/contents/7z.html)和cfw版本的任意切换

当前可选的汉化列表(可任意切换):
- https://github.com/BoyceLig/Clash_Chinese_Patch
- https://github.com/ender-zhao/Clash-for-Windows_Chinese

## 要求
#### 已安装cfw便携版并能科学上网  
   更新工具会用到cfw的代理来下载最新包，所以下载安装包的速度取决于cfw节点的速度。此工具只能用于更新cfw而不能全新安装

## 命令行
```
Usage of cfw-updater.exe:
  -V    显示版本号
  -f    强制更新cfw(默认策略和最新版本相同则不更新)
  -v string
        指定cfw安装版本
```
