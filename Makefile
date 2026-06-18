.PHONY: build test cover lint fmt model generate-example clean

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

# Transform the real .sysml source to the JSON intermediate using real SysML
# tooling (the OMG Pilot Implementation serializer). Requires Java + curl.
model:
	scripts/sysml2json.sh examples/order/OrderContext.sysml examples/order/model.json

generate-example:
	go run ./cmd/sysgo generate -c examples/order/sysgo.yaml --out ./out

clean:
	rm -rf out coverage.out
