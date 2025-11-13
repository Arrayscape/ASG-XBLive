all: tidy fmt vet test build-cmd

fmt:
	go fmt ./...

vet:
	go vet ./...
	staticcheck

test:
	go test -v ./...

tidy:
	go mod tidy

build-cmd:
	cd cmd && make
