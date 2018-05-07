#!/bin/bash
GOOS=linux GOARCH=386 CGO_ENABLED=0 go build -o ./distrib/linux/ccontainermain ccontainermain.go
