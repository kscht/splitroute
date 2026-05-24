BINARY := splitroute
GO_VERSION := 1.22.4
GO_INSTALL_DIR := /usr/local

.PHONY: build install

build:
	go build -mod=vendor -o $(BINARY) .

install:
	bash install.sh
