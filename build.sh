#!/bin/bash

GITHUB_TOKEN=""

PROJECT="Jrohy/cfw-updater"

#获取当前的这个脚本所在绝对路径
SHELL_PATH=$(cd `dirname $0`; pwd)

export CGO_ENABLED=1

function uploadfile() {
    FILE=$1

    CTYPE=$(file -b --mime-type $FILE)

    curl -H "Authorization: token ${GITHUB_TOKEN}" -H "Content-Type: ${CTYPE}" --data-binary @$FILE "https://uploads.github.com/repos/$PROJECT/releases/${RELEASE_ID}/assets?name=$(basename $FILE)"

    echo ""
}

function upload() {
    FILE=$1
    DGST=$1.dgst
    openssl dgst -md5 $FILE | sed 's/([^)]*)//g' >> $DGST
    openssl dgst -sha1 $FILE | sed 's/([^)]*)//g' >> $DGST
    openssl dgst -sha256 $FILE | sed 's/([^)]*)//g' >> $DGST
    openssl dgst -sha512 $FILE | sed 's/([^)]*)//g' >> $DGST
    uploadfile $FILE
    uploadfile $DGST
}

[[ -z `command -v goversioninfo` ]] && go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest

VERSION=`git describe --tags $(git rev-list --tags --max-count=1)`
NOW=`date "+%Y%m%d-%H%M"`
GO_VERSION=`go version|awk '{print $3,$4}'`
GIT_VERSION=`git rev-parse HEAD`
LDFLAGS="-w -s -X 'main.version=$VERSION' -X 'main.buildDate=$NOW' -X 'main.goVersion=$GO_VERSION' -X 'main.gitVersion=$GIT_VERSION'"

V=`echo "$VERSION"|sed 's/v//g'`
PATCH_VERSION=`echo $V|cut -d . -f3`
[[ -z $PATCH_VERSION ]] && PATCH_VERSION=-1
EXE_VERSION_INFO="-product-version $VERSION -ver-major `echo $V|cut -d . -f1` -ver-minor `echo $V|cut -d . -f2` -ver-patch $PATCH_VERSION"
goversioninfo -skip-versioninfo -64 -icon *.ico -copyright "Copyright © 2022 Jrohy" -product-version $VERSION -product-name "cfw-updater" -description "Clash for Windows便携版更新工具" $EXE_VERSION_INFO

GOOS=windows GOARCH=amd64 go build -ldflags "$LDFLAGS" -o result/cfw-updater.exe .

rm -f resource.syso

if [[ $# == 0 ]];then

    cd result

    UPLOAD_ITEM=($(ls -l|awk '{print $9}'|xargs -r))

    curl -X POST -H "Authorization: token ${GITHUB_TOKEN}" -H "Accept: application/vnd.github.v3+json" https://api.github.com/repos/$PROJECT/releases -d '{"tag_name":"'$VERSION'", "name":"'$VERSION'"}'

	sleep 2

	RELEASE_ID=`curl -H 'Cache-Control: no-cache' -s https://api.github.com/repos/$PROJECT/releases/latest|grep id|awk 'NR==1{print $2}'|sed 's/,//'`

    for ITEM in ${UPLOAD_ITEM[@]}
    do
        upload $ITEM
    done

    echo "upload completed!"

    cd $SHELL_PATH

    rm -rf result
fi
