CONFIGURATION?=debug

.PHONY: build
build:
	swift build --arch arm64 -c $(CONFIGURATION)
	mkdir -p bin
	cp .build/arm64-apple-macosx/$(CONFIGURATION)/vmctl bin/vmctl
	codesign --sign - --entitlements Resources/vmctl.entitlements bin/vmctl

.PHONY: format
format:
	swift format -i --recursive Sources Package.swift
