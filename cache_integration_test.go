// +build integration

package cache_test

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	cache "github.com/JohannesKaufmann/dynamodb-cache"
	// dynamodbAdapter "github.com/JohannesKaufmann/dynamodb-cache/dynamodb"
	"github.com/JohannesKaufmann/dynamodb-cache/memory"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/guregu/dynamo"
	"github.com/ory/dockertest"
)

var tableName = "TestCache"
var port string
var sess = session.New()

var db *dynamo.DB
var client *dynamodb.DynamoDB

func init() {
	dynamo.RetryTimeout = time.Second
}

func createTable(name string) error {
	type Cache struct {
		Key string `dynamo:"Key,hash"`
	}

	c := Cache{}
	return db.CreateTable(name, c).Run()
}

func TestMain(m *testing.M) {
	// -> https://stackoverflow.com/a/42764706

	fmt.Println("\tStarting DynamoDB via Docker")

	// uses a sensible default on windows (tcp/http) and linux/osx (socket)
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	// pulls an image, creates a container based on it and runs it
	resource, err := pool.Run("dwmkerr/dynamodb", "", nil)
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	port = resource.GetPort("8000/tcp")
	db = dynamo.New(sess, &aws.Config{
		Endpoint: aws.String("http://localhost:" + port),
	})
	client = dynamodb.New(sess, &aws.Config{
		Endpoint: aws.String("http://localhost:" + port),
	})

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	if err := pool.Retry(func() error {

		tables, err := db.ListTables().All()
		if err != nil {
			return err
		}
		if len(tables) != 0 {
			panic("there are already tables in the db")
		}

		return createTable(tableName)
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("recovered:", r)
		}
	}()
	code := m.Run()

	// You can't defer this because os.Exit doesn't care for defer
	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}
	fmt.Println("PURGED")

	os.Exit(code)
}

func reportPanic(t *testing.T) {
	if r := recover(); r != nil {
		t.Error("PANIC:", r)
	}
}

func testGetSetDel(c *cache.Cache, t *testing.T) {

	// - - get 1 - - //
	var target string
	err := c.Get("1", &target)
	if err != cache.ErrNotFound {
		t.Error(err)
	}
	if target != "" {
		t.Errorf("expected '' but got '%s'", target)
	}

	// - - set - - //
	err = c.Set("1", "One")
	if err != nil {
		t.Error(err)
	}

	// - - get 2 - - //
	err = c.Get("1", &target)
	if err != nil {
		t.Error(err) // TODO:
	}
	if target != "One" {
		t.Errorf("expected 'One' but got '%s'", target)
	}

	// - - del - - //
	err = c.Del("1")
	if err != nil {
		t.Error(err)
	}

	// - - get 3 - - //
	err = c.Get("1", &target)
	if err != cache.ErrNotFound {
		t.Error(err)
	}
}

func TestMemory(t *testing.T) {
	defer reportPanic(t)

	c, err := cache.New(
		memadapter.New(-1, false),
	)
	if err != nil {
		t.Error(err)
	}
	testGetSetDel(c, t)
}

func TestDynamoDB(t *testing.T) {
	defer reportPanic(t)

	c, err := cache.New(
		dynadapter.New(client, tableName, -1),
	)
	if err != nil {
		t.Error(err)
	}

	testGetSetDel(c, t)
}

func TestDynamoDBAndMemory(t *testing.T) {
	defer reportPanic(t)

	c, err := cache.New(
		memadapter.New(-1, false),
		dynadapter.New(client, tableName, -1),
	)
	if err != nil {
		t.Error(err)
	}

	testGetSetDel(c, t)
}

func TestExpire(t *testing.T) {
	defer reportPanic(t)

	memoryTTL := time.Second       // time.Millisecond * 10
	dynamoDBTTL := time.Second * 2 // time.Millisecond * 20

	c, err := cache.New(
		memadapter.New(memoryTTL, false),
		dynadapter.New(client, tableName, dynamoDBTTL),
	)
	if err != nil {
		t.Error(err)
	}

	// - - set - - //
	err = c.Set("1", "One")
	if err != nil {
		t.Error(err)
	}

	time.Sleep(memoryTTL / 2)

	// - - get 1 - - //
	var target string
	err = c.Get("1", &target)
	if err != nil {
		t.Error(err)
	}
	if target != "One" {
		t.Errorf("expected 'One' but got '%s'", target)
	}
	target = ""

	time.Sleep(memoryTTL)
	// - - get 2 (after memory expire) - - //
	err = c.Get("1", &target)
	if err != nil {
		t.Error(err)
	}
	if target != "One" {
		t.Errorf("expected 'One' but got '%s'", target)
	}
	target = ""

	time.Sleep(dynamoDBTTL - memoryTTL)
	// - - get 2 (after dynamodb expire) - - //
	err = c.Get("1", &target)
	if err != cache.ErrExpired {
		t.Error(err)
	}
	if target != "" {
		t.Errorf("expected '' but got '%s'", target)
	}
}
