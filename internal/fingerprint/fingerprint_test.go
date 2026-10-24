package fingerprint

import "testing"

func TestOfDeterministicAndSensitive(t *testing.T) {
	a := Of("111122223333", "ec2", "us-east-1", "sg-1", "sg-open-world")
	b := Of("111122223333", "ec2", "us-east-1", "sg-1", "sg-open-world")
	if a != b {
		t.Fatalf("not deterministic: %s != %s", a, b)
	}
	if len(a) != 16 {
		t.Fatalf("want 16 hex chars, got %d", len(a))
	}
	// region case must not change the id
	if Of("111122223333", "ec2", "US-EAST-1", "sg-1", "sg-open-world") != a {
		t.Fatal("region case changed fingerprint")
	}
	// any identifying change must change the id
	if Of("111122223333", "ec2", "us-east-1", "sg-2", "sg-open-world") == a {
		t.Fatal("different resource gave same fingerprint")
	}
	if Of("111122223333", "ec2", "us-east-1", "sg-1", "other") == a {
		t.Fatal("different key gave same fingerprint")
	}
}

func TestCanonical(t *testing.T) {
	if got := Canonical("own-check", "ec2-sg-open-world"); got != "sg-open-world" {
		t.Fatalf("own-check mapping: %s", got)
	}
	if got := Canonical("prowler", "ec2_securitygroup_allow_ingress_from_internet_to_tcp_port_22"); got != "sg-open-world" {
		t.Fatalf("prowler mapping: %s", got)
	}
	// unmapped -> source:checkID, never dropped
	if got := Canonical("prowler", "some_unmapped_check"); got != "prowler:some_unmapped_check" {
		t.Fatalf("unmapped: %s", got)
	}
}

func TestNormaliseResourceID(t *testing.T) {
	cases := []struct{ svc, id, want string }{
		{"s3", "my-bucket", "arn:aws:s3:::my-bucket"},
		{"s3", "arn:aws:s3:::my-bucket", "arn:aws:s3:::my-bucket"},
		{"ec2", "i-123", "arn:aws:ec2:us-east-1:acct:instance/i-123"},
		{"ec2", "sg-123", "arn:aws:ec2:us-east-1:acct:security-group/sg-123"},
		{"iam", "alice", "arn:aws:iam::acct:user/alice"},
	}
	for _, c := range cases {
		if got := NormaliseResourceID(c.svc, c.id, "us-east-1", "acct"); got != c.want {
			t.Errorf("%s/%s => %s, want %s", c.svc, c.id, got, c.want)
		}
	}
}
