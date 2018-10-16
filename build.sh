#!/bin/bash
# script to build linux exe from any GO supported arch.
go get github.com/hpcloud/tail
GOOS=linux GOARCH=386 CGO_ENABLED=0 go build -o ./distrib/linux/ccontainermain ccontainermain.go
