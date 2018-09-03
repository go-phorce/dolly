include .project/common.mk

GOFILES = $(shell find . -type f -name '*.go')
GOFILES_NOVENDOR = $(shell find . -type f -name '*.go' -not -path "./vendor/*" -not -path "./.tools/*" -not -path "./.gopath/*")

# location for vendor files
VENDOR_SRC=vendor
DOCKER_BIN=.docker

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

all: clean gopath vendor tools generate test

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
	$(call gitclone,${GITHUB_HOST},golang/lint,              ${TOOLS_SRC}/golang.org/x/lint,                   06c8688daad7faa9da5a0c2f163a3d14aac986ca)
	$(call gitclone,${GITHUB_HOST},jteeuwen/go-bindata,      ${TOOLS_SRC}/github.com/jteeuwen/go-bindata,      v3.0.7)
	$(call gitclone,${GITHUB_HOST},jstemmer/go-junit-report, ${TOOLS_SRC}/github.com/jstemmer/go-junit-report, 385fac0ced9acaae6dc5b39144194008ded00697)
	$(call gitclone,${GITHUB_HOST},ekspand/cov-report,       ${TOOLS_SRC}/github.com/go-phorce/cov-report,     master)

tools: gettools
	GOPATH=${TOOLS_PATH} go install golang.org/x/tools/cmd/stringer
	GOPATH=${TOOLS_PATH} go install golang.org/x/tools/cmd/gorename
	GOPATH=${TOOLS_PATH} go install golang.org/x/tools/cmd/godoc
	GOPATH=${TOOLS_PATH} go install golang.org/x/tools/cmd/guru
	GOPATH=${TOOLS_PATH} go install github.com/golang/lint/golint
	GOPATH=${TOOLS_PATH} go install github.com/jteeuwen/go-bindata/...
	GOPATH=${TOOLS_PATH} go install github.com/jstemmer/go-junit-report
	GOPATH=${TOOLS_PATH} go install github.com/go-phorce/cov-report/cmd/cov-report

getdevtools:
	$(call gitclone,${GITHUB_HOST},golang/tools,                ${GOPATH}/src/golang.org/x/tools,                  master)
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

get:
	$(call gitclone,${GITHUB_HOST},alecthomas/kingpin,    ${VENDOR_SRC}/gopkg.in/alecthomas/kingpin,      a39589180ebd6bbf43076e514b55f20a95d43086)
	$(call gitclone,${GITHUB_HOST},alecthomas/template,   ${VENDOR_SRC}/github.com/alecthomas/template,   a0175ee3bccc567396460bf5acd36800cb10c49c)
	$(call gitclone,${GITHUB_HOST},alecthomas/units,      ${VENDOR_SRC}/github.com/alecthomas/units,      2efee857e7cfd4f3d0138cc3cbb1b4966962b93a)
	$(call gitclone,${GITHUB_HOST},stretchr/testify,      ${VENDOR_SRC}/github.com/stretchr/testify,      4d4bfba8f1d1027c4fdbe371823030df51419987)
	$(call gitclone,${GITHUB_HOST},ugorji/go,             ${VENDOR_SRC}/github.com/ugorji/go,             5cd0f2b3b6cca8e3a0a4101821e41a73cb59bed6)
	$(call gitclone,${GITHUB_HOST},golang/crypto,         ${VENDOR_SRC}/golang.org/x/crypto,              453249f01cfeb54c3d549ddb75ff152ca243f9d8)
	$(call gitclone,${GITHUB_HOST},golang/net,            ${VENDOR_SRC}/golang.org/x/net,                 66aacef3dd8a676686c7ae3716979581e8b03c47)
	$(call gitclone,${GITHUB_HOST},golang/text,           ${VENDOR_SRC}/golang.org/x/text,                b19bf474d317b857955b12035d2c5acb57ce8b01)
	$(call gitclone,${GITHUB_HOST},juju/errors,           ${VENDOR_SRC}/github.com/juju/errors,           c7d06af17c68cd34c835053720b21f6549d9b0ee)
	$(call gitclone,${GITHUB_HOST},natefinch/lumberjack,  ${VENDOR_SRC}/gopkg.in/natefinch/lumberjack.v2, 514cbda263a734ae8caac038dadf05f8f3f9f738)

vendor: get

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
	cd ${TEST_DIR} && GOPATH=${TEST_GOPATH}  go list ./... | grep -v /vendor/ | xargs -L1 ${TOOLS_BIN}/golint -set_exit_status
	# cd ${TEST_DIR} && GOPATH=${TEST_GOPATH} ${PROJROOT}/golint.sh ${TOOLS_BIN}/golint -set_exit_status ${PROJECT_DIRS}

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
	${TOOLS_BIN}/cov-report -fmt xml -o ${COVPATH}/coverage.xml -ex ${COVERAGE_EXCLUSIONS} -cc ${COVPATH}/combined.out ${COVPATH}/cc*.out
	${TOOLS_BIN}/cov-report -fmt ds -o ${COVPATH}/summary.xml -ex ${COVERAGE_EXCLUSIONS} ${COVPATH}/cc*.out

