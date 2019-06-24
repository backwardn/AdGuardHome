Contents:
* Build AdGuard Home with libunbound support
	* Build on Linux for itself (AMD64)
	* Build on Ubuntu for ARM
	* Build on Fedora for ARM
	* Build on macOS for ARM
* Configure host where AdGuard Home with libunbound will be running


## Build AdGuard Home with libunbound support

AdGuard Home can be built with its own DNS resolver so no other upstream DNS server is required for AdGuard Home to work.  This feature can be enabled at compile time only meaning that you must build it manually.  Also, it will work only when there are no upstream servers configured.

To build AdGuard Home with support for libunbound, `cgo` must be used which means that the system must have gcc installed.  There is no need to install a development package for libunbound because we provide all necessary files: C include files and compiled binaries are located in `unbound/` directory.


### Build on Linux for itself (AMD64)

Install gcc (Ubuntu)

	apt-get install build-essential

Install gcc (Fedora)

	dnf install gcc make

Build AdGuard Home

	make unbound-linux


### Build on Ubuntu for ARM

Install gcc

	apt-get install build-essential arm-linux-gnueabihf-gcc libc6-armel-cross libc6-dev-armel-cross binutils-arm-linux-gnueabi libncurses5-dev

Build AdGuard Home

	make unbound-ubuntu-arm


### Build on Fedora for ARM

Install gcc

	dnf install binutils-arm-linux-gnu glibc-arm-linux-gnu kernel-cross-headers

Build AdGuard Home

	make unbound-fedora-arm


### Build on macOS for ARM

Install gcc

* Download cross-compiler package: https://s3.amazonaws.com/jaredwolff/rpi-xtools-201402102110.dmg.zip

* Unpack .zip file, mount .dmg and copy its contents (`arm-none-linux-gnueabi` directory) to the directory where AdGuard Home source code is located.

Build AdGuard Home

	make unbound-mac-arm


## Configure host where AdGuard Home with libunbound will be running

When AdGuard Home is built with libunbound support, it dynamically links to libunbound module which must be installed on your system.

Fedora:

	dnf install unbound-libs
	
Ubuntu:

	apt-get install libunbound2

Raspbian:

	apt-get install libunbound

To check if all dependencies are satisfied use this command:

	ldd ./AdGuardHome

which outputs something like this:

	linux-vdso.so.1 (0x7efda000)
	/usr/lib/arm-linux-gnueabihf/libarmmem.so (0x76f43000)
	libpthread.so.0 => /lib/arm-linux-gnueabihf/libpthread.so.0 (0x76f07000)
	libunbound.so.2 => /usr/lib/arm-linux-gnueabihf/libunbound.so.2 (0x76e5a000)
	libc.so.6 => /lib/arm-linux-gnueabihf/libc.so.6 (0x76d1b000)
	/lib/ld-linux-armhf.so.3 (0x76f59000)
	libhogweed.so.4 => /usr/lib/arm-linux-gnueabihf/libhogweed.so.4 (0x76cde000)
	libnettle.so.6 => /usr/lib/arm-linux-gnueabihf/libnettle.so.6 (0x76c97000)
	libgmp.so.10 => /usr/lib/arm-linux-gnueabihf/libgmp.so.10 (0x76c24000)
