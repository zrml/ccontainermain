# Makefile for a project

BINARY=ccontainermain

OS=linux
ARCH=amd64
LDFLAGS= 

all: get-deps build
build:
	env GOOS=${OS} GOARCH=${ARCH} go build ${LDFLAGS} -o distrib/${OS}/${BINARY} ccontainermain.go

get-deps:
	go get github.com/hpcloud/tail
