package store

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/sriramg-aivar/nexogent/internal/types"
)

// Dynamo persists reviews to a single DynamoDB table.
//
// Item schema (string attributes):
//
//	pk        = "REVIEW"             (constant partition for listing)
//	sk        = "<created_at_rfc3339nano>#<review_id>"  (sort, newest last)
//	review_id = <uuid>
//	body      = <json of the PostureReview>
//
// Listing recent reviews = Query pk="REVIEW" ScanIndexForward=false.
// Get-by-id uses a secondary item pk="REVIEW#<id>".
type Dynamo struct {
	c     *dynamodb.Client
	table string
}

const reviewPK = "REVIEW"

func NewDynamo(c *dynamodb.Client, table string) *Dynamo {
	return &Dynamo{c: c, table: table}
}

func (d *Dynamo) Kind() string { return "dynamodb:" + d.table }

func (d *Dynamo) Put(ctx context.Context, r *types.PostureReview) error {
	body, err := json.Marshal(r)
	if err != nil {
		return err
	}
	sk := r.CreatedAt.UTC().Format("2006-01-02T15:04:05.000000000Z") + "#" + r.ReviewID

	// 1) Listing item (pk=REVIEW, sk=time#id).
	listItem := map[string]ddbtypes.AttributeValue{
		"pk":        &ddbtypes.AttributeValueMemberS{Value: reviewPK},
		"sk":        &ddbtypes.AttributeValueMemberS{Value: sk},
		"review_id": &ddbtypes.AttributeValueMemberS{Value: r.ReviewID},
		"body":      &ddbtypes.AttributeValueMemberS{Value: string(body)},
	}
	if _, err := d.c.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(d.table), Item: listItem,
	}); err != nil {
		return fmt.Errorf("put list item: %w", err)
	}

	// 2) Direct-get item (pk=REVIEW#id, sk="-").
	getItem := map[string]ddbtypes.AttributeValue{
		"pk":   &ddbtypes.AttributeValueMemberS{Value: reviewPK + "#" + r.ReviewID},
		"sk":   &ddbtypes.AttributeValueMemberS{Value: "-"},
		"body": &ddbtypes.AttributeValueMemberS{Value: string(body)},
	}
	if _, err := d.c.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(d.table), Item: getItem,
	}); err != nil {
		return fmt.Errorf("put get item: %w", err)
	}
	return nil
}

func (d *Dynamo) Get(ctx context.Context, id string) (*types.PostureReview, error) {
	out, err := d.c.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(d.table),
		Key: map[string]ddbtypes.AttributeValue{
			"pk": &ddbtypes.AttributeValueMemberS{Value: reviewPK + "#" + id},
			"sk": &ddbtypes.AttributeValueMemberS{Value: "-"},
		},
	})
	if err != nil {
		return nil, err
	}
	if out.Item == nil {
		return nil, nil
	}
	return decodeBody(out.Item)
}

func (d *Dynamo) Latest(ctx context.Context) (*types.PostureReview, error) {
	rs, err := d.List(ctx, 1)
	if err != nil || len(rs) == 0 {
		return nil, err
	}
	return rs[0], nil
}

func (d *Dynamo) List(ctx context.Context, limit int) ([]*types.PostureReview, error) {
	var lim *int32
	if limit > 0 {
		l := int32(limit)
		lim = &l
	}
	out, err := d.c.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(d.table),
		KeyConditionExpression: aws.String("pk = :pk"),
		ExpressionAttributeValues: map[string]ddbtypes.AttributeValue{
			":pk": &ddbtypes.AttributeValueMemberS{Value: reviewPK},
		},
		ScanIndexForward: aws.Bool(false), // newest first
		Limit:            lim,
	})
	if err != nil {
		return nil, err
	}
	var reviews []*types.PostureReview
	for _, it := range out.Items {
		r, err := decodeBody(it)
		if err != nil || r == nil {
			continue
		}
		reviews = append(reviews, r)
	}
	sort.SliceStable(reviews, func(i, j int) bool {
		return reviews[i].CreatedAt.After(reviews[j].CreatedAt)
	})
	return reviews, nil
}

func decodeBody(item map[string]ddbtypes.AttributeValue) (*types.PostureReview, error) {
	bodyAttr, ok := item["body"].(*ddbtypes.AttributeValueMemberS)
	if !ok {
		return nil, nil
	}
	var r types.PostureReview
	if err := json.Unmarshal([]byte(bodyAttr.Value), &r); err != nil {
		return nil, err
	}
	return &r, nil
}
