GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_NAME=soundsprout-server

.PHONY: all
all: test build

.PHONY: build
build:
	$(GOBUILD) -o $(BINARY_NAME) -v ./cmd

.PHONY: test
test:
	$(GOTEST) -v ./...

.PHONY: clean
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

.PHONY: install
install:
	$(GOCLEAN)
	$(GOBUILD) install ./cmd

.PHONY: run
run:
	./$(BINARY_NAME)
