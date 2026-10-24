package report

import (
	"fmt"
	"time"
)

// Status is a report's workflow state (F10).
type Status string

const (
	StatusDraft    Status = "draft"
	StatusInReview Status = "in_review"
	StatusApproved Status = "approved"
	StatusRejected Status = "rejected"
)

// Meta is the report workflow record.
type Meta struct {
	ID         string     `json:"id"`
	Scope      string     `json:"scope"`
	Version    int        `json:"version"`
	ParentID   string     `json:"parent_id,omitempty"`
	Status     Status     `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
	CreatedBy  string     `json:"created_by"`
	Reviewer   string     `json:"reviewer,omitempty"`
	ReviewNote string     `json:"review_note,omitempty"`
	ReviewedAt *time.Time `json:"reviewed_at,omitempty"`
}

// Transition applies an action to a report's status, enforcing allowed moves.
// action is one of: submit, approve, reject. Returns the new status or an error
// (the caller maps the error to HTTP 409).
func Transition(cur Status, action, reviewer, note string) (Status, error) {
	switch action {
	case "submit":
		if cur != StatusDraft {
			return cur, fmt.Errorf("can only submit a draft")
		}
		if reviewer == "" {
			return cur, fmt.Errorf("reviewer is required to submit")
		}
		return StatusInReview, nil
	case "approve":
		if cur != StatusInReview {
			return cur, fmt.Errorf("can only approve a report in review")
		}
		return StatusApproved, nil
	case "reject":
		if cur != StatusInReview {
			return cur, fmt.Errorf("can only reject a report in review")
		}
		if note == "" {
			return cur, fmt.Errorf("a note is required to reject")
		}
		return StatusRejected, nil
	default:
		return cur, fmt.Errorf("unknown action %q", action)
	}
}

// buildSummary builds a deterministic executive summary from the numbers.
func buildSummary(d Data) string {
	counts := map[string]int{}
	for _, g := range d.ByService {
		for _, f := range g.Findings {
			if f.Status == "fixed" || f.Status == "accepted" {
				continue
			}
			counts[string(f.Severity)]++
		}
	}
	s := fmt.Sprintf("The account scored %d/100 (%s).", d.Score, d.Rating)
	if d.PrevScore != nil {
		diff := d.Score - *d.PrevScore
		switch {
		case diff > 0:
			s += fmt.Sprintf(" The score improved by %d since the last report.", diff)
		case diff < 0:
			s += fmt.Sprintf(" The score fell by %d since the last report.", -diff)
		default:
			s += " The score is unchanged since the last report."
		}
	}
	s += fmt.Sprintf(" There are %d critical and %d high findings open.", counts["CRITICAL"], counts["HIGH"])
	if len(d.Paths) > 0 {
		s += fmt.Sprintf(" %d attack path(s) were identified and should be prioritised.", len(d.Paths))
	}
	if d.Changes.FixedCount > 0 {
		s += fmt.Sprintf(" %d finding(s) were fixed since the last scan.", d.Changes.FixedCount)
	}
	return s
}
