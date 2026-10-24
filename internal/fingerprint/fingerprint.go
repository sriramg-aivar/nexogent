// Package fingerprint computes a stable identity for a finding so it survives
// rescans, and maps source-specific check ids to a shared canonical key so the
// same problem reported by different sources deduplicates (F1, F2).
package fingerprint

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"strings"
)

//go:embed equivalence.json
var equivalenceJSON []byte

// equivalence maps source -> checkID -> canonical key. Loaded once.
var equivalence map[string]map[string]string

func init() {
	_ = json.Unmarshal(equivalenceJSON, &equivalence)
	if equivalence == nil {
		equivalence = map[string]map[string]string{}
	}
}

// Canonical returns the canonical key for a (source, checkID) pair. When the
// pair is not mapped, it returns "<source>:<checkID>" so the finding is never
// dropped and simply does not merge with others.
func Canonical(source, checkID string) string {
	source = strings.ToLower(strings.TrimSpace(source))
	checkID = strings.TrimSpace(checkID)
	if m, ok := equivalence[source]; ok {
		if key, ok := m[checkID]; ok && key != "" {
			return key
		}
	}
	if checkID == "" {
		return source
	}
	return source + ":" + checkID
}

// Of builds the stable fingerprint from identifying fields. It is pure and
// deterministic: the same inputs always produce the same 16-hex-char output.
// Region is lower-cased so case differences do not change the id.
func Of(account, service, region, resourceID, canonicalKey string) string {
	parts := []string{
		strings.ToLower(strings.TrimSpace(account)),
		strings.ToLower(strings.TrimSpace(service)),
		strings.ToLower(strings.TrimSpace(region)),
		strings.TrimSpace(resourceID),
		strings.TrimSpace(canonicalKey),
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:])[:16]
}

// NormaliseResourceID normalises a resource id to a stable ARN-like form so the
// same resource reported as a bare id and as an ARN produces the same id.
func NormaliseResourceID(service, id, region, account string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return id
	}
	if strings.HasPrefix(id, "arn:") {
		return id
	}
	switch strings.ToLower(service) {
	case "s3":
		return "arn:aws:s3:::" + id
	case "ec2":
		switch {
		case strings.HasPrefix(id, "i-"):
			return "arn:aws:ec2:" + region + ":" + account + ":instance/" + id
		case strings.HasPrefix(id, "sg-"):
			return "arn:aws:ec2:" + region + ":" + account + ":security-group/" + id
		case strings.HasPrefix(id, "vol-"):
			return "arn:aws:ec2:" + region + ":" + account + ":volume/" + id
		}
	case "iam":
		// user/role name -> leave as-is unless clearly a name
		if !strings.Contains(id, "/") {
			return "arn:aws:iam::" + account + ":user/" + id
		}
	case "dynamodb":
		return "arn:aws:dynamodb:" + region + ":" + account + ":table/" + id
	case "eks":
		return "arn:aws:eks:" + region + ":" + account + ":cluster/" + id
	}
	return id
}
