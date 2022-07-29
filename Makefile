.PHONY: default build docker-image test stop clean-images clean

BINARY = quarkcm

VERSION=
BUILD=

PKG            = github.com/CentaurusInfra/quarkcm
GOCMD          = go
BUILD_DATE     = `date +%FT%T%z`
GOFLAGS       ?= $(GOFLAGS:)
LDFLAGS       := "-X '$(PKG)/cmd.buildDate=$(BUILD_DATE)'"

default: build build_cni build_quarkcmclient test

build: proto
	"$(GOCMD)" build ${GOFLAGS} -ldflags ${LDFLAGS} -o "${BINARY}"

docker-image:
	@docker build -t "${BINARY}" .

test:
	"$(GOCMD)" test -race -v ./...

stop:
	@docker stop "${BINARY}"

clean-images: stop
	@docker rmi "${BUILDER}" "${BINARY}"

clean:
	"$(GOCMD)" clean -i
	rm -rf build
	rm -rf quarkcmclient/target

proto:
	protoc --go_out=./ --go-grpc_out=. pkg/connectionmanager/grpc/quarkcmsvc.proto
	mv pkg/grpc/* pkg/connectionmanager/grpc
	rm -rf pkg/grpc

.PHONY: build_cni
build_cni:
	GO111MODULE="on" go build cmd/quarkcni/quarkcni.go; mkdir -p build/bin; mv quarkcni build/bin

.PHONY: build_quarkcmclient
build_quarkcmclient:
	cd quarkcmclient; cargo build
