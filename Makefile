SHELL = /bin/bash

# Project
VERSION ?= 1.0.0

ISRELEASED := $(shell git show-ref v$(VERSION) 2>&1 > /dev/null && echo "true")

# Utilities
# Default environment variables.
# Any variables already set will override the values in this file(s).
DOTENV := godotenv -f $(HOME)/.env,.env

# Go
GOFILES := $(shell find . -name '*.go' 2> /dev/null | grep -v vendor)

.PHONY: browse bumpmajor bumpminor bumppatch cattest clean tools _isreleased release _release test _test _test_setup

#
# End user targets
#
clean:
	rm -rf .cache $(BINARY) dist reports tmp vendor

test:
	@$(DOTENV) make _test

release: _isreleased
	@$(DOTENV) make _release

tools:
	@make $(GOINSTALL)

bumpmajor:
	git fetch --tags
	versionbump --checktags major Makefile

bumpminor:
	git fetch --tags
	versionbump --checktags minor Makefile

bumppatch:
	git fetch --tags
	versionbump --checktags patch Makefile

browse:
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
GOINSTALL := github.com/jstemmer/go-junit-report github.com/golangci/golangci-lint/cmd/golangci-lint \
		  github.com/ains/go-test-html github.com/fzipp/gocyclo/cmd/gocyclo github.com/joho/godotenv/cmd/godotenv \
		  github.com/crosseyed/versionbump/cmd/versionbump
.PHONY: $(GOINSTALL)
$(GOINSTALL):
	go install $@@latest

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

_test_setup:
	@mkdir -p tmp
	@mkdir -p reports/html

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
_isreleased:
ifeq ($(ISRELEASED),true)
	@echo "Version $(VERSION) has been released."
	@echo "Please bump with 'make bump(minor|patch|major)' depending on breaking changes."
	@exit 1
endif