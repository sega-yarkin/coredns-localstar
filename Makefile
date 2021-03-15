
.PHONY: build
build:
	go build -v -o coredns cmd/coredns.go


.PHONY: release
release:
	@rm -rf release && mkdir release
	GOOS=darwin  GOARCH=amd64 go build -o release/coredns_darwin_amd64 cmd/coredns.go
	GOOS=windows GOARCH=amd64 go build -o release/coredns_windows_amd64.exe cmd/coredns.go
	GOOS=linux   GOARCH=amd64 go build -o release/coredns_linux_amd64 cmd/coredns.go
	GOOS=linux   GOARCH=arm64 go build -o release/coredns_linux_arm64 cmd/coredns.go
