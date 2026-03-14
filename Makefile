.PHONY: build test install clean

build:
	go build -o logs-linter ./cmd/logs-linter

test:
	go test -v ./...

install:
	go install ./cmd/logs-linter

clean:
	rm -f logs-linter
