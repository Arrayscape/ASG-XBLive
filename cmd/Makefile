all: tidy fmt vet test build

fmt:
	go fmt ./...

vet:
	go vet ./...
	staticcheck

test:
	go test -v ./...

tidy:
	go mod tidy

build:
	go build -o xblive
