// Package collectors runs read-only security checks and returns normalized findings.
package collectors

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"

	"github.com/sriramg-aivar/nexogent/internal/ranking"
	"github.com/sriramg-aivar/nexogent/internal/types"
)

// OwnChecks runs free, read-only AWS config checks. Nothing is modified.
type OwnChecks struct {
	cfg       aws.Config
	accountID string
	region    string
}

func NewOwnChecks(cfg aws.Config, accountID, region string) *OwnChecks {
	return &OwnChecks{cfg: cfg, accountID: accountID, region: region}
}

func (o *OwnChecks) Name() string { return "own-check" }

func (o *OwnChecks) mk(service, resource string, sev types.Severity, title, desc, rec, control string, public, reachable bool) types.Finding {
	return types.Finding{
		ID: uuid.NewString(), Source: "own-check", NativeSeverity: string(sev),
		Severity: sev, Service: service, ResourceID: resource, Region: o.region,
		AccountID: o.accountID, Title: title, Description: desc, Recommendation: rec,
		FrameworkControl: control, Factors: ranking.DefaultFactors(sev, service, public, reachable),
		FirstSeen: time.Now().UTC(),
	}
}

// Collect runs every check. Per-check errors become warnings, never fatal.
func (o *OwnChecks) Collect(ctx context.Context) ([]types.Finding, []string) {
	var findings []types.Finding
	var warns []string
	run := func(fn func(context.Context) ([]types.Finding, error), name string) {
		f, err := fn(ctx)
		if err != nil {
			warns = append(warns, fmt.Sprintf("own-check %s: %v", name, err))
			return
		}
		findings = append(findings, f...)
	}
	run(o.checkIAM, "iam")
	run(o.checkS3, "s3")
	run(o.checkSGs, "ec2")
	run(o.checkEKS, "eks")
	return findings, warns
}

func (o *OwnChecks) checkIAM(ctx context.Context) ([]types.Finding, error) {
	c := iam.NewFromConfig(o.cfg)
	out, err := c.ListUsers(ctx, &iam.ListUsersInput{})
	if err != nil {
		return nil, err
	}
	var fs []types.Finding
	for _, u := range out.Users {
		name := aws.ToString(u.UserName)
		mfa, err := c.ListMFADevices(ctx, &iam.ListMFADevicesInput{UserName: u.UserName})
		if err == nil && len(mfa.MFADevices) == 0 {
			fs = append(fs, o.mk("iam", aws.ToString(u.Arn), types.SevHigh,
				fmt.Sprintf("IAM user '%s' has no MFA", name),
				"Console/CLI users without MFA are a common breach vector.",
				"Enable an MFA device for this user.", "CIS 1.10", false, true))
		}
		pols, err := c.ListAttachedUserPolicies(ctx, &iam.ListAttachedUserPoliciesInput{UserName: u.UserName})
		if err == nil {
			for _, p := range pols.AttachedPolicies {
				if aws.ToString(p.PolicyName) == "AdministratorAccess" {
					fs = append(fs, o.mk("iam", aws.ToString(u.Arn), types.SevMedium,
						fmt.Sprintf("IAM user '%s' has AdministratorAccess", name),
						"Direct admin on a user violates least privilege.",
						"Prefer roles and scoped policies over user-attached admin.", "CIS 1.16", false, true))
				}
			}
		}
	}
	return fs, nil
}

func (o *OwnChecks) checkS3(ctx context.Context) ([]types.Finding, error) {
	c := s3.NewFromConfig(o.cfg)
	out, err := c.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, err
	}
	var fs []types.Finding
	for _, b := range out.Buckets {
		name := aws.ToString(b.Name)
		arn := "arn:aws:s3:::" + name
		pab, err := c.GetPublicAccessBlock(ctx, &s3.GetPublicAccessBlockInput{Bucket: b.Name})
		blocked := false
		if err == nil && pab.PublicAccessBlockConfiguration != nil {
			cfg := pab.PublicAccessBlockConfiguration
			blocked = aws.ToBool(cfg.BlockPublicAcls) && aws.ToBool(cfg.IgnorePublicAcls) &&
				aws.ToBool(cfg.BlockPublicPolicy) && aws.ToBool(cfg.RestrictPublicBuckets)
		}
		if !blocked {
			fs = append(fs, o.mk("s3", arn, types.SevHigh,
				fmt.Sprintf("S3 bucket '%s' is not fully blocking public access", name),
				"Buckets without full public-access-block can be exposed.",
				"Enable all four Block Public Access settings.", "AWS FSBP S3.1", true, true))
		}
		if _, err := c.GetBucketEncryption(ctx, &s3.GetBucketEncryptionInput{Bucket: b.Name}); err != nil {
			fs = append(fs, o.mk("s3", arn, types.SevMedium,
				fmt.Sprintf("S3 bucket '%s' has no default encryption", name),
				"Objects may be stored unencrypted at rest.",
				"Enable default SSE-S3 or SSE-KMS encryption.", "AWS FSBP S3.4", false, false))
		}
	}
	return fs, nil
}

func (o *OwnChecks) checkSGs(ctx context.Context) ([]types.Finding, error) {
	c := ec2.NewFromConfig(o.cfg)
	out, err := c.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{})
	if err != nil {
		return nil, err
	}
	var fs []types.Finding
	for _, sg := range out.SecurityGroups {
		for _, perm := range sg.IpPermissions {
			for _, rng := range perm.IpRanges {
				if aws.ToString(rng.CidrIp) == "0.0.0.0/0" {
					from := int32(-1)
					if perm.FromPort != nil {
						from = *perm.FromPort
					}
					sev := types.SevHigh
					if from == 22 || from == 3389 {
						sev = types.SevCritical
					}
					arn := fmt.Sprintf("arn:aws:ec2:%s:%s:security-group/%s", o.region, o.accountID, aws.ToString(sg.GroupId))
					fs = append(fs, o.mk("ec2", arn, sev,
						fmt.Sprintf("Security group %s allows 0.0.0.0/0 on port %d", aws.ToString(sg.GroupId), from),
						"Open-to-the-world ingress exposes the service to the internet.",
						"Restrict the CIDR to known IPs or use SSM/VPN.", "AWS FSBP EC2.2", true, true))
				}
			}
		}
	}
	return fs, nil
}

func (o *OwnChecks) checkEKS(ctx context.Context) ([]types.Finding, error) {
	c := eks.NewFromConfig(o.cfg)
	out, err := c.ListClusters(ctx, &eks.ListClustersInput{})
	if err != nil {
		return nil, err
	}
	var fs []types.Finding
	for _, name := range out.Clusters {
		dc, err := c.DescribeCluster(ctx, &eks.DescribeClusterInput{Name: aws.String(name)})
		if err != nil || dc.Cluster == nil {
			continue
		}
		vpc := dc.Cluster.ResourcesVpcConfig
		arn := aws.ToString(dc.Cluster.Arn)
		if vpc != nil && vpc.EndpointPublicAccess {
			for _, cidr := range vpc.PublicAccessCidrs {
				if cidr == "0.0.0.0/0" {
					fs = append(fs, o.mk("eks", arn, types.SevHigh,
						fmt.Sprintf("EKS cluster '%s' API endpoint is public to 0.0.0.0/0", name),
						"A world-reachable Kubernetes API increases attack surface.",
						"Restrict publicAccessCidrs or use private endpoint access.", "CIS EKS 5.4.2", true, true))
				}
			}
		}
	}
	return fs, nil
}
