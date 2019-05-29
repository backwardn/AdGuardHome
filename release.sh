#!/usr/bin/env bash

set -eE
set -o pipefail
set -x

channel=${1:-release}
baseUrl="https://static.adguard.com/adguardhome/$channel"
dst=dist
version=`git describe --abbrev=4 --dirty --always --tags`

f() {
	make cleanfast; CGO_DISABLED=1 make
	if [[ $GOOS == darwin ]]; then
	    zip $dst/AdGuardHome_MacOS.zip AdGuardHome README.md LICENSE.txt
	elif [[ $GOOS == windows ]]; then
	    zip $dst/AdGuardHome_Windows_"$GOARCH".zip AdGuardHome.exe README.md LICENSE.txt
	else
	    rm -rf dist/AdguardHome
	    mkdir -p dist/AdGuardHome
	    cp -pv {AdGuardHome,LICENSE.txt,README.md} dist/AdGuardHome/
	    pushd dist
	    tar zcvf AdGuardHome_"$GOOS"_"$GOARCH".tar.gz AdGuardHome/
	    popd
	    rm -rf dist/AdguardHome
	fi
}

unbound_linux() {
	make cleanfast
	make unbound-linux
	rm -rf dist/AdguardHome
	mkdir -p dist/AdGuardHome
	cp -pv AdGuardHome libunbound/linux-"$GOARCH"/* LICENSE.txt README.md dist/AdGuardHome/
	pushd dist
	tar zcvf AdGuardHome_"$GOOS"_"$GOARCH".tar.gz AdGuardHome/
	popd
	rm -rf dist/AdguardHome
}

# Clean dist and build
make clean
rm -rf $dst

# Prepare the dist folder
mkdir -p $dst

# Try to build AGH with libunbound
LINUX_DISTRIB=`grep '^ID=' /etc/os-release`
CPREFIX=
if [ $LINUX_DISTRIB == "ID=fedora" ] || [ $LINUX_DISTRIB == "ID=\"centos\"" ]; then
	CPREFIX=arm-linux-gnu-
fi
if [ $LINUX_DISTRIB == "ID=ubuntu" ] || [ $LINUX_DISTRIB == "ID=debian" ]; then
	CPREFIX=arm-linux-gnueabihf-
fi
if [ $CPREFIX != "" ]; then
	CHANNEL=$channel GOOS=linux GOARCH=amd64 UNBOUND_BIN_DIR=linux-amd64 CPREFIX=$CPREFIX unbound_linux
	CHANNEL=$channel GOOS=linux GOARCH=arm GOARM=6 UNBOUND_BIN_DIR=linux-arm CPREFIX=$CPREFIX unbound_linux
else
	CHANNEL=$channel GOOS=linux GOARCH=amd64 f
	CHANNEL=$channel GOOS=linux GOARCH=arm GOARM=6 f
fi

# Prepare releases
CHANNEL=$channel GOOS=darwin GOARCH=amd64 f
CHANNEL=$channel GOOS=linux GOARCH=386 f
CHANNEL=$channel GOOS=linux GOARCH=arm64 GOARM=6 f
CHANNEL=$channel GOOS=windows GOARCH=amd64 f
CHANNEL=$channel GOOS=windows GOARCH=386 f
CHANNEL=$channel GOOS=linux GOARCH=mipsle GOMIPS=softfloat f
CHANNEL=$channel GOOS=linux GOARCH=mips GOMIPS=softfloat f

# Variables for CI
echo "version=$version" > $dst/version.txt

# Prepare the version.json file
echo "{" >> $dst/version.json
echo "  \"version\": \"$version\"," >> $dst/version.json
echo "  \"announcement\": \"AdGuard Home $version is now available!\"," >> $dst/version.json
echo "  \"announcement_url\": \"https://github.com/AdguardTeam/AdGuardHome/releases\"," >> $dst/version.json
echo "  \"download_windows_amd64\": \"$baseUrl/AdGuardHome_Windows_amd64.zip\"," >> $dst/version.json
echo "  \"download_windows_386\": \"$baseUrl/AdGuardHome_Windows_386.zip\"," >> $dst/version.json
echo "  \"download_darwin_amd64\": \"$baseUrl/AdGuardHome_MacOS.zip\"," >> $dst/version.json
echo "  \"download_linux_amd64\": \"$baseUrl/AdGuardHome_linux_amd64.tar.gz\"," >> $dst/version.json
echo "  \"download_linux_386\": \"$baseUrl/AdGuardHome_linux_386.tar.gz\"," >> $dst/version.json
echo "  \"download_linux_arm\": \"$baseUrl/AdGuardHome_linux_arm.tar.gz\"," >> $dst/version.json
echo "  \"download_linux_arm64\": \"$baseUrl/AdGuardHome_linux_arm64.tar.gz\"," >> $dst/version.json
echo "  \"download_linux_mips\": \"$baseUrl/AdGuardHome_linux_mips.tar.gz\"," >> $dst/version.json
echo "  \"download_linux_mipsle\": \"$baseUrl/AdGuardHome_linux_mipsle.tar.gz\"," >> $dst/version.json
echo "  \"selfupdate_min_version\": \"v0.0\"" >> $dst/version.json
echo "}" >> $dst/version.json
