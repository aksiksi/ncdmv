all: db-gen server test

server:
	go build -o ncdmv ./cmd/ncdmv

test:
	go test -v -short ./...

install:
	go build -v -o /usr/local/bin/ncdmv ./cmd/ncdmv

clean:
	rm -f ./ncdmv

shell:
	nix develop -c zsh

docker:
	nix build .#docker

# Re-generate SQL models from the schema.
db-gen:
	docker run --rm \
		-v $(CURDIR):/src -w /src \
		sqlc/sqlc generate

.PHONY: all clean db-gen docker install server test
