// Package api runs posture reviews and serves the HTTP API for the dashboard.
package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/google/uuid"

	"github.com/sriramg-aivar/nexogent/internal/auth"
	"github.com/sriramg-aivar/nexogent/internal/bedrock"
	"github.com/sriramg-aivar/nexogent/internal/collectors"
	"github.com/sriramg-aivar/nexogent/internal/discovery"
	"github.com/sriramg-aivar/nexogent/internal/ranking"
	"github.com/sriramg-aivar/nexogent/internal/store"
	"github.com/sriramg-aivar/nexogent/internal/types"
)

// supportedServices are the services shown on the portal landing page.
var supportedServices = []types.ServiceInfo{
	{Key: "ec2", Name: "Amazon EC2", Scope: "regional"},
	{Key: "s3", Name: "Amazon S3", Scope: "global"},
	{Key: "iam", Name: "AWS IAM", Scope: "global"},
	{Key: "lambda", Name: "AWS Lambda", Scope: "regional"},
	{Key: "rds", Name: "Amazon RDS", Scope: "regional"},
	{Key: "dynamodb", Name: "Amazon DynamoDB", Scope: "regional"},
	{Key: "eks", Name: "Amazon EKS", Scope: "regional"},
	{Key: "cloudtrail", Name: "AWS CloudTrail", Scope: "regional"},
}

// Server holds dependencies and the review store.
type Server struct {
	cfg            aws.Config
	region         string
	explainer      *bedrock.Explainer
	prowlerResults string
	scoutResults   string
	cspResults     string
	store          store.Store
	gate           *auth.Gate

	mu            sync.RWMutex
	cachedRegions []string
	regionsAt     time.Time
}

func NewServer(cfg aws.Config, region, bedrockModel, prowlerResults, scoutResults, cspResults string, st store.Store, gate *auth.Gate) *Server {
	return &Server{
		cfg: cfg, region: region,
		explainer:      bedrock.New(cfg, bedrockModel),
		prowlerResults: prowlerResults,
		scoutResults:   scoutResults,
		cspResults:     cspResults,
		store:          st,
		gate:           gate,
	}
}

func (s *Server) accountID(ctx context.Context) string {
	out, err := sts.NewFromConfig(s.cfg).GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "unknown"
	}
	return aws.ToString(out.Account)
}

// regions returns the enabled regions, cached for 10 minutes.
func (s *Server) regions(ctx context.Context) []string {
	s.mu.RLock()
	if len(s.cachedRegions) > 0 && time.Since(s.regionsAt) < 10*time.Minute {
		r := s.cachedRegions
		s.mu.RUnlock()
		return r
	}
	s.mu.RUnlock()

	disc := discovery.New(s.cfg, s.accountID(ctx), s.region)
	regions, _ := disc.EnabledRegions(ctx)
	s.mu.Lock()
	s.cachedRegions = regions
	s.regionsAt = time.Now()
	s.mu.Unlock()
	return regions
}

// RunReview performs a FULL read-only posture review across all regions and
// persists it as the canonical "latest" review.
func (s *Server) RunReview(ctx context.Context, _, _ string) (*types.PostureReview, error) {
	review := s.runScan(ctx, "", "")
	if err := s.store.Put(ctx, review); err != nil {
		log.Printf("store.Put: %v", err)
	}
	return review, nil
}

// RunAudit performs a SCOPED on-demand audit (one service and/or resource).
// It does NOT replace the canonical latest full review; instead it merges the
// fresh findings for that scope back into the stored latest so the portal
// stays consistent, and returns the scoped result to the caller.
func (s *Server) RunAudit(ctx context.Context, service, resource string) *types.PostureReview {
	scoped := s.runScan(ctx, service, resource)
	s.mergeScopedIntoLatest(ctx, service, resource, scoped)
	return scoped
}

// runScan does the actual read-only discovery + collection, optionally scoped.
func (s *Server) runScan(ctx context.Context, filterService, filterResource string) *types.PostureReview {
	start := time.Now()
	accountID := s.accountID(ctx)
	regions := s.regions(ctx)

	disc := discovery.New(s.cfg, accountID, s.region)
	resources, dwarns := disc.Inventory(ctx)
	// Scope resources to match the filter (keeps the scoped review coherent).
	resources = filterResources(resources, filterService, filterResource)

	own := collectors.NewOwnChecks(s.cfg, accountID, s.region).
		WithRegions(regions).
		WithFilter(filterService, filterResource)
	findings, cwarns := own.Collect(ctx)

	sourcesRun := []string{"own-check"}
	warnings := append(dwarns, cwarns...)

	// Prowler results (written by the Prowler CronJob / on-demand Job).
	prowler := collectors.NewProwlerCollector(s.prowlerResults, accountID, s.region)
	pfindings, pwarns := prowler.Collect(ctx)
	pfindings = filterFindings(pfindings, filterService, filterResource)
	if len(pfindings) > 0 {
		findings = append(findings, pfindings...)
		sourcesRun = append(sourcesRun, "prowler")
	}
	warnings = append(warnings, pwarns...)

	// ScoutSuite (open-source multi-service posture auditor).
	if s.scoutResults != "" {
		scout := collectors.NewScoutSuiteCollector(s.scoutResults, accountID, s.region)
		sfindings, swarns := scout.Collect(ctx)
		sfindings = filterFindings(sfindings, filterService, filterResource)
		if len(sfindings) > 0 {
			findings = append(findings, sfindings...)
			sourcesRun = append(sourcesRun, "scoutsuite")
		}
		warnings = append(warnings, swarns...)
	}

	// cloudsplaining (open-source IAM policy risk). Only relevant for IAM scope.
	if s.cspResults != "" && (filterService == "" || filterService == "iam") {
		csp := collectors.NewCloudsplainingCollector(s.cspResults, accountID, s.region)
		cfindings, cwarns2 := csp.Collect(ctx)
		cfindings = filterFindings(cfindings, filterService, filterResource)
		if len(cfindings) > 0 {
			findings = append(findings, cfindings...)
			sourcesRun = append(sourcesRun, "cloudsplaining")
		}
		warnings = append(warnings, cwarns2...)
	}

	findings = ranking.RankFindings(findings)
	score := ranking.PostureScore(findings)

	return &types.PostureReview{
		ReviewID: uuid.NewString(), AccountID: accountID, Regions: regions,
		Status: "completed", CreatedAt: time.Now().UTC(),
		DurationSeconds: time.Since(start).Seconds(),
		PostureScore:    score, Rating: ranking.RatingFor(score),
		Summary:    summarize(findings), Findings: findings, Resources: resources,
		SourcesRun: sourcesRun,
		Warnings:   warnings,
	}
}

// mergeScopedIntoLatest updates the stored latest full review: it replaces the
// findings for the scoped service/resource with the fresh ones, and merges any
// newly discovered resources. If there is no latest yet, the scoped result is
// persisted as-is.
func (s *Server) mergeScopedIntoLatest(ctx context.Context, service, resource string, scoped *types.PostureReview) {
	latest, _ := s.store.Latest(ctx)
	if latest == nil {
		_ = s.store.Put(ctx, scoped)
		return
	}
	// Drop old findings in scope, keep the rest.
	var kept []types.Finding
	for _, f := range latest.Findings {
		if service != "" && f.Service != service {
			kept = append(kept, f)
			continue
		}
		if resource != "" && f.ResourceID != resource {
			kept = append(kept, f)
			continue
		}
		// in-scope -> drop (will be replaced by fresh scoped findings)
	}
	kept = append(kept, scoped.Findings...)
	kept = ranking.RankFindings(kept)

	// Merge resources by ID.
	resByID := map[string]types.Resource{}
	for _, r := range latest.Resources {
		resByID[r.ID] = r
	}
	for _, r := range scoped.Resources {
		resByID[r.ID] = r
	}
	merged := make([]types.Resource, 0, len(resByID))
	for _, r := range resByID {
		merged = append(merged, r)
	}

	latest.Findings = kept
	latest.Resources = merged
	latest.Summary = summarize(kept)
	latest.PostureScore = ranking.PostureScore(kept)
	latest.Rating = ranking.RatingFor(latest.PostureScore)
	latest.CreatedAt = time.Now().UTC()
	if err := s.store.Put(ctx, latest); err != nil {
		log.Printf("merge store.Put: %v", err)
	}
}

func filterResources(rs []types.Resource, service, resource string) []types.Resource {
	if service == "" && resource == "" {
		return rs
	}
	var out []types.Resource
	for _, r := range rs {
		if service != "" && r.Service != service {
			continue
		}
		if resource != "" && r.ID != resource {
			continue
		}
		out = append(out, r)
	}
	return out
}

func filterFindings(fs []types.Finding, service, resource string) []types.Finding {
	if service == "" && resource == "" {
		return fs
	}
	var out []types.Finding
	for _, f := range fs {
		if service != "" && f.Service != service {
			continue
		}
		if resource != "" && f.ResourceID != resource {
			continue
		}
		out = append(out, f)
	}
	return out
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

// ---- HTTP routing ----

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.health)
	mux.HandleFunc("/api/v1/health", s.health)

	// Auth (public).
	mux.HandleFunc("/api/v1/auth/login", s.login)
	mux.HandleFunc("/api/v1/auth/status", s.authStatus)

	// Reviews.
	mux.HandleFunc("/api/v1/review", s.runReview)        // POST (full account review)
	mux.HandleFunc("/api/v1/review/latest", s.getLatest) // GET
	mux.HandleFunc("/api/v1/reviews", s.listReviews)     // GET (history)

	// Portal navigation.
	mux.HandleFunc("/api/v1/services", s.getServices)   // GET landing: services + counts
	mux.HandleFunc("/api/v1/regions", s.getRegions)     // GET enabled regions
	mux.HandleFunc("/api/v1/resources", s.getResources) // GET ?service=&region=
	mux.HandleFunc("/api/v1/resource", s.getResource)   // GET ?id= (resource audit view)

	// On-demand audit (B5).
	mux.HandleFunc("/api/v1/audit", s.audit) // POST ?service=&resource=

	// Explanations + integration toggles.
	mux.HandleFunc("/api/v1/explain", s.explain)           // POST ?review=&finding=
	mux.HandleFunc("/api/v1/integrations", s.integrations) // GET (disabled toggles)

	return cors(s.gate.Middleware(mux))
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"status": "ok", "service": "nexogent", "store": s.store.Kind()})
}

// ---- Auth handlers ----

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", 405)
		return
	}
	var body struct {
		Password string `json:"password"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	tok, ok := s.gate.Login(body.Password)
	if !ok {
		writeJSON(w, 401, map[string]string{"error": "invalid credentials"})
		return
	}
	writeJSON(w, 200, map[string]string{"token": tok, "role": auth.Role})
}

func (s *Server) authStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"auth_required": s.gate.Enabled(), "role": auth.Role})
}

// ---- Review handlers ----

func (s *Server) runReview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", 405)
		return
	}
	review, err := s.RunReview(r.Context(), "", "")
	if err != nil {
		writeJSON(w, 502, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, review)
}

func (s *Server) getLatest(w http.ResponseWriter, r *http.Request) {
	rev, _ := s.store.Latest(r.Context())
	if rev == nil {
		writeJSON(w, 404, map[string]string{"error": "no review yet"})
		return
	}
	writeJSON(w, 200, rev)
}

func (s *Server) listReviews(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	reviews, _ := s.store.List(r.Context(), limit)
	// Return a compact history (no full findings) for the trends view.
	type item struct {
		ReviewID     string    `json:"review_id"`
		CreatedAt    time.Time `json:"created_at"`
		PostureScore int       `json:"posture_score"`
		Rating       string    `json:"rating"`
		Total        int       `json:"total_findings"`
	}
	var out []item
	for _, rv := range reviews {
		out = append(out, item{rv.ReviewID, rv.CreatedAt, rv.PostureScore, rv.Rating, rv.Summary.Total})
	}
	writeJSON(w, 200, map[string]any{"reviews": out})
}

// ---- Portal navigation handlers ----

// getServices returns the supported services with resource/finding counts
// from the latest review (B2 landing page).
func (s *Server) getServices(w http.ResponseWriter, r *http.Request) {
	rev, _ := s.store.Latest(r.Context())
	resCount := map[string]int{}
	findCount := map[string]int{}
	if rev != nil {
		for _, res := range rev.Resources {
			resCount[res.Service]++
		}
		for _, f := range rev.Findings {
			findCount[f.Service]++
		}
	}
	out := make([]types.ServiceInfo, 0, len(supportedServices))
	for _, svc := range supportedServices {
		svc.ResourceCount = resCount[svc.Key]
		svc.FindingCount = findCount[svc.Key]
		out = append(out, svc)
	}
	writeJSON(w, 200, map[string]any{
		"services":   out,
		"has_review": rev != nil,
	})
}

func (s *Server) getRegions(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"regions": s.regions(r.Context())})
}

// getResources lists resources, optionally filtered by ?service= and ?region=
// (B3 service page + region filter).
func (s *Server) getResources(w http.ResponseWriter, r *http.Request) {
	rev, _ := s.store.Latest(r.Context())
	if rev == nil {
		writeJSON(w, 404, map[string]string{"error": "no review yet — run an audit first"})
		return
	}
	svc := r.URL.Query().Get("service")
	region := r.URL.Query().Get("region")

	var resources []types.Resource
	regionsSet := map[string]struct{}{}
	findByResource := map[string]int{}
	for _, f := range rev.Findings {
		findByResource[f.ResourceID]++
	}
	for _, res := range rev.Resources {
		if svc != "" && res.Service != svc {
			continue
		}
		regionsSet[res.Region] = struct{}{}
		if region != "" && res.Region != region {
			continue
		}
		resources = append(resources, res)
	}
	sort.Slice(resources, func(i, j int) bool {
		if resources[i].Region != resources[j].Region {
			return resources[i].Region < resources[j].Region
		}
		return resources[i].Name < resources[j].Name
	})

	// Regions present for this service (for the region filter dropdown).
	regionsList := make([]string, 0, len(regionsSet))
	for rg := range regionsSet {
		regionsList = append(regionsList, rg)
	}
	sort.Strings(regionsList)

	// Attach finding counts per resource.
	type resWithCount struct {
		types.Resource
		FindingCount int `json:"finding_count"`
	}
	enriched := make([]resWithCount, 0, len(resources))
	for _, res := range resources {
		enriched = append(enriched, resWithCount{res, findByResource[res.ID]})
	}

	writeJSON(w, 200, map[string]any{
		"service":   svc,
		"region":    region,
		"regions":   regionsList,
		"resources": enriched,
	})
}

// getResource returns one resource plus its findings (B4 resource = audit page).
func (s *Server) getResource(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeJSON(w, 400, map[string]string{"error": "id required"})
		return
	}
	rev, _ := s.store.Latest(r.Context())
	if rev == nil {
		writeJSON(w, 404, map[string]string{"error": "no review yet"})
		return
	}
	var resource *types.Resource
	for i := range rev.Resources {
		if rev.Resources[i].ID == id {
			resource = &rev.Resources[i]
			break
		}
	}
	var findings []types.Finding
	for _, f := range rev.Findings {
		if f.ResourceID == id {
			findings = append(findings, f)
		}
	}
	sort.SliceStable(findings, func(i, j int) bool { return findings[i].RiskScore > findings[j].RiskScore })
	writeJSON(w, 200, map[string]any{
		"review_id": rev.ReviewID,
		"resource":  resource,
		"findings":  findings,
		"count":     len(findings),
	})
}

// audit runs scanners on demand for a service/resource (B5). It triggers a
// fresh scoped review and returns its findings.
func (s *Server) audit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", 405)
		return
	}
	service := r.URL.Query().Get("service")
	resource := r.URL.Query().Get("resource")
	review := s.RunAudit(r.Context(), service, resource)
	writeJSON(w, 200, map[string]any{
		"review_id":   review.ReviewID,
		"service":     service,
		"resource":    resource,
		"findings":    review.Findings,
		"count":       len(review.Findings),
		"sources_run": review.SourcesRun,
		"scanned_at":  review.CreatedAt,
	})
}

// ---- Explain + integrations ----

func (s *Server) explain(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", 405)
		return
	}
	reviewID := r.URL.Query().Get("review")
	findingID := r.URL.Query().Get("finding")
	review, _ := s.store.Get(r.Context(), reviewID)
	if review == nil {
		// Fall back to latest (on-demand audits return a review_id the client may re-use).
		review, _ = s.store.Latest(r.Context())
	}
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

// integrations returns GuardDuty / Security Hub / Inspector as DISABLED toggles
// (B6). They are not wired to AWS in v1.
func (s *Server) integrations(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"integrations": []types.Integration{
		{Key: "guardduty", Name: "Amazon GuardDuty", Enabled: false, Available: false,
			Description: "Continuous threat detection (active threats, not just misconfig).",
			Note:        "Disabled in v1. Enable in a later version."},
		{Key: "securityhub", Name: "AWS Security Hub", Enabled: false, Available: false,
			Description: "Aggregated findings and compliance standards across services.",
			Note:        "Disabled in v1. Enable in a later version."},
		{Key: "inspector", Name: "Amazon Inspector", Enabled: false, Available: false,
			Description: "Automated vulnerability scanning for workloads and images.",
			Note:        "Disabled in v1. Enable in a later version."},
	}})
}

// ---- helpers ----

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func cors(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", strings.Join([]string{"Authorization", "Content-Type"}, ", "))
		if r.Method == http.MethodOptions {
			w.WriteHeader(204)
			return
		}
		h.ServeHTTP(w, r)
	})
}
