APP_NAME := WayTune
BIN_NAME := waytune

VERSION ?= $(shell printf "r%s.%s" $(shell git rev-list --count HEAD) $(shell git rev-parse --short=7 HEAD) || echo "Git")

PREFIX ?= /usr/local

BUILD_FLAGS := -w -s -X '$(APP_NAME)/cmd.Version=$(VERSION)'

all: build

deps: .dependency-stamp

.dependency-stamp:
	go get -v
	@touch .dependency-stamp

build: $(BIN_NAME)

$(BIN_NAME): deps
	go build -trimpath -ldflags "$(BUILD_FLAGS)" -o "$(BIN_NAME)"

install:
	@if [ ! -f "$(BIN_NAME)" ]; then \
		echo "Error: $(BIN_NAME) not found. Run 'make' first."; \
		exit 1; \
	fi
	@echo "Installing to '$(PREFIX)'..."
	install -Dm755 $(BIN_NAME) "$(PREFIX)/bin/$(BIN_NAME)"
	install -Dm644 README.md "$(PREFIX)/share/doc/$(APP_NAME)/README.md"
	install -Dm644 LICENSE "$(PREFIX)/share/licenses/$(APP_NAME)/LICENSE"

clean:
	rm -f $(BIN_NAME) .dependency-stamp .build-stamp
