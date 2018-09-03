# common.mk: this contains commonly used helpers for makefiles.
SHELL=/bin/bash
ROOT := $(shell pwd)

## Project variables
ORG_NAME=$(shell .project/var.sh project_org)
PROJ_NAME=$(shell .project/var.sh project_name)
REPO_NAME=${ORG_NAME}/${PROJ_NAME}
PROJ_PACKAGE := ${REPO_NAME}

## Common variables
HOSTNAME := $(shell echo $$HOSTNAME)
UNAME := $(shell uname)
GITHUB_HOST := github.com
GOLANG_HOST := golang.org
GIT_DIRTY := $(shell git describe --dirty --always --tags --long | grep -q -e '-dirty' && echo -$$HOSTNAME)
GIT_HASH := $(shell git rev-parse --short HEAD)
COMMITS_COUNT := $(shell git rev-list --count ${GIT_HASH})# number of commits in master
PROD_VERSION := $(shell cat .VERSION)
GIT_VERSION := $(shell printf %s-%d%s ${PROD_VERSION} ${COMMITS_COUNT} ${GIT_DIRTY})
COVPATH=.coverage

export PROJROOT=$(ROOT)

# if PROJ_GOPATH is defined,
# then GOPATH and GOROOT are expected to be set, and symbolic link to Stampy must be created;
# otherwise create necessary environment
ifndef PROJ_GOPATH
export PROJ_GOPATH_DIR=.gopath
export PROJ_GOPATH := ${ROOT}/${PROJ_GOPATH_DIR}
export GOPATH := ${PROJ_GOPATH}
export GOROOT := $(shell go env GOROOT)
export PATH := ${PATH}:${GOPATH}/bin:${GOROOT}/bin
endif

PROJ_REPO_TARGET := "${PROJ_GOPATH_DIR}/src/${REPO_NAME}"

# tools path
export TOOLS_PATH := ${PROJ_GOPATH}/src/${REPO_NAME}/${VENDOR_SRC}/.tools
export TOOLS_SRC := ${TOOLS_PATH}/src
export TOOLS_BIN := ${TOOLS_PATH}/bin
export PATH := ${PATH}:${TOOLS_BIN}

# test path
TEST_GOPATH := "${PROJ_GOPATH}"
TEST_DIR := "${PROJ_REPO_TARGET}"


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
	find $(1) -name .git -exec sh -c 'cd {}/.. && [[ $$(git log --oneline HEAD...origin/master | wc -l) -gt 0 ]] && echo "\n" && pwd && git log --pretty=oneline --abbrev=0 --graph HEAD...origin/master' \;
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

# list the make targets
# http://stackoverflow.com/questions/4219255/how-do-you-get-the-list-of-targets-in-a-makefile/15058900#15058900
no_targets__:
list:
	sh -c "$(MAKE) -p no_targets__ | awk -F':' '/^[a-zA-Z0-9][^\$$#\/\\t=]*:([^=]|$$)/ {split(\$$1,A,/ /);for(i in A)print A[i]}' | grep -v '__\$$' | sort"

# go_test_cover will run go test on a package tree, with code coverage turned on, it writes coverage results
# to ./${COVPATH}
# the 5 params are
#		1) the working dir to run the tests in
#		2) the GOPATH to run the tests with
#		3) flag to enable race detector
#		4) options to race detector such as log_path for storing the results of the race detector
#		5) the name of the root package to test
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
	cp ${COVPATH}/combined.out ${ROOT}/coverage.out
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
