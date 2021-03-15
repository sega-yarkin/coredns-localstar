COMMIT := $(shell git describe --dirty --always)
LDFLAGS := "-s -w -X github.com/coredns/coredns/coremain.GitCommit=$(COMMIT)"

.PHONY: build
build:
	CGO_ENABLED=0 go build -v -ldflags $(LDFLAGS) -o coredns cmd/coredns.go


.PHONY: release
release:
	@rm -rf release && mkdir release
	GOOS=darwin  GOARCH=amd64 CGO_ENABLED=0 go build -ldflags $(LDFLAGS) -o release/coredns_darwin_amd64 cmd/coredns.go
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags $(LDFLAGS) -o release/coredns_windows_amd64.exe cmd/coredns.go
	GOOS=linux   GOARCH=amd64 CGO_ENABLED=0 go build -ldflags $(LDFLAGS) -o release/coredns_linux_amd64 cmd/coredns.go
	GOOS=linux   GOARCH=arm64 CGO_ENABLED=0 go build -ldflags $(LDFLAGS) -o release/coredns_linux_arm64 cmd/coredns.go
