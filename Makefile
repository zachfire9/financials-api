.PHONY: build-FinancialsApiFunction

build-FinancialsApiFunction:
	GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o $(ARTIFACTS_DIR)/bootstrap ./cmd/lambda
