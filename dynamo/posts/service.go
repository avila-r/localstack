package posts

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var ctx = context.TODO()

type PostService interface {
	Insert(post Post) (string, error)
	Find(id string) (*Post, error)
}

var _ PostService = (*DynamoPostService)(nil)

type DynamoPostService struct {
	client *dynamodb.Client
	table  string
}

func Service(client *dynamodb.Client, table ...string) *DynamoPostService {
	name := "posts"
	if len(table) > 0 {
		name = table[0]
	}

	return &DynamoPostService{
		client: client,
		table:  name,
	}
}

func (s *DynamoPostService) Insert(post Post) (string, error) {
	value, err := attributevalue.MarshalMap(post)
	if err != nil {
		return "", err
	}

	input := &dynamodb.PutItemInput{
		TableName: &s.table,
		Item:      value,
	}

	if _, err := s.client.PutItem(ctx, input); err != nil {
		return "", err
	}

	return post.Id, nil
}

func (s *DynamoPostService) Find(id string) (*Post, error) {
	input := &dynamodb.GetItemInput{
		TableName: &s.table,
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	}

	res, err := s.client.GetItem(ctx, input)
	if err != nil {
		return nil, err
	}

	if len(res.Item) == 0 {
		return nil, fmt.Errorf("record with ID %v is not found", id)
	}

	post := Post{}
	if err := attributevalue.UnmarshalMap(res.Item, &post); err != nil {
		return nil, err
	}

	return &post, nil
}
