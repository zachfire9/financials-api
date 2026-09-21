.PHONY: build-FinancialsApiFunction

export GOOS := linux
export GOARCH := arm64

BOOTSTRAP := $(ARTIFACTS_DIR)/bootstrap

build-FinancialsApiFunction:
	go build -tags lambda.norpc -o "$(BOOTSTRAP)" ./cmd/lambda
