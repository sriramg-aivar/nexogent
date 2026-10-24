// Command nexogent runs the SecOps agent API server.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"

	"github.com/sriramg-aivar/nexogent/internal/api"
)

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	region := env("AWS_REGION", env("AWS_DEFAULT_REGION", "us-east-1"))
	bedrockModel := env("BEDROCK_MODEL_ID", "us.anthropic.claude-haiku-4-5-20251001-v1:0")
	addr := env("LISTEN_ADDR", ":8080")
	prowlerResults := env("PROWLER_RESULTS_DIR", "/data/prowler")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		log.Fatalf("load AWS config: %v", err)
	}

	srv := api.NewServer(cfg, region, bedrockModel, prowlerResults)
	log.Printf("nexogent listening on %s (region %s)", addr, region)
	if err := http.ListenAndServe(addr, srv.Routes()); err != nil {
		log.Fatal(err)
	}
}
