include .project/go-project.mk

.PHONY: *

.SILENT:

default: help

all: clean gopath tools generate test

gettools:
	mkdir -p ${TOOLS_SRC}
	$(call gitclone,${GITHUB_HOST},golang/tools,             ${TOOLS_SRC}/golang.org/x/tools,                  release-branch.go1.10)
	$(call gitclone,${GITHUB_HOST},jteeuwen/go-bindata,      ${TOOLS_SRC}/github.com/jteeuwen/go-bindata,      6025e8de665b31fa74ab1a66f2cddd8c0abf887e)
	$(call gitclone,${GITHUB_HOST},jstemmer/go-junit-report, ${TOOLS_SRC}/github.com/jstemmer/go-junit-report, 385fac0ced9acaae6dc5b39144194008ded00697)
	$(call gitclone,${GITHUB_HOST},go-phorce/cov-report,     ${TOOLS_SRC}/github.com/go-phorce/cov-report,     master)
	$(call gitclone,${GITHUB_HOST},golang/lint,              ${TOOLS_SRC}/golang.org/x/lint,                   06c8688daad7faa9da5a0c2f163a3d14aac986ca)
	#$(call gitclone,${GITHUB_HOST},golangci/golangci-lint,   ${TOOLS_SRC}/github.com/golangci/golangci-lint,   master)

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

getdevtools:
	$(call gitclone,${GITHUB_HOST},golang/tools,                ${GOPATH}/src/golang.org/x/tools,                  release-branch.go1.10)
	$(call gitclone,${GITHUB_HOST},derekparker/delve,           ${GOPATH}/src/github.com/derekparker/delve,        master)
	$(call gitclone,${GITHUB_HOST},uudashr/gopkgs,              ${GOPATH}/src/github.com/uudashr/gopkgs,           master)
	$(call gitclone,${GITHUB_HOST},nsf/gocode,                  ${GOPATH}/src/github.com/nsf/gocode,               master)
	$(call gitclone,${GITHUB_HOST},rogpeppe/godef,              ${GOPATH}/src/github.com/rogpeppe/godef,           master)
	$(call gitclone,${GITHUB_HOST},acroca/go-symbols,           ${GOPATH}/src/github.com/acroca/go-symbols,        master)
	$(call gitclone,${GITHUB_HOST},ramya-rao-a/go-outline,      ${GOPATH}/src/github.com/ramya-rao-a/go-outline,   master)
	$(call gitclone,${GITHUB_HOST},ddollar/foreman,             ${GOPATH}/src/github.com/ddollar/foreman,          master)
	$(call gitclone,${GITHUB_HOST},sqs/goreturns,               ${GOPATH}/src/github.com/sqs/goreturns,            master)
	$(call gitclone,${GITHUB_HOST},karrick/godirwalk,           ${GOPATH}/src/github.com/karrick/godirwalk,        master)
	$(call gitclone,${GITHUB_HOST},pkg/errors,                  ${GOPATH}/src/github.com/pkg/errors,               master)

devtools: getdevtools
	go install golang.org/x/tools/cmd/fiximports
	go install golang.org/x/tools/cmd/goimports
	go install github.com/derekparker/delve/cmd/dlv
	go install github.com/uudashr/gopkgs/cmd/gopkgs
	go install github.com/nsf/gocode
	go install github.com/rogpeppe/godef
	go install github.com/acroca/go-symbols
	go install github.com/ramya-rao-a/go-outline
	go install github.com/sqs/goreturns

version:
	gofmt -r '"GIT_VERSION" -> "$(GIT_VERSION)"' version/current.template > version/current.go

build:
	echo "Build skipped for pkg"