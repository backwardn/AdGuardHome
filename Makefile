GIT_VERSION := $(shell git describe --abbrev=4 --dirty --always --tags)
NATIVE_GOOS = $(shell unset GOOS; go env GOOS)
NATIVE_GOARCH = $(shell unset GOARCH; go env GOARCH)
GOPATH := $(shell go env GOPATH)
JSFILES = $(shell find client -path client/node_modules -prune -o -type f -name '*.js')
STATIC = build/static/index.html
CHANNEL ?= release
THISPATH := $(shell realpath .)
MAC_XCOMPILER_PATH := .
UNBOUND_INC_PATH := $(THISPATH)/libunbound
UNBOUND_BIN_DIR ?= linux-amd64
UNBOUND_BIN_PATH := $(THISPATH)/libunbound/$(UNBOUND_BIN_DIR)
CPREFIX ?= arm-linux-gnueabihf-

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

# Download libunbound's include and binary files
libunbound:
	curl https://cdn.adguard.com/public/Resources/libunbound_adguard_home.zip -o libunbound_adguardhome.zip
	unzip libunbound_adguardhome.zip

# This target calls "go build" and is used internally by other more specific "unbound" targets
_unbound: libunbound
	CGO_ENABLED=0 GOOS=$(NATIVE_GOOS) GOARCH=$(NATIVE_GOARCH) GO111MODULE=off go get -v github.com/gobuffalo/packr/...
	PATH=$(GOPATH)/bin:$(PATH) packr -z
	PATH=$(_PATH):$(PATH) \
		CGO_ENABLED=1 \
		GOOS=$(_GOOS) GOARCH=$(_GOARCH) GOARM=$(_GOARM) \
		CC=$(_CC) \
		go build \
		-ldflags="-s -w -X main.version=$(GIT_VERSION) -X main.channel=$(CHANNEL)" \
		-asmflags="-trimpath=$(PWD)" \
		-gcflags="-trimpath=$(PWD)" \
		-tags agh_mod_unbound
	PATH=$(GOPATH)/bin:$(PATH) packr clean

# build on Linux for AMD64 or ARM
unbound-linux:
ifeq ($(UNBOUND_BIN_DIR),linux-amd64)
	CGO_CFLAGS="-I$(UNBOUND_INC_PATH)" \
		CGO_LDFLAGS="-L$(UNBOUND_BIN_PATH)"' -Wl,-rpath,$$ORIGIN' \
		make -f Makefile _unbound
else ifeq ($(UNBOUND_BIN_DIR),linux-arm)
	CGO_CFLAGS="-I/usr/arm-linux-gnu/include -I$(UNBOUND_INC_PATH)" \
		CGO_LDFLAGS="-L$(UNBOUND_BIN_PATH) -Wl,-rpath-link,$(UNBOUND_BIN_PATH)"' -Wl,-rpath,$$ORIGIN' \
		make -f Makefile _unbound \
		_GOOS=linux _GOARCH=arm _GOARM=6 \
		_CC=$(CPREFIX)gcc
endif

# build on macOS for Linux ARM
unbound-mac-arm:
	CGO_CFLAGS="-I$(MAC_XCOMPILER_PATH)/arm-none-linux-gnueabi/sysroot/usr/include -I$(UNBOUND_INC_PATH)" \
		CGO_LDFLAGS="-L$(UNBOUND_BIN_PATH) -Wl,-rpath-link,$(UNBOUND_BIN_PATH)" \
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
