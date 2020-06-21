include .project/gomod-project.mk
export GO111MODULE=on
BUILD_FLAGS=-mod=vendor

CERTS_PREFIX=test_${PROJ_NAME}_

.PHONY: *

.SILENT:

default: help

all: clean tools generate hsmconfig gen_test_certs covtest

#
# clean produced files
#
clean:
	go clean
	rm -rf \
		${COVPATH} \
		${PROJ_BIN}

tools:
	go install golang.org/x/tools/cmd/stringer
	go install github.com/go-phorce/cov-report/cmd/cov-report
	go install golang.org/x/lint/golint
	go install github.com/mattn/goveralls
	go install github.com/cloudflare/cfssl/cmd/cfssl
	go install github.com/cloudflare/cfssl/cmd/cfssljson

version:
	gofmt -r '"GIT_VERSION" -> "$(GIT_VERSION)"' version/current.template > version/current.go

build:
	echo "*** running build"
	go build ${BUILD_FLAGS} -o ${PROJ_ROOT}/bin/dollypki ./cmd/dollypki

hsmconfig:
	echo "*** Running hsmconfig"
	mkdir -p ~/softhsm2
	.project/config-softhsm.sh --pin-file ~/softhsm2/pin_unittest.txt --generate-pin -s dolly_unittest -o ./etc/dev/softhsm_unittest.json --list-slots --list-object --delete

gen_test_certs:
	echo "*** Running gen_test_certs"
	.project/gen_test_certs.sh --ca-config $(PROJ_ROOT)/etc/dev/ca-config.dev.json --out-dir $(PROJ_ROOT) --prefix $(CERTS_PREFIX) --root-ca --ca --server --client --peers --admin
