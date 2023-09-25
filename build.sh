#!/bin/bash

github_token=""

project="Jrohy/cfw-updater"

#获取当前的这个脚本所在绝对路径
shell_path=$(cd `dirname $0`; pwd)

export CGO_ENABLED=1

function uploadfile() {
    file=$1

    ctype=$(file -b --mime-type $file)

    curl -H "Authorization: token ${github_token}" -H "Content-Type: ${ctype}" --data-binary @$file "https://uploads.github.com/repos/$project/releases/${release_id}/assets?name=$(basename $file)"

    echo ""
}

function upload() {
    file=$1
    dgst=$1.dgst
    openssl dgst -md5 $file | sed 's/([^)]*)//g' >> $dgst
    openssl dgst -sha1 $file | sed 's/([^)]*)//g' >> $dgst
    openssl dgst -sha256 $file | sed 's/([^)]*)//g' >> $dgst
    openssl dgst -sha512 $file | sed 's/([^)]*)//g' >> $dgst
    uploadfile $file
    uploadfile $dgst
}

[[ -z `command -v goversioninfo` ]] && go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest

version=`git describe --tags $(git rev-list --tags --max-count=1)`
now=`date "+%Y%m%d-%H%M"`
go_version=`go version|awk '{print $3,$4}'`
git_version=`git rev-parse HEAD`
ldflags="-w -s -X 'main.version=$version' -X 'main.buildDate=$now' -X 'main.goversion=$go_version' -X 'main.gitversion=$git_version'"

v=`echo "$version"|sed 's/v//g'`
patch_version=`echo $v|cut -d . -f3`
[[ -z $patch_version ]] && patch_version=-1
exe_version_info="-product-version $version -ver-major `echo $v|cut -d . -f1` -ver-minor `echo $v|cut -d . -f2` -ver-patch $patch_version"
goversioninfo -skip-versioninfo -64 -icon *.ico -copyright "Copyright © 2022 Jrohy" -product-version $version -product-name "cfw-updater" -description "Clash for Windows便携版更新工具" $exe_version_info
GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc go build -ldflags "$ldflags" -o result/cfw-updater.exe .
rm -f resource.syso

if [[ `uname` == "Darwin" ]];then
    GOOS=darwin GOARCH=arm64 go build -ldflags "$ldflags" -o result/cfw-updater_mac_arm64 .
    GOOS=darwin GOARCH=amd64 go build -ldflags "$ldflags" -o result/cfw-updater_mac_amd64 .
fi

if [[ $# == 0 ]];then

    cd result

    upload_item=($(ls -l|awk '{print $9}'|xargs -r))

    curl -X POST -H "Authorization: token ${github_token}" -H "Accept: application/vnd.github.v3+json" https://api.github.com/repos/$project/releases -d '{"tag_name":"'$version'", "name":"'$version'"}'

	sleep 2

	release_id=`curl -H 'Cache-Control: no-cache' -s https://api.github.com/repos/$project/releases/latest|grep id|awk 'NR==1{print $2}'|sed 's/,//'`

    for item in ${upload_item[@]}
    do
        upload $item
    done

    echo "upload completed!"

    cd $shell_path

    rm -rf result
fi
