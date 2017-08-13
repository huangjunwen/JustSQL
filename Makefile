GOBUILD := CGO_ENABLED=0 go build
LDFLAGS += -X "github.com/huangjunwen/JustSQL/utils.BuildTS=$(shell date)"
LDFLAGS += -X "github.com/huangjunwen/JustSQL/utils.GitHash=$(shell git rev-parse HEAD)"
OUT := bin/justsql

all: $(OUT)

$(OUT):
	$(GOBUILD) -ldflags '$(LDFLAGS)' -o $(OUT) "github.com/huangjunwen/JustSQL/justsql"
