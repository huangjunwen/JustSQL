
OS := $(shell uname -s)
MACHINE := $(shell uname -m)
VER := $(shell git tag --points-at HEAD)
ifeq "$(VER)" ""
	VER := $(shell git rev-parse --short HEAD)
endif

ifneq "$(GOOS)" ""
	TARGET := bin/justsql-$(VER)-$(GOOS)-$(GOARCH)
else
	TARGET := bin/justsql-$(VER)-$(OS)-$(MACHINE)
endif

GOBUILD := CGO_ENABLED=0 go build
LDFLAGS += -X "github.com/huangjunwen/JustSQL/utils.BuildTS=$(shell date)"
LDFLAGS += -X "github.com/huangjunwen/JustSQL/utils.GitHash=$(shell git rev-parse HEAD)"

all: $(TARGET).tgz

$(TARGET).tgz: $(TARGET)
	tar -zcvf $(TARGET).tgz $(TARGET)

$(TARGET):
	GOOS=$(GOOS) GOARCH=$(GOARCH) $(GOBUILD) -ldflags '$(LDFLAGS)' -o $(TARGET) "github.com/huangjunwen/JustSQL/justsql"
