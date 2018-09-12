# go-project.mk: this contains commonly used helpers for makefiles.
SHELL=/bin/bash

# Used envaronment variables:
#
# 	PROJ_DIR
#		project's absolute root directory
#
# 	PROJ_BIN
#		project's bin folder
#
# 	ORG_NAME
#		Git organization name, for example: github.com/go-phorce
#
#	PROJ_NAME
#		Git project name, for example: go-makefile
#
#	REPO_NAME
#		Git repo name consists of the org and project: github.com/go-phorce/go-makefile
#
#	PROJ_GOFILES
#		List of all .go files in the project, exluding vendor and tools
#
# Test flags:
#
#	TEST_RACEFLAG
#		Use -race when running go test
#
#	TEST_GORACEOPTIONS
#		Race options
#
# Functions:
#
#	show_dep_updates {folder}
#		Show dependencies updates in {folder}
#
#	gitclone {org} {repo} {destination_dir}
#
#	go_test_cover
#
#	go_test_cover_junit


PROJ_ROOT := $(shell pwd)

## Project variables
ORG_NAME=$(shell .project/config_var.sh project_org)
PROJ_NAME=$(shell .project/config_var.sh project_name)
REPO_NAME=${ORG_NAME}/${PROJ_NAME}
PROJ_PACKAGE := ${REPO_NAME}

## Common variables
HOSTNAME := $(shell echo $$HOSTNAME)
UNAME := $(shell uname)
GITHUB_HOST := github.com
GOLANG_HOST := golang.org
# GIT_DIRTY is empty if the project is not modified, otherwise it's current host name
GIT_DIRTY := $(shell git describe --dirty --always --tags --long | grep -q -e '-dirty' && echo -$$HOSTNAME)
GIT_HASH := $(shell git rev-parse --short HEAD)
# number of commits
COMMITS_COUNT := $(shell git rev-list --count ${GIT_HASH})
#
PROD_VERSION := $(shell cat .VERSION)
GIT_VERSION := $(shell printf %s-%d%s ${PROD_VERSION} ${COMMITS_COUNT} ${GIT_DIRTY})
COVPATH=.coverage

# List of all .go files in the project, excluding vendor and .tools
GOFILES_NOVENDOR = $(shell find . -type f -name '*.go' -not -path "./vendor/*" -not -path "./.tools/*" -not -path "./.gopath/*")

export PROJ_DIR=$(PROJ_ROOT)
export PROJ_BIN=$(PROJ_ROOT)/bin
export VENDOR_SRC=$(PROJ_ROOT)/vendor

# if PROJ_GOPATH is defined,
# then GOPATH and GOPROJ_DIR are expected to be set, and symbolic link to the project must be created;
# otherwise create necessary environment
ifndef PROJ_GOPATH
export PROJ_GOPATH_DIR=.gopath
export PROJ_GOPATH := ${PROJ_DIR}/${PROJ_GOPATH_DIR}
export GOPATH := ${PROJ_GOPATH}
export GOPROJ_DIR := $(shell go env GOPROJ_DIR)
export PATH := ${PATH}:${GOPATH}/bin:${GOPROJ_DIR}/bin
endif

# tools path
export TOOLS_PATH := ${PROJ_DIR}/.tools
export TOOLS_SRC := ${TOOLS_PATH}/src
export TOOLS_BIN := ${TOOLS_PATH}/bin
export PATH := ${PATH}:${TOOLS_BIN}

PROJ_REPO_TARGET := "${PROJ_GOPATH_DIR}/src/${REPO_NAME}"

# test path
TEST_GOPATH := "${PROJ_GOPATH}"
TEST_DIR := "${PROJ_REPO_TARGET}"

# List of all .go files in the project, exluding vendor and tools
PROJ_GOFILES = $(shell find . -type f -name '*.go' -not -path "./vendor/*" -not -path "./.gopath/*" -not -path "./.tools/*")

COVERAGE_EXCLUSIONS="/rt\.go|/bindata\.go"

# flags
INTEGRATION_TAG="integration"
TEST_RACEFLAG ?=
TEST_GORACEOPTIONS ?=

# flag to enable golang race detector. Usage: make $(test_target) RACE=true. For example, make test RACE=true
RACE ?=
ifeq ($(RACE),true)
	TEST_GORACEOPTIONS = "log_path=${PROJ_DIR}/${COVPATH}/race/report"
	TEST_RACEFLAG = -race
endif

# SSH clones over the VPN get killed by some kind of DOS protection run amook
# set clone_delay to add a delay between each git clone/fetch to work around that
# e.g. CLONE_DELAY=1 make all
# the default is no delayWorking on $(PROJ_PACKAGE) in
CLONE_DELAY ?= 0

# this prints out the git log between the checked out version and origin/master for all the git repos in the supplied tree
#
# the find cmd finds all the git repos by looking for .git diretories
# the [[ $$(git log) ... ]] at the start the script checks to see if there are any log entries, it only does the rest
# of the command if there are some
# it runs git log in the relevant directory to show the log entries betweeen HEAD and origin/master
define show_dep_updates
	find $(1) -name .git -exec sh -c 'cd {}/.. && [ $$(git log --oneline HEAD...origin/master | wc -l) -gt 0 ] && echo "\n" && pwd && git --no-pager log --pretty=oneline --abbrev=0 --graph HEAD...origin/master' \;
endef

# gitclone is a function that will do a clone, or a fetch / checkout [if we'd previous done a clone]
# usage, $(call gitclone,github.com,ekspand/foo,/some/directory,some_sha)
# it builds a repo url from the first 2 params, the 3rd param is the directory to place the repo
# and the final param is the commit to checkout [a sha or branch or tag]
define gitclone
	@echo "Checking/Updating dependency https://$(1)/$(2)"
	@if [ -d $(3) ]; then cd $(3) && git fetch origin; fi			# update from remote if we've already cloned it
	@if [ ! -d $(3) ]; then git clone -q -n https://$(1)/$(2) $(3); fi  # clone a new copy
	@cd $(3) && git checkout -q $(4)								# checkout out specific commit
	@sleep ${CLONE_DELAY}
endef

## Common targets/functions for golang projects
# 	They assume that
#	a) GOPATH has been set with an export GOPATH somewhere
#	b) the Makefile variable PROJ_PACKAGE has been set to the name of the go pacakge to operate on
#

# go_test_cover will run go test on a package tree, with code coverage turned on, it writes coverage results
# to ./${COVPATH}
# the 5 params are
#		1) the working dir to run the tests in
#		2) the GOPATH to run the tests with
#		3) flag to enable race detector
#		4) options to race detector such as log_path for storing the results of the race detector
#		5) the name of the PROJ_DIR package to test
#		6) the list of source exclusions to apply to the generated code coverage result calculation
#
# it assumes you've built the cov-report tool into ${TOOLS_BIN}
#
define go_test_cover
	echo  "Testing in $(1)"
	mkdir -p ${COVPATH}/race
	exitCode=0 \
	&& cd ${1} && go list $(5)/... | ( while read -r pkg; do \
		result=`GOPATH=$(2) GORACE=$(4) go test $$pkg -coverpkg=$(5)/... -covermode=count $(3) \
			-coverprofile=${COVPATH}/cc_$$(echo $$pkg | tr "/" "_").out \
			2>&1 | grep --invert-match "warning: no packages"` \
			&& test_result=`echo "$$result" | tail -1` \
			&& echo "$$test_result" \
			&& if echo $$test_result | grep ^FAIL ; then \
				exitCode=1 && echo "Test for $$pkg failed. Result: $$result, exit code: $$exitCode" \
			; fi \
		; done \
		&& echo "Completed with status code $$exitCode" \
		&& if [ $$exitCode -ne "0" ] ; then echo "Test failed, exit code: $$exitCode" && exit $$exitCode ; fi )
	cov-report -ex $(6) -cc ${COVPATH}/combined.out ${COVPATH}/cc*.out
	cp ${COVPATH}/combined.out ${PROJ_DIR}/coverage.out
endef

# same as go_test_cover except it also generates results in the junit format
# assuming ${TOOLS_BIN} contains go-junit-report & cov-report
define go_test_cover_junit
	echo  "Testing in $(1)"
	mkdir -p ${COVPATH}/race
	set -o pipefail; failure=0; while read -r pkg; do \
		cd $(1) && GOPATH=$(2) GORACE=$(4) go test -v $$pkg -coverpkg=$(5)/... -covermode=count $(3) \
			-coverprofile=${COVPATH}/cc_$$(echo $$pkg | tr "/" "_").out \
			>> ${COVPATH}/citest_$$(echo $(5) | tr "/" "_").log \
			|| failure=1; \
    done <<< "$$(cd $(1) && go list $(5)/...)" && \
    cat ${COVPATH}/citest_$$(echo $(5) | tr "/" "_").log | go-junit-report >> ${COVPATH}/citest_$$(echo $(5) | tr "/" "_").xml && \
    exit $$failure
endef

# list the make targets
# http://stackoverflow.com/questions/4219255/how-do-you-get-the-list-of-targets-in-a-makefile/15058900#15058900
no_targets__:
list:
	sh -c "$(MAKE) -p no_targets__ | awk -F':' '/^[a-zA-Z0-9][^\$$#\/\\t=]*:([^=]|$$)/ {split(\$$1,A,/ /);for(i in A)print A[i]}' | grep -v '__\$$' | sort"

#
# print environment variables
#
vars:
	[ -d "${PROJ_REPO_TARGET}" ] && echo "Repo target exists: ${PROJ_REPO_TARGET}" || echo "Symbolic link does not exist: ${PROJ_REPO_TARGET}"
	echo "PROJ_DIR=$(PROJ_DIR)"
	echo "GOROOT=$(GOROOT)"
	echo "GOPATH=$(GOPATH)"
	echo "PROJ_PACKAGE=$(PROJ_PACKAGE)"
	echo "PROJ_GOPATH=$(PROJ_GOPATH)"
	echo "TOOLS_PATH=$(TOOLS_PATH)"
	echo "TEST_GOPATH=$(TEST_GOPATH)"
	echo "TEST_DIR=$(TEST_DIR)"
	echo "GIT_VERSION=$(GIT_VERSION)"

#
# clean produced files
#
clean:
	go clean
	rm -rf \
		${COVPATH} \
		${PROJ_BIN}

#
# clean and purge tools and vendor
#
purge: clean
	rm -rf \
		${TOOLS_PATH} \
		${VENDOR_SRC}

#
# create a symbolic link to project's PROJ_GOPATH,
# if it's not cloned in GOPATH.
#
gopath:
	@[ ! -d $(PROJ_REPO_TARGET) ] && \
		rm -f "${PROJ_REPO_TARGET}" && \
		mkdir -p "${PROJ_GOPATH_DIR}/src/${ORG_NAME}" && \
		ln -s ../../../.. "${PROJ_REPO_TARGET}" && \
		echo "Created symbolic link: ${PROJ_REPO_TARGET}" || \
	echo "Repo target exists: ${PROJ_REPO_TARGET}"

#
# show updates in Tools and vendor folder.
#
showupdates:
	@$(call show_dep_updates,${TOOLS_SRC})
	@$(call show_dep_updates,${VENDOR_SRC})

#
# list packages
#
lspkg:
	cd ${TEST_DIR} && go list ./...

#
# print out GO environment
#
env:
	GOPATH=${GOPATH} go env

#
# print out GO test environment
#
testenv:
	GOPATH=${TEST_GOPATH} go env

#
# GO test with bench
#
bench:
	GOPATH=${TEST_GOPATH} go test  ${TEST_RACEFLAG} -bench . ${PROJ_PACKAGE}/...

generate:
	PATH=${TOOLS_BIN}:${PATH} go generate ./...

fmt:
	echo "Running Fmt"
	gofmt -s -l -w ${GOFILES_NOVENDOR}

vet: build
	echo "Running vet"
	cd ${TEST_DIR} && go vet ./...

lint:
	echo "Running lint"
	cd ${TEST_DIR} && GOPATH=${TEST_GOPATH}  go list ./... | grep -v /vendor/ | xargs -L1 golint -set_exit_status

test: fmt vet lint
	echo "Running test"
	cd ${TEST_DIR} && go test ${TEST_RACEFLAG} ./...

testshort:
	echo "Running testshort"
	cd ${TEST_DIR} && go test ${TEST_RACEFLAG} ./... --test.short

covtest: fmt vet lint
	echo "Running covtest"
	$(call go_test_cover,${TEST_DIR},${TEST_GOPATH},${TEST_RACEFLAG},${TEST_GORACEOPTIONS},.,${COVERAGE_EXCLUSIONS})

# Runs integration tests as well
testint: fmt vet lint
	echo "Running testint"
	GOPATH=${TEST_GOPATH} go test ${TEST_RACEFLAG} -tags=${INTEGRATION_TAG} ${PROJ_PACKAGE}/...

# shows the coverages results assuming they were already generated by a call to go_test_cover
coverage:
	echo "Running coverage"
	GOPATH=${TEST_GOPATH} go tool cover -html=${COVPATH}/combined.out

# generates a HTML based code coverage report, and writes it to a file in the results directory
# assumes you've run go_test_cover (or go_test_cover_junit)
cicoverage:
	echo "Running cicoverage"
	mkdir -p ${COVPATH}/cover
	GOPATH=${TEST_GOPATH} go tool cover -html=${COVPATH}/combined.out -o ${COVPATH}/cover/coverage.html

# as Jenkins runs citestint as well which will run all unit tests + integration tests with code coverage
# this unitest step can skip coverage reporting which speeds it up massively
citest: vet lint
	echo "Running citest"
	$(call go_test_cover_junit,${TEST_DIR},${GOPATH},${TEST_RACEFLAG},${TEST_GORACEOPTIONS},.,${COVERAGE_EXCLUSIONS})
	cov-report -fmt xml -o ${COVPATH}/coverage.xml -ex ${COVERAGE_EXCLUSIONS} -cc ${COVPATH}/combined.out ${COVPATH}/cc*.out
	cov-report -fmt ds -o ${COVPATH}/summary.xml -ex ${COVERAGE_EXCLUSIONS} ${COVPATH}/cc*.out

coveralls: covtest
	echo "Running coveralls"
	goveralls -v -coverprofile=coverage.out -service=travis-ci -package ./...

help:
	echo "make vars - print make variables"
	echo "make upgrade-project.mk - upgrade project.mk files"
	echo "make env - pring GO environment"
	echo "make testenv - pring GO test environment"
	echo "make clean - clean produced files"
	echo "make purge - clean and purge .tools and vendor folders"
	echo "make gopath - create a symbolic link to project's PROJ_GOPATH, if it's not cloned in GOPATH."
	echo "make showupdates - show updates in .tools and vendor folders"
	echo "make lspkg - list GO packeges in the current project"
	echo "make generate - generate GO files"
	echo "make bench - GO test with bench"
	echo "make fmt - run go fmt on project files"
	echo "make vet - run go vet on project files"
	echo "make lint - run go lint on project files"
	echo "make test - run test"
	echo "make testshort - run test with -short flag"
	echo "make covtest - run test with coverage report"
	echo "make coverage - open coverage report"
	echo "make coveralls - publish coverage to coveralls"
	echo "make devtools - install dev tools"

getdevtools:
	$(call gitclone,${GITHUB_HOST},golang/tools,           ${GOPATH}/src/golang.org/x/tools,                  release-branch.go1.10)
	$(call gitclone,${GITHUB_HOST},derekparker/delve,      ${GOPATH}/src/github.com/derekparker/delve,        master)
	$(call gitclone,${GITHUB_HOST},uudashr/gopkgs,         ${GOPATH}/src/github.com/uudashr/gopkgs,           master)
	$(call gitclone,${GITHUB_HOST},nsf/gocode,             ${GOPATH}/src/github.com/nsf/gocode,               master)
	$(call gitclone,${GITHUB_HOST},rogpeppe/godef,         ${GOPATH}/src/github.com/rogpeppe/godef,           master)
	$(call gitclone,${GITHUB_HOST},acroca/go-symbols,      ${GOPATH}/src/github.com/acroca/go-symbols,        master)
	$(call gitclone,${GITHUB_HOST},ramya-rao-a/go-outline, ${GOPATH}/src/github.com/ramya-rao-a/go-outline,   master)
	$(call gitclone,${GITHUB_HOST},ddollar/foreman,        ${GOPATH}/src/github.com/ddollar/foreman,          master)
	$(call gitclone,${GITHUB_HOST},sqs/goreturns,          ${GOPATH}/src/github.com/sqs/goreturns,            master)
	$(call gitclone,${GITHUB_HOST},karrick/godirwalk,      ${GOPATH}/src/github.com/karrick/godirwalk,        master)
	$(call gitclone,${GITHUB_HOST},pkg/errors,             ${GOPATH}/src/github.com/pkg/errors,               master)

devtools: getdevtools
	go install golang.org/x/tools/go/buildutil
	go install golang.org/x/tools/cmd/fiximports
	go install golang.org/x/tools/cmd/goimports
	go install github.com/derekparker/delve/cmd/dlv
	go install github.com/uudashr/gopkgs/cmd/gopkgs
	go install github.com/nsf/gocode
	go install github.com/rogpeppe/godef
	go install github.com/acroca/go-symbols
	go install github.com/ramya-rao-a/go-outline
	go install github.com/sqs/goreturns

upgrade-project.mk:
	wget -O vscode.sh https://raw.githubusercontent.com/go-phorce/go-makefile/master/vscode.sh
	wget -O .project/go-project.mk https://raw.githubusercontent.com/go-phorce/go-makefile/master/.project/go-project.mk