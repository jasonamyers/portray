# Makefile to help with building portray
#

help:
	@echo ""
	@echo ""
	@echo "  build       	builds portray for your current environment"
	@echo "  build_multi    builds portray for multiple environments"
	@echo ""

build:
	go get github.com/aws/aws-sdk-go
	@echo "Building portray for your current environment"
	go build main.go -o portray

build_multi:
	go get github.com/aws/aws-sdk-go
	@echo "Building portray for Linux amd64"
	GOOS=linux GOARCH=amd64 go build -o portray-linux-amd64 main.go
	@echo "Building portray for Mac OS X amd64"
	GOOS=darwin GOARCH=amd64 go build -o portray-mac-amd64 main.go
	@echo "Building portray for Windows amd64"
	GOOS=windows GOARCH=amd64 go build -o portray-windows-amd64 main.go
