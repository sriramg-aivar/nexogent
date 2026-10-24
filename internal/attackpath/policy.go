package attackpath

import "strings"

// PolicyDoc is a decoded IAM policy document (subset).
type PolicyDoc struct {
	Statement []Statement `json:"Statement"`
}

// Statement is one policy statement. Action/Resource may be a string or []string;
// decode with UnmarshalJSON-friendly types at the caller, or use StringOrSlice.
type Statement struct {
	Effect   string   `json:"Effect"`
	Action   []string `json:"-"`
	Resource []string `json:"-"`
}

// Classification is the result of analysing a role's policies.
type Classification struct {
	IsAdmin         bool
	WildcardActions bool
	S3Broad         bool
}

// Classify analyses decoded statements plus a list of attached managed policy
// names. Pure function, fully testable.
func Classify(statements []Statement, attachedManaged []string) Classification {
	var c Classification
	for _, name := range attachedManaged {
		if name == "AdministratorAccess" {
			c.IsAdmin = true
		}
	}
	for _, st := range statements {
		if !strings.EqualFold(st.Effect, "Allow") {
			continue
		}
		resAll := containsAny(st.Resource, "*")
		for _, a := range st.Action {
			al := strings.ToLower(a)
			if a == "*" && resAll {
				c.IsAdmin = true
				c.WildcardActions = true
			}
			if strings.HasSuffix(al, ":*") && resAll {
				c.WildcardActions = true
			}
			if resAll && (al == "s3:*" || al == "s3:get*" || al == "s3:put*" ||
				strings.HasPrefix(al, "s3:get") || strings.HasPrefix(al, "s3:put")) {
				c.S3Broad = true
			}
		}
	}
	return c
}

func containsAny(xs []string, v string) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}
