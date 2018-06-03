package main

import (
	"fmt"
	"log"
	"time"

	cache "github.com/JohannesKaufmann/dynamodb-cache"
	"github.com/JohannesKaufmann/dynamodb-cache/dynadapter"
	"github.com/JohannesKaufmann/dynamodb-cache/memadapter"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

var db *dynamodb.DynamoDB

func init() {
	sess, err := session.NewSession(&aws.Config{
		Endpoint: aws.String("http://localhost:8000"),
	})
	if err != nil {
		log.Fatal(err)
	}
	db = dynamodb.New(sess)

	in := &dynamodb.CreateTableInput{
		TableName: aws.String("Cache"),
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String("Key"),
				AttributeType: aws.String("S"),
			},
		},
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String("Key"),
				KeyType:       aws.String("HASH"),
			},
		},
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(10),
			WriteCapacityUnits: aws.Int64(10),
		},
	}
	_, err = db.CreateTable(in)
	if err != nil {
		log.Fatal(err)
	}
}

/*
- - LOCALLY - -
docker run -p 8000:8000 dwmkerr/dynamodb
create table (without ttl)

- - PRODUCTION - -
create table with
- `hash key`: "Key"
- `ttl field`: "TTL"
*/

type person struct {
	Name string
	Age  int
}

func main() {
	// - - - initialize - - - //
	c, err := cache.New(
		memadapter.New(time.Hour, false),
		dynadapter.New(db, "Cache", time.Hour*24*7),
		// the order matters: items are saved (Set) and deleted (Del)
		// from to both but retrieved (Get) from the first adapter
		// (memadapter) first. If not found it tries the next
		// adapter (dynadapter).
	)
	if err != nil {
		log.Fatal(err)
	}

	// - - - set - - - //
	john := person{
		Name: "John",
		Age:  19,
	}
	// set can be called with strings, ints, structs, maps, slices, ...
	err = c.Set("1234", john)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("set person")

	// - - - get - - - //
	var p person
	// remember to pass in a pointer
	err = c.Get("1234", &p)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("get person: %+v \n", p)

	// - - - del - - - //
	err = c.Del("123")
	if err != nil {
		log.Fatal(err)
	}
}
