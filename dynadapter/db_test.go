package dynadapter

import (
	"bytes"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/vmihailenco/msgpack"

	"github.com/JohannesKaufmann/dynamodb-cache"

	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
)

type mockDynamoDBClient struct {
	dynamodbiface.DynamoDBAPI

	GetItemFunc    func(input *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error)
	PutItemFunc    func(input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error)
	DeleteItemFunc func(input *dynamodb.DeleteItemInput) (*dynamodb.DeleteItemOutput, error)
}

func (m *mockDynamoDBClient) GetItem(input *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
	return m.GetItemFunc(input)
}
func (m *mockDynamoDBClient) PutItem(input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
	return m.PutItemFunc(input)
}
func (m *mockDynamoDBClient) DeleteItem(input *dynamodb.DeleteItemInput) (*dynamodb.DeleteItemOutput, error) {
	return m.DeleteItemFunc(input)
}

func new(mock *mockDynamoDBClient, ttl time.Duration) (cache.Adapter, error) {
	return New(mock, "TestCache", ttl)()
}

func TestNew(t *testing.T) {
	_, err := New(nil, "", time.Second)()
	if err.Error() != "dynamodb: name of table is empty" {
		t.Error(err)
	}

	_, err = New(nil, "TestCache", 0)()
	if err.Error() != "dynamodb: ttl needs to be above 0 (ttl active) or -1 (no ttl)" {
		t.Error(err)
	}
}

func TestInternalGet(t *testing.T) {
	d := []byte("data")
	mockSvc := &mockDynamoDBClient{
		GetItemFunc: func(input *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{
				Item: map[string]*dynamodb.AttributeValue{
					"Key": {
						S: aws.String("1"),
					},
					"Data": {
						B: d,
					},
				},
			}, nil
		},
	}

	i := item{Key: "1"}
	err := i.get(mockSvc, "TestCache")
	if err != nil {
		t.Error(err)
	}

	if i.Key != "1" {
		t.Error("wrong key")
	}
	if !bytes.Equal(i.Data, d) {
		t.Error("got different data")
	}
}
func TestGet_NotFound(t *testing.T) {
	mockSvc := &mockDynamoDBClient{
		GetItemFunc: func(input *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{}, nil
		},
	}

	c, err := new(mockSvc, -1)
	if err != nil {
		t.Error(err)
	}
	data, err := c.Get("1")
	if err != cache.ErrNotFound {
		t.Error(err)
	}
	if data != nil {
		t.Error("expected nil but got []byte")
	}
}
func TestGet_Found(t *testing.T) {
	d, _ := msgpack.Marshal("One")

	mockSvc := &mockDynamoDBClient{
		GetItemFunc: func(input *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {

			return &dynamodb.GetItemOutput{
				Item: map[string]*dynamodb.AttributeValue{
					"Key": {
						S: aws.String("1"),
					},
					"Data": {
						B: d,
					},
				},
			}, nil
		},
	}

	c, err := new(mockSvc, -1)
	if err != nil {
		t.Error(err)
	}
	data, err := c.Get("1")
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(data, d) {
		t.Error("got different []byte")
	}
}

func TestGet_FoundButExpired(t *testing.T) {
	mockSvc := &mockDynamoDBClient{
		GetItemFunc: func(input *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
			d, _ := msgpack.Marshal("One")
			now := time.Now().Add(-time.Second).Unix()

			return &dynamodb.GetItemOutput{
				Item: map[string]*dynamodb.AttributeValue{
					"Key": {
						S: aws.String("1"),
					},
					"TTL": {
						N: aws.String(strconv.Itoa(int(now))),
					},
					"Data": {
						B: d,
					},
				},
			}, nil
		},
	}

	c, err := new(mockSvc, 1)
	if err != nil {
		t.Error(err)
	}
	data, err := c.Get("1")
	if err != cache.ErrExpired {
		t.Error(err)
	}
	if data != nil {
		t.Error("expected nil but got []byte")
	}
}

func TestGet_Err(t *testing.T) {
	var e = errors.New("some error")
	mockSvc := &mockDynamoDBClient{
		GetItemFunc: func(input *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
			return nil, e
		},
	}

	c, err := new(mockSvc, 1)
	if err != nil {
		t.Error(err)
	}
	data, err := c.Get("1")
	if err != e {
		t.Error(err)
	}
	if data != nil {
		t.Error("expected nil but got []byte")
	}
}
func TestSet_Success(t *testing.T) {
	mockSvc := &mockDynamoDBClient{
		PutItemFunc: func(input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
			return nil, nil
		},
	}

	c, err := new(mockSvc, 1)
	if err != nil {
		t.Error(err)
	}
	err = c.Set("1", []byte("One"))
	if err != nil {
		t.Error(err)
	}
}
func TestSet_Err(t *testing.T) {
	var e = errors.New("some error")
	mockSvc := &mockDynamoDBClient{
		PutItemFunc: func(input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
			return nil, e
		},
	}

	c, err := new(mockSvc, 1)
	if err != nil {
		t.Error(err)
	}
	err = c.Set("1", []byte("One"))
	if err != e {
		t.Error(err)
	}
}

func TestDel_Err(t *testing.T) {
	var e = errors.New("some error")
	mockSvc := &mockDynamoDBClient{
		DeleteItemFunc: func(*dynamodb.DeleteItemInput) (*dynamodb.DeleteItemOutput, error) {
			return nil, e
		},
	}

	c, err := new(mockSvc, 1)
	if err != nil {
		t.Error(err)
	}
	err = c.Del("1")
	if err != e {
		t.Error(err)
	}
}
