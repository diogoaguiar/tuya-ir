.PHONY: build test install clean

build:
	go build -o bin/tuya-ir ./cmd/tuya-ir/

test:
	go test ./...

install:
	go install ./cmd/tuya-ir/

clean:
	rm -rf bin/
