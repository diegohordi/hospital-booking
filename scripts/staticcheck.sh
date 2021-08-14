#!/bin/bash

go get honnef.co/go/tools/cmd/staticcheck
go mod vendor
staticcheck -checks all ./...