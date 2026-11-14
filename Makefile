.PHONY: all build test bench clean web

all: build

web:
	cd web && npm run build

build: web
	go build -v -o tokenprune.exe ./cmd/tokenprune

test:
	go test -v -race ./...

bench:
	go test -bench=. -benchmem ./internal/...

clean:
	rm -f tokenprune tokenprune.exe
	rm -rf web/dist
