SHELL = /bin/bash

# Project
VERSION ?= 1.0.0

ISRELEASED := $(shell git show-ref v$(VERSION) 2>&1 > /dev/null && echo "true")

# Utilities
# Default environment variables.
# Any variables already set will override the values in this file(s).
DOTENV := $(shell $(test -f $(HOME)/.env && echo "godotenv -f $(HOME)/.env,.env" || echo "godotenv -f .env"))

# Go
GOPATH := $(shell go env GOPATH)
GOFILES := $(shell find . -name '*.go' 2> /dev/null | grep -v vendor)

#
# End user targets
#
.PHONY: clean
clean:
	rm -rf .cache $(BINARY) dist reports tmp vendor

.PHONY: test
test:
	@$(DOTENV) make _test

.PHONY: release
release: _isreleased
	@$(DOTENV) make _release

.PHONY: tools
tools:
	@$(MAKE) $(GOPATH)/bin/go-junit-report $(GOPATH)/bin/golangci-lint $(GOPATH)/bin/go-test-html $(GOPATH)/bin/gocyclo $(GOPATH)/bin/godotenv $(GOPATH)/bin/versionbump

.PHONY: forcetools
forcetools:
	@$(MAKE) --always-make tools

.PHONY: bumpmajor
bumpmajor:
	git fetch --tags
	versionbump --checktags major Makefile

.PHONY: bumpminor
bumpminor:
	git fetch --tags
	versionbump --checktags minor Makefile

.PHONY: bumppatch
bumppatch:
	git fetch --tags
	versionbump --checktags patch Makefile

.PHONY: browse
browse:
	@make $(REPORTS)

.PHONY: cattest
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
.PHONY: _test
_test: _test_setup
	### Unit Tests
	@(go test -race -timeout 5s -covermode atomic -coverprofile=./reports/coverage.out -v ./...; echo $$? > reports/exitcode.txt) 2>&1 | tee reports/test.txt
	@cat ./reports/test.txt | go-junit-report > reports/junit.xml
	### Code Coverage
	@go tool cover -func=./reports/coverage.out | tee ./reports/coverage.txt
	@go tool cover -html=reports/coverage.out -o reports/html/coverage.html
	### Cyclomatix Complexity Report
	@gocyclo -avg $(GOFILES) | grep -v _test.go | tee reports/cyclocomplexity.txt
	@exit $$(cat reports/exitcode.txt)

.PHONY: _test_setup
_test_setup:
	@mkdir -p tmp
	@mkdir -p reports/html

.PHONY: _release
_release:
	@echo "### Releasing $(VERSION)"
	git tag v$(VERSION)
	git push 

REPORTS = reports/html/coverage.html
.PHONY: $(REPORTS)
$(REPORTS):
	@test -f $@ && \
	which open 2> /dev/null && open $@ || \
	which xdg-open && xdg-open $@

# Check versionbump
.PHONY: _isreleased
_isreleased:
ifeq ($(ISRELEASED),true)
	@echo "Version $(VERSION) has been released."
	@echo "Please bump with 'make bump(minor|patch|major)' depending on breaking changes."
	@exit 1
endif

#
# File targets
#
$(GOPATH)/bin/go-junit-report:
	@go install github.com/jstemmer/go-junit-report@latest

$(GOPATH)/bin/golangci-lint:
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

$(GOPATH)/bin/go-test-html:
	@go install github.com/ains/go-test-html@latest

$(GOPATH)/bin/gocyclo:
	@go install github.com/fzipp/gocyclo/cmd/gocyclo@latest

$(GOPATH)/bin/godotenv:
	@go install github.com/joho/godotenv/cmd/godotenv@latest

$(GOPATH)/bin/versionbump:
	@go install github.com/crosseyed/versionbump/cmd/versionbump@latest