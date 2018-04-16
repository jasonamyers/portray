# Makefile to help with building portray
#
BUILD_VERSION:=$(shell git describe --tags)
GIT_COMMIT := $(shell git rev-parse HEAD)
BUILD_TIME := $(shell TZ=utc date)
GO_VERSION := $(shell go version | sed 's/go version //')
LDFLAGS=-ldflags "-X main.Version=${BUILD_VERSION} -X main.GitCommit=${GIT_COMMIT} -X \"main.BuildTime=${BUILD_TIME}\" -X \"main.GoVersion=${GO_VERSION}\""

# Colors
NOCOLOR=\033[0m
RED=\033[0;31m
GREEN=\033[0;32m

help:
	@echo ""
	@echo ""
	@echo "  build       	builds portray for your current environment"
	@echo "  build_multi    builds portray for multiple environments"
	@echo ""

clean:
	@go clean

build:
	go get
	@echo "Building portray for your current environment"
	go build ${LDFLAGS} && echo "${GREEN}Success!${NOCOLOR}" || echo "${RED}Build failed!${NOCOLOR}";

build_multi:
	go get
	@echo "Building portray for Linux amd64"
	GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o portray-linux-amd64 main.go
	@echo "Building portray for Mac OS X amd64"
	GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o portray-mac-amd64 main.go
	@echo "Building portray for Windows amd64"
	GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o portray-windows-amd64 main.go

debug_in_tmux_pane:
	@echo "Launching delve debugger in bottom-right tmux pane"
	@if pgrep dlv >/dev/null 2>&1; then killall dlv; fi
	@tmux send-keys -t bottom-right 'echo "Magic launching Delve debugger"' Enter
	@tmux send-keys -t bottom-right 'dlv exec portray -- ${PORTRAY_ARGS}' ENTER

exec_in_tmux_pane:
	@echo "Executing in bottom-right tmux pane"
	@if pgrep dlv >/dev/null 2>&1; then killall dlv; fi
	@tmux send-keys -t bottom-right './portray ${PORTRAY_ARGS}' ENTER

execloop:
	@echo "Starting file watcher"
	@fswatch --exclude='.*\.git' \
    --exclude='.*\.yaml' \
    --exclude='.*\.json' \
	--exclude='.*\.swp' \
	--exclude='.*\debug.*?' \
	--exclude='.*4913' \
	--exclude='Makefile' \
	--exclude='LICENSE' \
	--exclude='.*/portray/portray' \
	--recursive . | \
	xargs -n1 -I{} sh -c 'echo "Change detected: {}"; make clean; make build; if [ -f portray ]; then make exec_in_tmux_pane PORTRAY_ARGS="${PORTRAY_ARGS}"; fi'

debugloop:
	@echo "Starting file watcher"
	@fswatch --exclude='.*\.git' \
    --exclude='.*\.yaml' \
    --exclude='.*\.json' \
	--exclude='.*\.swp' \
	--exclude='.*\debug.*?' \
	--exclude='.*4913' \
	--exclude='Makefile' \
	--exclude='LICENSE' \
	--exclude='.*/portray/portray' \
	--recursive . | \
	xargs -n1 -I{} sh -c 'echo "Change detected: {}"; make clean; make build; if [ -f portray ]; then make debug_in_tmux_pane PORTRAY_ARGS="${PORTRAY_ARGS}"; fi'
