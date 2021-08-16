#!/bin/bash

go install golang.org/x/tools/go/analysis/passes/shadow/cmd/shadow@latest
shadow ./...