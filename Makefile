#!/usr/bin/make -f

COMMANDS := novm novmm noguest
BINDIR   := bin

CGO_ENABLED ?= 0
GO ?= go

.PHONY: all build test vet fmt clean

all: build

build:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build -o $(BINDIR)/ ./cmd/novm ./cmd/novmm ./cmd/noguest

test:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) test -v ./...

vet:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) vet ./...

fmt:
	gofmt -l -w .

clean:
	rm -rf $(BINDIR) dist/ _obj debbuild/ rpmbuild/ *.deb *.rpm

