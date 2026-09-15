BINARY := splitroute

.PHONY: build install

build:
	go build -mod=vendor -o $(BINARY) .

install:
	bash install.sh
