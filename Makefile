# Makefile for the Go project

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_NAME=rfid-jukebox

all: test build
build:
	$(GOBUILD) -o $(BINARY_NAME) -v ./...
test:
	$(GOTEST) ./... -coverprofile=./cover.out -covermode=atomic -coverpkg=./...
	$(GOCMD) tool cover -html=./cover.out -o ./cover.html
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
run:
	./$(BINARY_NAME)
