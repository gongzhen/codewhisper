.PHONY: build run test clean

build:
	go build -o bin/codewhisper cmd/codewhisper/main.go

run: build
	./bin/codewhisper

test:
	go test ./...

clean:
	rm -rf bin/

install: build
	cp bin/codewhisper $(GOPATH)/bin/