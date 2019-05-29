# Build AdGuard Home with libunbound

## Build on Linux for itself (AMD64)

Install gcc (Ubuntu)

	apt-get install build-essential

Install gcc (Fedora)

	dnf install gcc make

Build AdGuard Home

	make UNBOUND_BIN_DIR=linux-amd64 unbound-linux


## Build on Linux for ARM

Install gcc (Ubuntu)

	apt-get install build-essential arm-linux-gnueabihf-gcc libc6-armel-cross libc6-dev-armel-cross binutils-arm-linux-gnueabi libncurses5-dev

Install gcc (Fedora)

	dnf install binutils-arm-linux-gnu glibc-arm-linux-gnu kernel-cross-headers

Build AdGuard Home (Ubuntu)

	make UNBOUND_BIN_DIR=linux-arm CPREFIX=arm-linux-gnueabihf- unbound-linux

Build AdGuard Home (Fedora)

	make UNBOUND_BIN_DIR=linux-arm CPREFIX=arm-linux-gnu- unbound-linux


## Build on macOS for ARM

Install gcc

* Download cross-compiler package: https://s3.amazonaws.com/jaredwolff/rpi-xtools-201402102110.dmg.zip

* Unpack .zip file, mount .dmg and copy its contents (`arm-none-linux-gnueabi` directory) to the directory where AdGuard Home source code is located.

Build AdGuard Home

	make unbound-mac-arm
