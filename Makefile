.PHONY: build run clean

build:
\tgo build -o bin/dbback ./cmd/dbback

run: build
\t./bin/dbback

clean:
\trm -rf bin

