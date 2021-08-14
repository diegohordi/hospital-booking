#!/bin/bash

go get github.com/kisielk/errcheck
go mod vendor
errcheck ./...