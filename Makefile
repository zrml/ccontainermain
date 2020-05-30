# Makefile for a project

BINARY=ccontainermain

OS=linux
ARCH=amd64
LDFLAGS=

.PHONY: all ccontainermain get-deps clean

all: ccontainermain

ccontainermain: get-deps distrib/${OS}/ccontainermain

distrib/${OS}/ccontainermain: ccontainermain.go
	env GOOS=${OS} GOARCH=${ARCH} go build ${LDFLAGS} -o distrib/${OS}/${BINARY} ccontainermain.go

get-deps:
	go get github.com/hpcloud/tail

clean:
	-rm distrib/${OS}/ccontainermain
