package dynadapter

import (
	"github.com/JohannesKaufmann/dynamodb-cache"

	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
)

type item struct {
	Key  string
	TTL  int64  `json:",omitempty"`
	Data []byte `json:",omitempty"`
}

func (i *item) marshal() (map[string]*dynamodb.AttributeValue, error) {
	return dynamodbattribute.MarshalMap(i)
}
func (i *item) unmarshal(data map[string]*dynamodb.AttributeValue) error {
	return dynamodbattribute.UnmarshalMap(data, i)
}

func (i *item) put(client dynamodbiface.DynamoDBAPI, table string) error {
	item, err := i.marshal()
	if err != nil {
		return err
	}

	input := &dynamodb.PutItemInput{
		Item:      item,
		TableName: &table,
	}

	_, err = client.PutItem(input)
	return err
}

func (i *item) get(client dynamodbiface.DynamoDBAPI, table string) error {
	key, err := i.marshal()
	if err != nil {
		return err
	}

	input := &dynamodb.GetItemInput{
		TableName: &table,
		Key:       key,
	}

	result, err := client.GetItem(input)
	if err != nil {
		return err
	}

	if len(result.Item) == 0 {
		return cache.ErrNotFound
	}

	return i.unmarshal(result.Item)
}

func (i item) del(client dynamodbiface.DynamoDBAPI, table string) error {
	key, err := i.marshal()
	if err != nil {
		return err
	}

	input := &dynamodb.DeleteItemInput{
		TableName: &table,
		Key:       key,
	}
	_, err = client.DeleteItem(input)
	return err
}
