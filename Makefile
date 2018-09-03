include .project/common.mk

GOFILES = $(shell find . -type f -name '*.go')
GOFILES_NOVENDOR = $(shell find . -type f -name '*.go' -not -path "./vendor/*" -not -path "./.tools/*" -not -path "./.gopath/*")

COVERAGE_EXCLUSIONS="/rt\.go|/bindata\.go"

# flags
INTEGRATION_TAG="integration"
TEST_RACEFLAG ?=
TEST_GORACEOPTIONS ?=

# flag to enable golang race detector. Usage: make $(test_target) RACE=true. For example, make test RACE=true
RACE ?=
ifeq ($(RACE),true)
	TEST_GORACEOPTIONS = "log_path=${PROJROOT}/${COVPATH}/race/report"
	TEST_RACEFLAG = -race
endif

.PHONY: *

.SILENT:

default: all

vars:
	echo "HOSTNAME=$(HOSTNAME)"
	echo "PROJROOT=$(PROJROOT)"
	echo "GOROOT=$(GOROOT)"
	echo "GOPATH=$(GOPATH)"
	echo "VENDOR_SRC=$(VENDOR_SRC)"
	echo "PROJ_PACKAGE=$(PROJ_PACKAGE)"
	echo "PROJ_GOPATH=$(PROJ_GOPATH)"
	echo "TOOLS_PATH=$(TOOLS_PATH)"
	echo "TEST_GOPATH=$(TEST_GOPATH)"
	echo "TEST_DIR=$(TEST_DIR)"
	echo "VERSION=$(GIT_VERSION)"
	[ -d "${PROJ_REPO_TARGET}" ] && echo "Link exists: ${PROJ_REPO_TARGET}" || echo "Link does not exist: ${PROJ_REPO_TARGET}"

all: clean gopath tools generate test

clean:
	go clean
	rm -rf \
		${COVPATH} \
		bin

purge: clean
	rm -rf \
		${TOOLS_PATH} \
		${VENDOR_SRC}

gopath:
	@[ ! -d $(PROJ_REPO_TARGET) ] && \
		rm -f "${PROJ_REPO_TARGET}" && \
		mkdir -p "${PROJ_GOPATH_DIR}/src/${ORG_NAME}" && \
		ln -s ../../../.. "${PROJ_REPO_TARGET}" && \
		echo "Created link: ${PROJ_REPO_TARGET}" || \
	echo "Link already exists: ${PROJ_REPO_TARGET}"

showupdates:
	@$(call show_dep_updates,${TOOLS_SRC})
	@$(call show_dep_updates,${VENDOR_SRC})

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

generate:
	PATH=${TOOLS_BIN}:${PATH} go generate ./...

version:
	gofmt -r '"GIT_VERSION" -> "$(GIT_VERSION)"' version/current.template > version/current.go

listpkg: vars
	cd ${TEST_DIR} && go list ./...

vet:
	echo "Running vet"
	cd ${TEST_DIR} && go vet ./...

lint:
	echo "Running lint"
	cd ${TEST_DIR} && GOPATH=${TEST_GOPATH}  go list ./... | grep -v /vendor/ | xargs -L1 golint -set_exit_status
	# cd ${TEST_DIR} && GOPATH=${TEST_GOPATH} ${PROJROOT}/golint.sh golint -set_exit_status ${PROJECT_DIRS}

# print out the go environment
env:
	GOPATH=${GOPATH} go env

testenv:
	GOPATH=${TEST_GOPATH} go env

bench:
	echo "Running bench"
	GOPATH=${TEST_GOPATH} go test ${TEST_RACEFLAG} -bench . ${PROJ_PACKAGE}/...

fmt:
	echo "Running Fmt"
	gofmt -s -l -w ${GOFILES_NOVENDOR}

test: fmt vet lint
	echo "Running test"
	cd ${TEST_DIR} && go test ${TEST_RACEFLAG} ./...

testshort:
	cd ${TEST_DIR} && go test ${TEST_RACEFLAG} ./... --test.short

covtest: fmt vet lint
	$(call go_test_cover,${TEST_DIR},${TEST_GOPATH},${TEST_RACEFLAG},${TEST_GORACEOPTIONS},.,${COVERAGE_EXCLUSIONS})

# Runs integration tests as well
testint: vet lint
	GOPATH=${TEST_GOPATH} go test ${TEST_RACEFLAG} -tags=${INTEGRATION_TAG} ${PROJ_PACKAGE}/...

# shows the coverages results assuming they were already generated by a call to go_test_cover
coverage:
	GOPATH=${TEST_GOPATH} go tool cover -html=${COVPATH}/combined.out

# generates a HTML based code coverage report, and writes it to a file in the results directory
# assumes you've run go_test_cover (or go_test_cover_junit)
cicoverage:
	mkdir -p ${COVPATH}/cover
	GOPATH=${TEST_GOPATH} go tool cover -html=${COVPATH}/combined.out -o ${COVPATH}/cover/coverage.html

# as Jenkins runs citestint as well which will run all unit tests + integration tests with code coverage
# this unitest step can skip coverage reporting which speeds it up massively
citest: vet lint
	$(call go_test_cover_junit,${TEST_DIR},${GOPATH},${TEST_RACEFLAG},${TEST_GORACEOPTIONS},.,${COVERAGE_EXCLUSIONS})
	cov-report -fmt xml -o ${COVPATH}/coverage.xml -ex ${COVERAGE_EXCLUSIONS} -cc ${COVPATH}/combined.out ${COVPATH}/cc*.out
	cov-report -fmt ds -o ${COVPATH}/summary.xml -ex ${COVERAGE_EXCLUSIONS} ${COVPATH}/cc*.out

