GOCMD?=go
GOBUILD=$(GOCMD) build
GOINSTALL=$(GOCMD) install
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOTOOL=$(GOCMD) tool
BINARY_NAME=soundsprout-server

.PHONY: all
all: test build

.PHONY: build
build:
	$(GOBUILD) -o $(BINARY_NAME) -v ./cmd

.PHONY: test
test:
	$(GOTEST) -v ./... -coverprofile=./../../cover.out -covermode=atomic -coverpkg=./...
	$(GOTOOL) cover -html=./../../cover.out -o ./../../cover.html

.PHONY: clean
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

.PHONY: install
install:
	$(GOCLEAN)
	GOBIN=/usr/local/bin/ $(GOINSTALL) ./cmd

.PHONY: run
run:
	./$(BINARY_NAME)
