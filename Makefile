# Root Makefile

.PHONY: all
all: build

.PHONY: clean
clean:
	$(MAKE) -C ./system/go-pyrfid-juke-support clean

.PHONY: build
build:
	$(MAKE) -C ./system/go-pyrfid-juke-support build

.PHONY: test
test:
	$(MAKE) -C ./system/go-pyrfid-juke-support test

.PHONY: install
install:
	$(MAKE) -C ./system/go-pyrfid-juke-support install

.PHONY: run
run:
	$(MAKE) -C ./system/go-pyrfid-juke-support run
