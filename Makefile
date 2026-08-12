TEST ?= ./...
TESTARGS ?=
BINARY = terraform-provider-tailscale-membership

.PHONY: build install test testacc testacc_local format

default: install

build:
	go build -o $(BINARY) .

install:
	go install .

test:
	go test -race $(TEST)

testacc:
	TF_ACC=1 go test $(TEST) -v $(TESTARGS) -timeout 120m

testacc_local:
	TF_ACC=1 TAILSCALE_BASE_URL=http://localhost:31544 TAILSCALE_API_KEY=$$(jq -r .apiKey /tmp/terraform-api-key.json) go test $(TEST) -v $(TESTARGS) -timeout 120m

format:
	go fmt ./...
	go run golang.org/x/tools/cmd/goimports -w -local github.com/pmpaulino .
	go mod tidy
