GO_PACKAGES ?= $(shell go list ./... | grep -v /vendor/)

TARGETOS ?= linux
TARGETARCH ?= amd64

VERSION ?= dev
VERSION_NUMBER ?= 0.0.0
ifneq ($(CI_COMMIT_TAG),)
	VERSION := $(CI_COMMIT_TAG:v%=%)
	VERSION_NUMBER := ${VERSION}
endif

# append commit-sha to dev version
BUILD_VERSION ?= $(VERSION)
ifeq ($(BUILD_VERSION),next)
	CI_COMMIT_SHA ?= $(shell git rev-parse HEAD)
	BUILD_VERSION := $(shell echo "next-$(shell echo ${CI_COMMIT_SHA} | head -c 8)")
endif

##@ General

all: help

.PHONY: version
version: ## Print the current version
	@echo ${BUILD_VERSION}

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: vendor
vendor: ## Update the vendor directory
	go mod tidy
	go mod vendor

.PHONY: dep-update
dep-update: ## Update the vendor directory
	go get -u ./...
	go mod tidy
	go mod vendor