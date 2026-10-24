// Package discovery inventories AWS resources (read-only), grouped by service,
// across all enabled regions. Nothing is modified.
package discovery

import (
	"context"
	"sort"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/sriramg-aivar/nexogent/internal/types"
)

// Discoverer inventories resources for one account across regions. READ-ONLY.
type Discoverer struct {
	cfg       aws.Config
	accountID string
	region    string // the "home" region for global services / fallbacks
}

func New(cfg aws.Config, accountID, region string) *Discoverer {
	return &Discoverer{cfg: cfg, accountID: accountID, region: region}
}

// EnabledRegions returns every region enabled for the account (read-only).
// Falls back to the home region on error so discovery still works.
func (d *Discoverer) EnabledRegions(ctx context.Context) ([]string, error) {
	c := ec2.NewFromConfig(d.cfg)
	out, err := c.DescribeRegions(ctx, &ec2.DescribeRegionsInput{
		AllRegions: aws.Bool(false), // only regions enabled for this account
	})
	if err != nil {
		return []string{d.region}, err
	}
	var regions []string
	for _, r := range out.Regions {
		regions = append(regions, aws.ToString(r.RegionName))
	}
	if len(regions) == 0 {
		regions = []string{d.region}
	}
	sort.Strings(regions)
	return regions, nil
}

// cfgForRegion clones the base config pinned to a specific region.
func (d *Discoverer) cfgForRegion(ctx context.Context, region string) aws.Config {
	c, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		// Fall back to copying the base cfg with the region overridden.
		cp := d.cfg.Copy()
		cp.Region = region
		return cp
	}
	return c
}

// Inventory lists resources across ALL enabled regions. Global services
// (IAM, S3 listing) are collected once. Per-service, per-region errors become
// warnings, never fatal.
func (d *Discoverer) Inventory(ctx context.Context) ([]types.Resource, []string) {
	regions, rerr := d.EnabledRegions(ctx)

	var (
		mu    sync.Mutex
		res   []types.Resource
		warns []string
	)
	addRes := func(rs []types.Resource) {
		mu.Lock()
		res = append(res, rs...)
		mu.Unlock()
	}
	addWarn := func(w string) {
		mu.Lock()
		warns = append(warns, w)
		mu.Unlock()
	}
	if rerr != nil {
		addWarn("regions: " + rerr.Error() + " (using home region only)")
	}

	// ---- Global services (once) ----
	if rs, err := d.iam(ctx, d.cfg); err != nil {
		addWarn("iam: " + err.Error())
	} else {
		addRes(rs)
	}
	if rs, err := d.s3(ctx, d.cfg); err != nil {
		addWarn("s3: " + err.Error())
	} else {
		addRes(rs)
	}

	// ---- Regional services (concurrent per region) ----
	var wg sync.WaitGroup
	sem := make(chan struct{}, 6) // cap concurrency
	for _, region := range regions {
		wg.Add(1)
		go func(region string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			rcfg := d.cfgForRegion(ctx, region)
			type regFn struct {
				name string
				fn   func(context.Context, aws.Config, string) ([]types.Resource, error)
			}
			for _, f := range []regFn{
				{"ec2", d.ec2},
				{"lambda", d.lambda},
				{"rds", d.rds},
				{"dynamodb", d.dynamodb},
				{"eks", d.eks},
			} {
				rs, err := f.fn(ctx, rcfg, region)
				if err != nil {
					addWarn(f.name + " (" + region + "): " + err.Error())
					continue
				}
				addRes(rs)
			}
		}(region)
	}
	wg.Wait()

	return res, warns
}

// ---- Global services ----

func (d *Discoverer) iam(ctx context.Context, cfg aws.Config) ([]types.Resource, error) {
	c := iam.NewFromConfig(cfg)
	out, err := c.ListUsers(ctx, &iam.ListUsersInput{})
	if err != nil {
		return nil, err
	}
	var rs []types.Resource
	for _, u := range out.Users {
		rs = append(rs, types.Resource{
			ID: aws.ToString(u.Arn), Service: "iam", Type: "user",
			Name: aws.ToString(u.UserName), Region: "global", AccountID: d.accountID,
		})
	}
	return rs, nil
}

func (d *Discoverer) s3(ctx context.Context, cfg aws.Config) ([]types.Resource, error) {
	c := s3.NewFromConfig(cfg)
	out, err := c.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, err
	}
	var rs []types.Resource
	for _, b := range out.Buckets {
		name := aws.ToString(b.Name)
		region := d.bucketRegion(ctx, c, name)
		rs = append(rs, types.Resource{
			ID: "arn:aws:s3:::" + name, Service: "s3", Type: "bucket",
			Name: name, Region: region, AccountID: d.accountID,
		})
	}
	return rs, nil
}

// bucketRegion resolves a bucket's region; empty LocationConstraint == us-east-1.
func (d *Discoverer) bucketRegion(ctx context.Context, c *s3.Client, bucket string) string {
	out, err := c.GetBucketLocation(ctx, &s3.GetBucketLocationInput{Bucket: aws.String(bucket)})
	if err != nil {
		return "global"
	}
	region := string(out.LocationConstraint)
	if region == "" {
		return "us-east-1"
	}
	return region
}

// ---- Regional services ----

func (d *Discoverer) ec2(ctx context.Context, cfg aws.Config, region string) ([]types.Resource, error) {
	c := ec2.NewFromConfig(cfg)
	out, err := c.DescribeInstances(ctx, &ec2.DescribeInstancesInput{})
	if err != nil {
		return nil, err
	}
	var rs []types.Resource
	for _, r := range out.Reservations {
		for _, i := range r.Instances {
			name := aws.ToString(i.InstanceId)
			tags := map[string]string{}
			for _, t := range i.Tags {
				tags[aws.ToString(t.Key)] = aws.ToString(t.Value)
				if aws.ToString(t.Key) == "Name" {
					name = aws.ToString(t.Value)
				}
			}
			rs = append(rs, types.Resource{
				ID: aws.ToString(i.InstanceId), Service: "ec2", Type: "instance",
				Name: name, Region: region, AccountID: d.accountID, Tags: tags,
			})
		}
	}
	return rs, nil
}

func (d *Discoverer) lambda(ctx context.Context, cfg aws.Config, region string) ([]types.Resource, error) {
	c := lambda.NewFromConfig(cfg)
	out, err := c.ListFunctions(ctx, &lambda.ListFunctionsInput{})
	if err != nil {
		return nil, err
	}
	var rs []types.Resource
	for _, f := range out.Functions {
		rs = append(rs, types.Resource{
			ID: aws.ToString(f.FunctionArn), Service: "lambda", Type: "function",
			Name: aws.ToString(f.FunctionName), Region: region, AccountID: d.accountID,
		})
	}
	return rs, nil
}

func (d *Discoverer) rds(ctx context.Context, cfg aws.Config, region string) ([]types.Resource, error) {
	c := rds.NewFromConfig(cfg)
	out, err := c.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{})
	if err != nil {
		return nil, err
	}
	var rs []types.Resource
	for _, i := range out.DBInstances {
		rs = append(rs, types.Resource{
			ID: aws.ToString(i.DBInstanceArn), Service: "rds", Type: "db-instance",
			Name: aws.ToString(i.DBInstanceIdentifier), Region: region, AccountID: d.accountID,
		})
	}
	return rs, nil
}

func (d *Discoverer) dynamodb(ctx context.Context, cfg aws.Config, region string) ([]types.Resource, error) {
	c := dynamodb.NewFromConfig(cfg)
	out, err := c.ListTables(ctx, &dynamodb.ListTablesInput{})
	if err != nil {
		return nil, err
	}
	var rs []types.Resource
	for _, name := range out.TableNames {
		rs = append(rs, types.Resource{
			ID: name, Service: "dynamodb", Type: "table",
			Name: name, Region: region, AccountID: d.accountID,
		})
	}
	return rs, nil
}

func (d *Discoverer) eks(ctx context.Context, cfg aws.Config, region string) ([]types.Resource, error) {
	c := eks.NewFromConfig(cfg)
	out, err := c.ListClusters(ctx, &eks.ListClustersInput{})
	if err != nil {
		return nil, err
	}
	var rs []types.Resource
	for _, name := range out.Clusters {
		rs = append(rs, types.Resource{
			ID: name, Service: "eks", Type: "cluster",
			Name: name, Region: region, AccountID: d.accountID,
		})
	}
	return rs, nil
}
