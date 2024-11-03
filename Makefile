# Root Makefile

.PHONY: all
all: build

.PHONY: clean
clean:
	$(MAKE) -C ./system/soundsprout clean

.PHONY: build
build:
	$(MAKE) -C ./system/soundsprout build

.PHONY: test
test:
	$(MAKE) -C ./system/soundsprout test

.PHONY: install
install:
	$(MAKE) -C ./system/soundsprout install

.PHONY: run
run:
	$(MAKE) -C ./system/soundsprout run
