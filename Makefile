UNAME := $(shell uname)

ifeq ($(UNAME), Linux)
BUILDS := build-linux
INSTALL_SRC := builds/linux/secure-environment
endif

ifeq ($(UNAME), Darwin)
BUILDS := build-linux build-macos
INSTALL_SRC := builds/macos/secure-environment
endif


all: $(BUILDS)

build-linux:
	mkdir -p builds/linux
	docker run --rm -v `pwd`:/go/src/github.com/virtru/secure-environment -w /go/src/github.com/virtru/secure-environment golang:1.7 go build -o builds/linux/secure-environment

build-macos:
	mkdir -p builds/macos
	go build -o builds/macos/secure-environment
