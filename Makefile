APP_NAME := Ephemeral's Waybar Modules
REPO_NAME := EWM
BIN_NAME := ewmod

PREFIX ?= /usr/local/
VERSION ?= $(shell git describe --tags 2>/dev/null || echo "Git")

INSTALL_DIR ?= $(shell printf '$(PREFIX)/bin/' | sed 's:/\+:/:g')

BUILD_FLAGS := -w -s -X github.com/Nadim147c/$(REPO_NAME)/cmd.Version=$(VERSION)

all: build

deps: .dependency-stamp

.dependency-stamp:
	go get -v
	@touch .dependency-stamp

build: deps
	go build -ldflags "$(BUILD_FLAGS)" -o "$(BIN_NAME)"

install: build
	if [ -w "$(INSTALL_DIR)" ]; then \
		mkdir -pv "$(INSTALL_DIR)"; \
		cp -v "$(BIN_NAME)" "$(INSTALL_DIR)"; \
	else \
		sudo mkdir -pv "$(INSTALL_DIR)"; \
		sudo cp "$(BIN_NAME)" "$(INSTALL_DIR)"; \
	fi
