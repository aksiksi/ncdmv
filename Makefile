SHELL := /bin/bash

all: db-gen server test

server:
	go build -o ncdmv ./cmd/ncdmv

test:
	go test -v -short ./...

install:
	go build -v -o /usr/local/bin/ncdmv ./cmd/ncdmv

clean:
	rm -f ./ncdmv

docker:
	docker build . -t ncdmv

# Re-generate SQL models from the schema.
db-gen:
	docker run --rm \
		-v $(CURDIR):/src -w /src \
		kjconroy/sqlc generate

.PHONY: all clean db-gen docker install server test
