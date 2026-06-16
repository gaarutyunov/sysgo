.PHONY: build test cover lint fmt generate-example clean

build:
	go build ./...

test:
	go test -race ./...

cover:
	go test -covermode=atomic -coverprofile=coverage.out -coverpkg=./... ./...
	go tool cover -func=coverage.out | tail -n 1

lint:
	gofmt -l .
	go vet ./...

fmt:
	gofmt -w $(shell find . -name '*.go' -not -path './examples/*')

generate-example:
	go run ./cmd/sysgo generate -c examples/order/sysgo.yaml --out ./out

clean:
	rm -rf out coverage.out
