EXECUTABLE=gen-wallet
WINDOWS=$(EXECUTABLE)_windows_amd64.exe
LINUX=$(EXECUTABLE)_linux_amd64
DARWIN=$(EXECUTABLE)_darwin_amd64
DARWINARM=$(EXECUTABLE)_darwin_arm64

.PHONY: all clean

all: windows linux darwin darwinarm ## Build binaries
	@echo "Building Finished"

windows: $(WINDOWS) ## Build for Windows

linux: $(LINUX) ## Build for Linux

darwin: $(DARWIN) ## Build for Darwin (macOS)

darwinarm: $(DARWINARM) ## Build for Darwin ARM (macOS)

$(WINDOWS):
	env GOOS=windows GOARCH=amd64 go build -v -o $(WINDOWS) -ldflags="-s -w" *.go

$(LINUX):
	env GOOS=linux GOARCH=amd64 go build -v -o $(LINUX) -ldflags="-s -w" *.go

$(DARWIN):
	env GOOS=darwin GOARCH=amd64 go build -v -o $(DARWIN) -ldflags="-s -w" *.go

$(DARWINARM):
	env GOOS=darwin GOARCH=arm64 go build -v -o $(DARWINARM) -ldflags="-s -w" *.go

clean: ## Remove previous build
	rm -f $(WINDOWS) $(LINUX) $(DARWIN) $(DARWINARM)

help: ## Display available commands
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
