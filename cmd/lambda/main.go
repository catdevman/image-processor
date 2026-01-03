package main

import (
	"context"
	"log"
	"os"

	// Import your internal package using your module name
	"github.com/catdevman/image-processor/internal/processor"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func main() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	h := &processor.Handler{
		S3Client:  s3.NewFromConfig(cfg),
		DBClient:  dynamodb.NewFromConfig(cfg),
		GeoParser: &processor.ExifGeoParser{},
		TableName: os.Getenv("TABLE_NAME"),
	}

	lambda.Start(h.Invoke)
}
