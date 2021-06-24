include .project/gomod-project.mk
export GO111MODULE=on
BUILD_FLAGS=

CERTS_PREFIX=test_${PROJ_NAME}_

.PHONY: *

.SILENT:

default: help

all: clean tools generate start-local-kms hsmconfig gen_test_certs covtest

#
# clean produced files
#
clean:
	go clean ./...
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
	mkdir -p ~/softhsm2 /tmp/dolly
	.project/config-softhsm.sh \
		--pin-file ~/softhsm2/dolly_pin_unittest.txt \
		--generate-pin \
		-s dolly_unittest \
		-o /tmp/dolly/softhsm_unittest.json \
		--list-slots --list-object --delete

gen_test_certs:
	echo "*** Running gen_test_certs"
	mkdir -p /tmp/dolly/certs
	.project/gen_test_certs.sh \
		--ca-config $(PROJ_ROOT)/etc/dev/ca-config.dev.json \
		--csr-dir $(PROJ_ROOT)/etc/dev/certs/csr \
		--out-dir /tmp/dolly/certs \
		--prefix $(CERTS_PREFIX) \
		--root --ca1 --ca2 --bundle \
		--server --client --peers --admin

coveralls-github:
	echo "Running coveralls"
	goveralls -v -coverprofile=coverage.out -service=github -package ./...

start-local-kms:
	# Container state will be true (it's already running), false (exists but stopped), or missing (does not exist).
	# Annoyingly, when there is no such container and Docker returns an error, it also writes a blank line to stdout.
	# Hence the sed to trim whitespace.
	LKMS_CONTAINER_STATE=$$(echo $$(docker inspect -f "{{.State.Running}}" dolly-unittest-local-kms 2>/dev/null || echo "missing") | sed -e 's/^[ \t]*//'); \
	if [ "$$LKMS_CONTAINER_STATE" = "missing" ]; then \
		docker pull nsmithuk/local-kms && \
		docker run --network host \
			-d -e 'PORT=4599' \
			-p 4599:4599 \
			--name dolly-unittest-local-kms \
			nsmithuk/local-kms && \
			sleep 1; \
	elif [ "$$LKMS_CONTAINER_STATE" = "false" ]; then docker start dolly-unittest-local-kms && sleep 1; fi;
