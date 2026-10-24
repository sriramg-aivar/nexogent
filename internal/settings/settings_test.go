package settings

import "testing"

func TestDefaultsValid(t *testing.T) {
	if err := Defaults().Validate(); err != nil {
		t.Fatalf("defaults should be valid: %v", err)
	}
}

func TestWeightsMustSumToOne(t *testing.T) {
	s := Defaults()
	s.Rubric.Weights.Severity = 0.9 // sum now > 1
	if err := s.Validate(); err == nil {
		t.Fatal("expected invalid weights to be rejected")
	}
}

func TestInvalidInterval(t *testing.T) {
	s := Defaults()
	s.ScanInterval = "3h"
	if err := s.Validate(); err == nil {
		t.Fatal("expected invalid interval rejected")
	}
}

func TestLogoRejectsSVG(t *testing.T) {
	if err := ValidateLogo("data:image/svg+xml;base64,PHN2Zz4="); err == nil {
		t.Fatal("svg logo must be rejected")
	}
	if err := ValidateLogo("data:image/png;base64,iVBORw0KG"); err != nil {
		t.Fatalf("png logo should pass: %v", err)
	}
}
