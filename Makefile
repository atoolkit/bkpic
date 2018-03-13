ROOT=$(PWD)

.PHONY: build clean deps fmt test

build: fmt
	@go build

clean:
	@git clean -dxf

deps:
	@go get -u -v golang.org/x/tools/cmd/goimports
	@go get -t -v *

fmt:
	@goimports -w *.go
	@goimports -w */*.go

test: fmt
	@go test
