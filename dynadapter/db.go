package dynadapter

import (
	"errors"
	"time"

	cache "github.com/JohannesKaufmann/dynamodb-cache"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
)

type Adapter struct {
	client dynamodbiface.DynamoDBAPI
	table  string
	ttl    time.Duration
}

func New(client dynamodbiface.DynamoDBAPI, table string, ttl time.Duration) cache.InitAdapter {
	return func() (cache.Adapter, error) {
		if table == "" {
			return nil, errors.New("dynamodb: name of table is empty")
		}
		if ttl == 0 {
			return nil, errors.New("dynamodb: ttl needs to be above 0 (ttl active) or -1 (no ttl)")
		}

		return &Adapter{client: client, table: table, ttl: ttl}, nil
	}
}

func (a *Adapter) Get(key string) ([]byte, error) {
	i := item{Key: key}
	err := i.get(a.client, a.table)
	if err != nil {
		return nil, err
	}

	if a.ttl != -1 && time.Now().Unix() > i.TTL {
		return nil, cache.ErrExpired
	}

	return i.Data, nil
}
func (a *Adapter) Set(key string, data []byte) error {
	future := time.Now().Add(a.ttl).Unix()
	i := item{Key: key, TTL: future, Data: data}

	return i.put(a.client, a.table)
}
func (a *Adapter) Del(key string) error {
	return item{Key: key}.del(a.client, a.table)
}
