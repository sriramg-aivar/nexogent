// Package discovery inventories AWS resources (read-only), grouped by service.
package discovery

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/sriramg-aivar/nexogent/internal/types"
)

// Discoverer inventories resources for one account/region. READ-ONLY.
type Discoverer struct {
	cfg       aws.Config
	accountID string
	region    string
}

func New(cfg aws.Config, accountID, region string) *Discoverer {
	return &Discoverer{cfg: cfg, accountID: accountID, region: region}
}

// Inventory lists resources across the supported free-tier services.
// Errors from any single service are collected as warnings, not fatal.
func (d *Discoverer) Inventory(ctx context.Context) ([]types.Resource, []string) {
	var res []types.Resource
	var warns []string

	add := func(r []types.Resource, err error, svc string) {
		if err != nil {
			warns = append(warns, svc+": "+err.Error())
			return
		}
		res = append(res, r...)
	}

	add(d.ec2(ctx))
	add(d.s3(ctx))
	add(d.iam(ctx))
	add(d.lambda(ctx))
	add(d.rds(ctx))
	add(d.dynamodb(ctx))
	add(d.eks(ctx))
	return res, warns
}

func (d *Discoverer) ec2(ctx context.Context) ([]types.Resource, error, string) {
	c := ec2.NewFromConfig(d.cfg)
	out, err := c.DescribeInstances(ctx, &ec2.DescribeInstancesInput{})
	if err != nil {
		return nil, err, "ec2"
	}
	var rs []types.Resource
	for _, r := range out.Reservations {
		for _, i := range r.Instances {
			rs = append(rs, types.Resource{
				ID: aws.ToString(i.InstanceId), Service: "ec2", Type: "instance",
				Name: aws.ToString(i.InstanceId), Region: d.region, AccountID: d.accountID,
			})
		}
	}
	return rs, nil, "ec2"
}

func (d *Discoverer) s3(ctx context.Context) ([]types.Resource, error, string) {
	c := s3.NewFromConfig(d.cfg)
	out, err := c.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, err, "s3"
	}
	var rs []types.Resource
	for _, b := range out.Buckets {
		rs = append(rs, types.Resource{
			ID: "arn:aws:s3:::" + aws.ToString(b.Name), Service: "s3", Type: "bucket",
			Name: aws.ToString(b.Name), Region: "global", AccountID: d.accountID,
		})
	}
	return rs, nil, "s3"
}

func (d *Discoverer) iam(ctx context.Context) ([]types.Resource, error, string) {
	c := iam.NewFromConfig(d.cfg)
	out, err := c.ListUsers(ctx, &iam.ListUsersInput{})
	if err != nil {
		return nil, err, "iam"
	}
	var rs []types.Resource
	for _, u := range out.Users {
		rs = append(rs, types.Resource{
			ID: aws.ToString(u.Arn), Service: "iam", Type: "user",
			Name: aws.ToString(u.UserName), Region: "global", AccountID: d.accountID,
		})
	}
	return rs, nil, "iam"
}

func (d *Discoverer) lambda(ctx context.Context) ([]types.Resource, error, string) {
	c := lambda.NewFromConfig(d.cfg)
	out, err := c.ListFunctions(ctx, &lambda.ListFunctionsInput{})
	if err != nil {
		return nil, err, "lambda"
	}
	var rs []types.Resource
	for _, f := range out.Functions {
		rs = append(rs, types.Resource{
			ID: aws.ToString(f.FunctionArn), Service: "lambda", Type: "function",
			Name: aws.ToString(f.FunctionName), Region: d.region, AccountID: d.accountID,
		})
	}
	return rs, nil, "lambda"
}

func (d *Discoverer) rds(ctx context.Context) ([]types.Resource, error, string) {
	c := rds.NewFromConfig(d.cfg)
	out, err := c.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{})
	if err != nil {
		return nil, err, "rds"
	}
	var rs []types.Resource
	for _, i := range out.DBInstances {
		rs = append(rs, types.Resource{
			ID: aws.ToString(i.DBInstanceArn), Service: "rds", Type: "db-instance",
			Name: aws.ToString(i.DBInstanceIdentifier), Region: d.region, AccountID: d.accountID,
		})
	}
	return rs, nil, "rds"
}

func (d *Discoverer) dynamodb(ctx context.Context) ([]types.Resource, error, string) {
	c := dynamodb.NewFromConfig(d.cfg)
	out, err := c.ListTables(ctx, &dynamodb.ListTablesInput{})
	if err != nil {
		return nil, err, "dynamodb"
	}
	var rs []types.Resource
	for _, name := range out.TableNames {
		rs = append(rs, types.Resource{
			ID: name, Service: "dynamodb", Type: "table",
			Name: name, Region: d.region, AccountID: d.accountID,
		})
	}
	return rs, nil, "dynamodb"
}

func (d *Discoverer) eks(ctx context.Context) ([]types.Resource, error, string) {
	c := eks.NewFromConfig(d.cfg)
	out, err := c.ListClusters(ctx, &eks.ListClustersInput{})
	if err != nil {
		return nil, err, "eks"
	}
	var rs []types.Resource
	for _, name := range out.Clusters {
		rs = append(rs, types.Resource{
			ID: name, Service: "eks", Type: "cluster",
			Name: name, Region: d.region, AccountID: d.accountID,
		})
	}
	return rs, nil, "eks"
}
