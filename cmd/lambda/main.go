package main

import (
	"log"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/zachfire9/financials-api/internal/app"
	"github.com/zachfire9/financials-api/internal/config"
	"github.com/zachfire9/financials-api/internal/lambdahttp"
)

func main() {
	cfg, err := config.Load(".env")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	handler, err := app.NewHandler(cfg)
	if err != nil {
		log.Fatalf("configure application: %v", err)
	}

	log.Printf("financials-api lambda starting with %s storage", cfg.StorageDriver)
	adapter := lambdahttp.NewAdapter(handler)
	lambda.Start(adapter.Proxy)
}
