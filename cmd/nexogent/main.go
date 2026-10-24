// Command nexogent runs the SecOps agent API server.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"github.com/sriramg-aivar/nexogent/internal/api"
	"github.com/sriramg-aivar/nexogent/internal/auth"
	"github.com/sriramg-aivar/nexogent/internal/store"
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
	scoutResults := env("SCOUTSUITE_RESULTS_DIR", "/data/scoutsuite")
	cspResults := env("CLOUDSPLAINING_RESULTS_DIR", "/data/cloudsplaining")
	reviewsTable := env("REVIEWS_TABLE", "") // DynamoDB table; empty => in-memory
	adminPassword := env("SUPERADMIN_PASSWORD", "")
	scanInterval := env("SCAN_INTERVAL", "") // e.g. "6h"; empty => no in-process scheduler

	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		log.Fatalf("load AWS config: %v", err)
	}

	// Store: DynamoDB if a table is configured, else in-memory.
	var st store.Store
	if reviewsTable != "" {
		st = store.NewDynamo(dynamodb.NewFromConfig(cfg), reviewsTable)
		log.Printf("persistence: DynamoDB table %q", reviewsTable)
	} else {
		st = store.NewMemory()
		log.Printf("persistence: in-memory (set REVIEWS_TABLE for DynamoDB)")
	}

	// Auth gate (single superadmin role).
	gate := auth.New(adminPassword)
	if !gate.Enabled() {
		log.Printf("WARNING: SUPERADMIN_PASSWORD not set — auth gate is OPEN (dev only)")
	} else {
		log.Printf("auth: superadmin password gate enabled")
	}

	srv := api.NewServer(cfg, region, bedrockModel, prowlerResults, scoutResults, cspResults, st, gate)

	// Optional in-process scheduler for periodic reviews (E3).
	if scanInterval != "" {
		if d, err := time.ParseDuration(scanInterval); err == nil && d > 0 {
			log.Printf("scheduler: running a review every %s", d)
			go runScheduler(ctx, srv, d)
		} else {
			log.Printf("scheduler: invalid SCAN_INTERVAL %q, disabled", scanInterval)
		}
	}

	log.Printf("nexogent listening on %s (region %s)", addr, region)
	if err := http.ListenAndServe(addr, srv.Routes()); err != nil {
		log.Fatal(err)
	}
}

func runScheduler(ctx context.Context, srv *api.Server, d time.Duration) {
	// Run once shortly after start, then on the interval.
	time.Sleep(20 * time.Second)
	for {
		rctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		if _, err := srv.RunReview(rctx, "", ""); err != nil {
			log.Printf("scheduled review failed: %v", err)
		} else {
			log.Printf("scheduled review completed")
		}
		cancel()
		time.Sleep(d)
	}
}
