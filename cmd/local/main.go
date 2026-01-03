package main

import (
	"context"
	"flag"
	"log"

	"github.com/catdevman/image-processor/internal/processor"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func main() {
	bucket := flag.String("bucket", "", "S3 Bucket Name")
	key := flag.String("key", "", "S3 Object Key")
	table := flag.String("table", "ImageLocations", "DynamoDB Table")
	flag.Parse()

	if *bucket == "" || *key == "" {
		log.Fatal("Usage: go run cmd/local/main.go -bucket <bucket> -key <key>")
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-east-1"))
	if err != nil {
		log.Fatal(err)
	}

	h := &processor.Handler{
		S3Client:  s3.NewFromConfig(cfg),
		DBClient:  dynamodb.NewFromConfig(cfg),
		GeoParser: &processor.ExifGeoParser{},
		TableName: *table,
	}

	event := events.S3Event{
		Records: []events.S3EventRecord{
			{
				S3: events.S3Entity{
					Bucket: events.S3Bucket{Name: *bucket},
					Object: events.S3Object{URLDecodedKey: *key},
				},
			},
		},
	}

	if err := h.Invoke(context.Background(), event); err != nil {
		log.Fatalf("Error: %v", err)
	}
	log.Println("Success")
}
