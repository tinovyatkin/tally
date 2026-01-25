.PHONY: build test lint lint-fix clean release publish-prepare publish-npm publish-pypi publish-gem publish

build:
	CGO_ENABLED=0 go build -ldflags "-s -w" -o tally

test:
	go test -race -count=1 -timeout=30s ./...

GOLANGCI_LINT_VERSION := v2.8.0
GORELEASER_VERSION := v2.13.3

lint: bin/golangci-lint-$(GOLANGCI_LINT_VERSION)
	bin/golangci-lint run

lint-fix: bin/golangci-lint-$(GOLANGCI_LINT_VERSION)
	bin/golangci-lint run --fix

bin/golangci-lint-$(GOLANGCI_LINT_VERSION):
	@rm -f bin/golangci-lint bin/golangci-lint-*
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/$(GOLANGCI_LINT_VERSION)/install.sh | sh -s -- -b bin/ $(GOLANGCI_LINT_VERSION)
	@touch $@

bin/goreleaser-$(GORELEASER_VERSION):
	@rm -f bin/goreleaser bin/goreleaser-*
	GOBIN=$(CURDIR)/bin go install github.com/goreleaser/goreleaser/v2@$(GORELEASER_VERSION)
	@touch $@

clean:
	rm -f tally
	rm -rf bin/ dist/

# Release and publish targets
# Prerequisites:
#   - NPM_API_KEY env var (or npm login)
#   - UV_PUBLISH_TOKEN env var for PyPI
#   - ~/.gem/credentials for RubyGems

release: bin/goreleaser-$(GORELEASER_VERSION)
	bin/goreleaser release --clean --snapshot

publish-prepare: release
	cd packaging && ruby pack.rb prepare

publish-npm: publish-prepare
	cd packaging && ruby pack.rb publish_npm

publish-pypi: publish-prepare
	cd packaging && ruby pack.rb publish_pypi

publish-gem: publish-prepare
	cd packaging && ruby pack.rb publish_gem

publish: publish-prepare
	cd packaging && ruby pack.rb publish
