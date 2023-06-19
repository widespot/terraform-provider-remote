default: testacc

# Run acceptance tests
.PHONY: testacc
testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

doc:
	go generate ./...

INSTALL_DIR=playground
TARGET_ARCH=$(shell go env GOOS)_$(shell go env GOARCH)
PROVIDER_PATH=.terraform/providers/registry.terraform.io/widespot/remote/99.0.0/$(TARGET_ARCH)
BIN_PATH=$(INSTALL_DIR)/$(PROVIDER_PATH)/terraform-provider-remote_v99.0.0
install:
	mkdir -p $(INSTALL_DIR)/$(PROVIDER_PATH)
	go build -ldflags="-s -w -X main.version=99.0.0" -o $(BIN_PATH)

playground:
	docker-compose -f playground/docker-compose.yml up -d

clean:
	docker-compose -f playground/docker-compose.yml down
