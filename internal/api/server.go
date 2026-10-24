// Package api runs posture reviews and serves the HTTP API for the dashboard.
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/google/uuid"

	"github.com/sriramg-aivar/nexogent/internal/bedrock"
	"github.com/sriramg-aivar/nexogent/internal/collectors"
	"github.com/sriramg-aivar/nexogent/internal/discovery"
	"github.com/sriramg-aivar/nexogent/internal/ranking"
	"github.com/sriramg-aivar/nexogent/internal/types"
)

// Server holds dependencies and the in-memory review store.
type Server struct {
	cfg            aws.Config
	region         string
	explainer      *bedrock.Explainer
	prowlerResults string // dir where the Prowler CronJob writes OCSF JSON

	mu      sync.RWMutex
	reviews map[string]*types.PostureReview
	latest  *types.PostureReview
}

func NewServer(cfg aws.Config, region, bedrockModel, prowlerResults string) *Server {
	return &Server{
		cfg: cfg, region: region,
		explainer:      bedrock.New(cfg, bedrockModel),
		prowlerResults: prowlerResults,
		reviews:        map[string]*types.PostureReview{},
	}
}

// RunReview performs a full read-only posture review.
func (s *Server) RunReview(ctx context.Context) (*types.PostureReview, error) {
	start := time.Now()
	accountID := s.accountID(ctx)

	disc := discovery.New(s.cfg, accountID, s.region)
	resources, dwarns := disc.Inventory(ctx)

	own := collectors.NewOwnChecks(s.cfg, accountID, s.region)
	findings, cwarns := own.Collect(ctx)

	sourcesRun := []string{"own-check"}
	warnings := append(dwarns, cwarns...)

	// Prowler (open-source scanner) results, written by the Prowler CronJob.
	prowler := collectors.NewProwlerCollector(s.prowlerResults, accountID, s.region)
	pfindings, pwarns := prowler.Collect(ctx)
	if len(pfindings) > 0 {
		findings = append(findings, pfindings...)
		sourcesRun = append(sourcesRun, "prowler")
	}
	warnings = append(warnings, pwarns...)

	findings = ranking.RankFindings(findings)
	score := ranking.PostureScore(findings)

	review := &types.PostureReview{
		ReviewID: uuid.NewString(), AccountID: accountID, Regions: []string{s.region},
		Status: "completed", CreatedAt: time.Now().UTC(),
		DurationSeconds: time.Since(start).Seconds(),
		PostureScore:    score, Rating: ranking.RatingFor(score),
		Summary:    summarize(findings), Findings: findings, Resources: resources,
		SourcesRun: sourcesRun,
		Warnings:   warnings,
	}

	s.mu.Lock()
	s.reviews[review.ReviewID] = review
	s.latest = review
	s.mu.Unlock()
	return review, nil
}

func (s *Server) accountID(ctx context.Context) string {
	out, err := sts.NewFromConfig(s.cfg).GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "unknown"
	}
	return aws.ToString(out.Account)
}

func summarize(fs []types.Finding) types.PostureSummary {
	sum := types.PostureSummary{Total: len(fs),
		BySeverity: map[string]int{}, ByService: map[string]int{}, BySource: map[string]int{}}
	for _, f := range fs {
		sum.BySeverity[string(f.Severity)]++
		sum.ByService[f.Service]++
		sum.BySource[f.Source]++
	}
	return sum
}

// ---- HTTP handlers ----

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.health)
	mux.HandleFunc("/api/v1/health", s.health)
	mux.HandleFunc("/api/v1/review", s.runReview)          // POST
	mux.HandleFunc("/api/v1/review/latest", s.getLatest)   // GET
	mux.HandleFunc("/api/v1/resources", s.getResources)    // GET (grouped by service)
	mux.HandleFunc("/api/v1/explain", s.explain)           // POST ?review=&finding=
	return cors(mux)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]string{"status": "ok", "service": "nexogent"})
}

func (s *Server) runReview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", 405)
		return
	}
	review, err := s.RunReview(r.Context())
	if err != nil {
		writeJSON(w, 502, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, review)
}

func (s *Server) getLatest(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.latest == nil {
		writeJSON(w, 404, map[string]string{"error": "no review yet"})
		return
	}
	writeJSON(w, 200, s.latest)
}

// getResources returns the latest inventory grouped by service.
func (s *Server) getResources(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.latest == nil {
		writeJSON(w, 404, map[string]string{"error": "no review yet"})
		return
	}
	grouped := map[string][]types.Resource{}
	for _, res := range s.latest.Resources {
		grouped[res.Service] = append(grouped[res.Service], res)
	}
	services := make([]string, 0, len(grouped))
	for svc := range grouped {
		services = append(services, svc)
	}
	sort.Strings(services)
	writeJSON(w, 200, map[string]any{"services": services, "by_service": grouped})
}

func (s *Server) explain(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", 405)
		return
	}
	reviewID := r.URL.Query().Get("review")
	findingID := r.URL.Query().Get("finding")
	s.mu.RLock()
	review := s.reviews[reviewID]
	s.mu.RUnlock()
	if review == nil {
		writeJSON(w, 404, map[string]string{"error": "review not found"})
		return
	}
	for _, f := range review.Findings {
		if f.ID == findingID {
			writeJSON(w, 200, s.explainer.Explain(r.Context(), f))
			return
		}
	}
	writeJSON(w, 404, map[string]string{"error": "finding not found"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func cors(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		if r.Method == http.MethodOptions {
			w.WriteHeader(204)
			return
		}
		h.ServeHTTP(w, r)
	})
}
