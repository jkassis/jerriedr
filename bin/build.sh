#!/bin/bash

# GOOS - Target Operating System	GOARCH - Target Platform
# android	arm
# darwin	386
# darwin	amd64
# darwin	arm
# darwin	arm64
# dragonfly	amd64
# freebsd	386
# freebsd	amd64
# freebsd	arm
# linux	386
# linux	amd64
# linux	arm
# linux	arm64
# linux	ppc64
# linux	ppc64le
# linux	mips
# linux	mipsle
# linux	mips64
# linux	mips64le
# netbsd	386
# netbsd	amd64
# netbsd	arm
# openbsd	386
# openbsd	amd64
# openbsd	arm
# plan9	386
# plan9	amd64
# solaris	amd64
# windows	386
# windows	amd64

env go build -o dist/main_linux_arm64 cmd/*.go
