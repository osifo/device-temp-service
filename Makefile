.PHONY: build test clean docker unittest lint

ARCH=$(shell uname -m)

MICROSERVICES=cmd/device-temp-service
.PHONY: $(MICROSERVICES)

VERSION=$(shell cat ./VERSION 2>/dev/null || echo 0.0.0)
SDKVERSION=$(VERSION)
DOCKER_TAG=$(VERSION)-dev

GOFLAGS=-ldflags "-X github.com/osifo/device-temp-service.Version=$(VERSION)" -trimpath -mod=readonly

GOTESTFLAGS?=-race

GIT_SHA=$(shell git rev-parse HEAD)

build: $(MICROSERVICES)

tidy:
	go mod tidy

# CGO is enabled by default and cause docker builds to fail due to no gcc,
# but is required for test with -race, so must disable it for the builds only
cmd/device-temp-service:
	CGO_ENABLED=0  go build $(GOFLAGS) -o $@ ./cmd

docker:
	docker build \
		-f cmd/device-temp-service/Dockerfile \
		--label "git_sha=$(GIT_SHA)" \
		-t osifo/device-temp-service:$(GIT_SHA) \
		-t osifo/device-temp-service:$(DOCKER_TAG) \
		.

unittest:
	go test $(GOTESTFLAGS) -coverprofile=coverage.out ./...

lint:
	@which golangci-lint >/dev/null || echo "WARNING: go linter not installed. To install, run make install-lint"
	@if [ "z${ARCH}" = "zx86_64" ] && which golangci-lint >/dev/null ; then golangci-lint run --config .golangci.yml ; else echo "WARNING: Linting skipped (not on x86_64 or linter not installed)"; fi

install-lint:
	sudo curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.54.2

test: unittest lint
	GO111MODULE=on go vet ./...
	gofmt -l $$(find . -type f -name '*.go'| grep -v "/vendor/")
	[ "`gofmt -l $$(find . -type f -name '*.go'| grep -v "/vendor/")`" = "" ]
	./bin/test-attribution-txt.sh

clean:
	rm -f $(MICROSERVICES)

vendor:
	go mod vendor
