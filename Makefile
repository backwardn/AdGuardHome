GIT_VERSION := $(shell git describe --abbrev=4 --dirty --always --tags)
NATIVE_GOOS = $(shell unset GOOS; go env GOOS)
NATIVE_GOARCH = $(shell unset GOARCH; go env GOARCH)
GOPATH := $(shell go env GOPATH)
JSFILES = $(shell find client -path client/node_modules -prune -o -type f -name '*.js')
STATIC = build/static/index.html
CHANNEL ?= release
THISPATH := $(shell realpath .)
MAC_XCOMPILER_PATH := .

TARGET=AdGuardHome

.PHONY: all build clean
all: build

build: $(TARGET)

client/node_modules: client/package.json client/package-lock.json
	npm --prefix client install
	touch client/node_modules

$(STATIC): $(JSFILES) client/node_modules
	npm --prefix client run build-prod

$(TARGET): $(STATIC) *.go home/*.go dhcpd/*.go dnsfilter/*.go dnsforward/*.go
	GOOS=$(NATIVE_GOOS) GOARCH=$(NATIVE_GOARCH) GO111MODULE=off go get -v github.com/gobuffalo/packr/...
	PATH=$(GOPATH)/bin:$(PATH) packr -z
	CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=$(GIT_VERSION) -X main.channel=$(CHANNEL)" -asmflags="-trimpath=$(PWD)" -gcflags="-trimpath=$(PWD)"
	PATH=$(GOPATH)/bin:$(PATH) packr clean

# This target calls "go build" and is used internally by other more specific "unbound" targets
_unbound:
	CGO_ENABLED=0 GOOS=$(NATIVE_GOOS) GOARCH=$(NATIVE_GOARCH) GO111MODULE=off go get -v github.com/gobuffalo/packr/...
	PATH=$(GOPATH)/bin:$(PATH) packr -z
	PATH=$(_PATH):$(PATH) \
		CGO_ENABLED=1 \
		GOOS=$(_GOOS) GOARCH=$(_GOARCH) GOARM=$(_GOARM) \
		CC=$(_CC) \
		go build \
		-tags agh_mod_unbound
	PATH=$(GOPATH)/bin:$(PATH) packr clean

# build on Ubuntu or Fedora for itself (AMD64)
unbound-linux:
	CGO_CFLAGS="-I$(THISPATH)/unbound" \
		CGO_LDFLAGS="-L$(THISPATH)/unbound/ubuntu-amd64 -Wl,-rpath-link,$(THISPATH)/unbound/ubuntu-amd64" \
		make -f Makefile _unbound

# build on Fedora for ARM
unbound-fedora-arm:
	CGO_CFLAGS="-I/usr/arm-linux-gnu/include -I$(THISPATH)/unbound" \
		CGO_LDFLAGS="-L$(THISPATH)/unbound/raspbian-arm -Wl,-rpath-link,$(THISPATH)/unbound/raspbian-arm" \
		make -f Makefile _unbound \
		_GOOS=linux _GOARCH=arm _GOARM=6 \
		_CC=arm-linux-gnu-gcc

# build on Ubuntu for ARM
unbound-ubuntu-arm:
	CGO_CFLAGS="-I/usr/arm-linux-gnu/include -I$(THISPATH)/unbound" \
		CGO_LDFLAGS="-L$(THISPATH)/unbound/raspbian-arm -Wl,-rpath-link,$(THISPATH)/unbound/raspbian-arm" \
		make -f Makefile _unbound \
		_GOOS=linux _GOARCH=arm _GOARM=6 \
		_CC=arm-linux-gnueabihf-gcc

# build on macOS for ARM
unbound-mac-arm:
	CGO_CFLAGS="-I$(MAC_XCOMPILER_PATH)/arm-none-linux-gnueabi/sysroot/usr/include -I$(THISPATH)/unbound" \
		CGO_LDFLAGS="-L$(THISPATH)/unbound/raspbian-arm -Wl,-rpath-link,$(THISPATH)/unbound/raspbian-arm" \
		make -f Makefile _unbound \
		_GOOS=linux _GOARCH=arm _GOARM=6 \
		_CC=arm-none-linux-gnueabi-gcc \
		_PATH=$(MAC_XCOMPILER_PATH)/bin

clean:
	$(MAKE) cleanfast
	rm -rf build
	rm -rf client/node_modules

cleanfast:
	rm -f $(TARGET)
