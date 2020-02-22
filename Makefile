SHELL = /bin/bash

# Project
VERSION ?= 1.0.0
COMMIT := $(shell test -d .git && git rev-parse --short HEAD)
BUILD_INFO := $(COMMIT)-$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

RELEASE_DESCRIPTION := Standard release

ISRELEASED := $(shell git show-ref v$(VERSION) 2>&1 > /dev/null && echo "true")

# Utilities
# Default environment variables.
# Any variables already set will override the values in this file(s).
DOTENV := godotenv -f $(HOME)/.env,.env

# Variables
ROOT = $(shell pwd)

# GITLAB variables
ifdef GITLAB_FEATURES
export USEGITLAB := true
export USECACHE := true
export PATH := $(CI_PROJECT_DIR)/.cache/go/bin:$(PATH)
export GOPATH := $(CI_PROJECT_DIR)/.cache/go
export GOCACHE := $(CI_PROJECT_DIR)/.cache/gocache
export GITSSH := $(shell echo git@$$(sed 's|^https://||; s|/|:|' <<< $(CI_PROJECT_URL)).git)
endif

# Go
GOMODOPTS = GO111MODULE=on
GOGETOPTS = GO111MODULE=off
GOFILES := $(shell find . -name '*.go' 2> /dev/null | grep -v vendor)

.PHONY: build _build _build_xcompile browsetest cattest clean deps _deps depsdev deploy lint nonenforcedlint enforcedlint \
        _go.mod _go.mod_err _isreleased lint release _release _release_gitlab test _test _test_setup

#
# End user targets
#
build: deps
	@$(DOTENV) make _build

install: dist/$(BINARY)-$(VERSION)
	cp dist/$(BINARY)-$(VERSION) $(GOPATH)/bin/$(BINARY)

clean:
	rm -rf .cache $(BINARY) dist reports tmp vendor

test: deps
	@$(DOTENV) make _test

lint: enforcedlint nonenforcedlint

lintenforced:
	golangci-lint run --no-config --disable-all --enable gosimple,golint,varcheck,unused

lintunenforced:
	golangci-lint run

deploy: build
	@echo TODO

release: _isreleased
	@$(DOTENV) make _release

deps: go.mod
ifeq ($(USEGITLAB),true)
	@mkdir -p $(ROOT)/.cache/{go,gomod}
endif
	@$(DOTENV) make _deps

keytest:
	@mkdir -p ~/.ssh
	@echo -e 'Host *\n\tStrictHostKeyChecking no\n\n' > ~/.ssh/config
	@chmod -R og-rwx ${RELEASE_KEY} ~/.ssh
	@ssh-add ${RELEASE_KEY}
	### Test SSH push
	ssh -T git@cd.splunkdev.com

depsdev:
ifeq ($(USEGITLAB),true)
	@mkdir -p $(ROOT)/.cache/{go,gomod}
endif
	@make $(GOGETS)

bumpmajor:
	git fetch --tags
	versionbump --checktags major Makefile

bumpminor:
	git fetch --tags
	versionbump --checktags minor Makefile

bumppatch:
	git fetch --tags
	versionbump --checktags patch Makefile

browsetest:
	@make $(REPORTS)

cattest:
	### Unit Tests
	@cat reports/test.txt
	### Code Coverage
	@cat reports/coverage.txt
	### Cyclomatix Complexity Report
	@cat reports/cyclocomplexity.txt

#
# Helper targets
#
_build:
ifeq ($(XCOMPILE),true)
	@make _build_xcompile
else
	@make _build_native
endif

_build_native: $(BINARY)

_build_xcompile: dist/$(BINARY)-$(VERSION)-linux-amd64 dist/$(BINARY)-$(VERSION)-darwin-amd64

_deps:
	$(GOMODOPTS) go mod tidy
	$(GOMODOPTS) go mod vendor

GOGETS := github.com/jstemmer/go-junit-report github.com/golangci/golangci-lint/cmd/golangci-lint \
		  github.com/ains/go-test-html github.com/fzipp/gocyclo github.com/joho/godotenv/cmd/godotenv \
		  github.com/crosseyed/versionbump/cmd/versionbump github.com/stretchr/testify
.PHONY: $(GOGETS)
$(GOGETS):
	go get -u $@

_test:
	@make _test_setup
	### Unit Tests
	@(go test -timeout 5s -covermode atomic -coverprofile=./reports/coverage.out -v ./...; echo $$? > reports/exitcode.txt) 2>&1 | tee reports/test.txt
	@cat ./reports/test.txt | go-junit-report > reports/junit.xml
	### Code Coverage
	@go tool cover -func=./reports/coverage.out | tee ./reports/coverage.txt
	@go tool cover -html=reports/coverage.out -o reports/html/coverage.html
	### Cyclomatix Complexity Report
	@gocyclo -avg $(GOFILES) | grep -v _test.go | tee reports/cyclocomplexity.txt
	@exit $$(cat reports/exitcode.txt)

_test_setup:
	@mkdir -p tmp
	@mkdir -p reports/html

_release:
	@echo "### Releasing $(VERSION)"

REPORTS = reports/html/coverage.html
.PHONY: $(REPORTS)
$(REPORTS):
	@test -f $@ && open $@

# Check versionbump
_isreleased:
ifeq ($(ISRELEASED),true)
	@echo "Version $(VERSION) has been released."
	@echo "Please bump with 'make bump(minor|patch|major)' depending on breaking changes."
	@exit 1
endif

#
# File targets
#
$(BINARY): dist/$(BINARY)-$(VERSION)
	install -m 755 dist/$(BINARY)-$(VERSION) $(BINARY)

dist/$(BINARY)-$(VERSION): $(GOFILES)
	@mkdir -p dist
	CGO_ENABLED=0 go build \
	-a -installsuffix cgo \
	-ldflags "-X cd.splunkdev.com/cloudworks/gocwb/internal.Version=$(VERSION)-$(BUILD_INFO)" \
	-o $@

dist/$(BINARY)-$(VERSION)-linux-amd64: $(GOFILES)
	@mkdir -p dist
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build \
	-a -installsuffix cgo \
	-ldflags "-X cd.splunkdev.com/cloudworks/gocwb/internal.Version=$(VERSION)-$(BUILD_INFO)" \
	-o $@

dist/$(BINARY)-$(VERSION)-darwin-amd64: $(GOFILES)
	@mkdir -p dist
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build \
	-a -installsuffix cgo \
	-ldflags "-X cd.splunkdev.com/cloudworks/gocwb/internal.Version=$(VERSION)-$(BUILD_INFO)" \
	-o $@

go.mod:
	@$(DOTENV) make _go.mod

_go.mod:
ifndef GOSERVER
	@make _go.mod_err
else ifndef GOGROUP
	@make _go.mod_err
endif
	go mod init $(GOSERVER)/$(GOGROUP)/$(BINARY)
	@make deps

_go.mod_err:
	@echo 'Please run "go mod init server.com/group/project"'
	@echo 'Alternatively set "GOSERVER=$$YOURSERVER" and "GOGROUP=$$YOURGROUP" in ~/.env or $(ROOT)/.env file'
	@exit 1
