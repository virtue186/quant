build:
	go build -o ./bin/quant

run: build
	./bin/quant

test:
	go test -v ./...
