
OS := $(shell uname -s)
MACHINE := $(shell uname -m)
VER := $(shell git tag --points-at HEAD)
ifeq "$(VER)" ""
	VER := $(shell git rev-parse --short HEAD)
endif

BIN := bin
TARGET := justsql

ifneq "$(GOOS)" ""
	TARGET_TARBALL := $(TARGET)-$(VER)-$(GOOS)-$(GOARCH).tgz
else
	TARGET_TARBALL := $(TARGET)-$(VER)-$(OS)-$(MACHINE).tgz
endif

GOBUILD := GOOS=$(GOOS) GOARCH=$(GOARCH) GOPATH=$(GOPATH)/src/github.com/pingcap/tidb/_vendor:$(GOPATH) CGO_ENABLED=0 go build
LDFLAGS += -X "github.com/huangjunwen/JustSQL/utils.BuildTS=$(shell date)"
LDFLAGS += -X "github.com/huangjunwen/JustSQL/utils.GitHash=$(shell git rev-parse HEAD)"

all: help
	
.PHONY: help

help:
	@echo "Use 'GOOS=xxx GOARCH=xxx make release' to build release. GOOS and GOARCH can be empty."
	
release: $(BIN)/$(TARGET_TARBALL)

$(BIN)/$(TARGET_TARBALL): $(BIN)/$(TARGET)
	cd $(BIN) && tar -zcvf $(TARGET_TARBALL) $(TARGET) && rm $(TARGET)

$(BIN)/$(TARGET):
	$(GOBUILD) -ldflags '$(LDFLAGS)' -o $(BIN)/$(TARGET) "github.com/huangjunwen/JustSQL/justsql"
