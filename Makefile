hello:
	@echo Hello World!

test:
	cd server && go test ./...

build:
	cd server && go build ./...

run:
	cd server && go run . serve
