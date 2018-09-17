include .project/go-project.mk

.PHONY: *

.SILENT:

default: help

all: clean gopath tools generate covtest

gettools:
	mkdir -p ${TOOLS_SRC}
	$(call httpsclone,${GITHUB_HOST},golang/tools,             ${TOOLS_SRC}/golang.org/x/tools,                  release-branch.go1.10)
	$(call httpsclone,${GITHUB_HOST},jteeuwen/go-bindata,      ${TOOLS_SRC}/github.com/jteeuwen/go-bindata,      6025e8de665b31fa74ab1a66f2cddd8c0abf887e)
	$(call httpsclone,${GITHUB_HOST},jstemmer/go-junit-report, ${TOOLS_SRC}/github.com/jstemmer/go-junit-report, 385fac0ced9acaae6dc5b39144194008ded00697)
	$(call httpsclone,${GITHUB_HOST},go-phorce/cov-report,     ${TOOLS_SRC}/github.com/go-phorce/cov-report,     master)
	$(call httpsclone,${GITHUB_HOST},golang/lint,              ${TOOLS_SRC}/golang.org/x/lint,                   06c8688daad7faa9da5a0c2f163a3d14aac986ca)
	#$(call httpsclone,${GITHUB_HOST},golangci/golangci-lint,   ${TOOLS_SRC}/github.com/golangci/golangci-lint,   master)
	$(call httpsclone,${GITHUB_HOST},mattn/goveralls,          ${TOOLS_SRC}/github.com/mattn/goveralls,          88fc0d50edb2e4cf09fe772457b17d6981826cff)

tools: gettools
	GOPATH=${TOOLS_PATH} go install golang.org/x/tools/cmd/stringer
	GOPATH=${TOOLS_PATH} go install golang.org/x/tools/cmd/gorename
	GOPATH=${TOOLS_PATH} go install golang.org/x/tools/cmd/godoc
	GOPATH=${TOOLS_PATH} go install golang.org/x/tools/cmd/guru
	GOPATH=${TOOLS_PATH} go install github.com/jteeuwen/go-bindata/...
	GOPATH=${TOOLS_PATH} go install github.com/jstemmer/go-junit-report
	GOPATH=${TOOLS_PATH} go install github.com/go-phorce/cov-report/cmd/cov-report
	GOPATH=${TOOLS_PATH} go install golang.org/x/lint/golint
	#GOPATH=${TOOLS_PATH} go install github.com/golangci/golangci-lint/cmd/golangci-lint
	GOPATH=${TOOLS_PATH} go install github.com/mattn/goveralls

build:
	echo "Build skipped for pkg"