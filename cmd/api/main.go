package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var (
	s3Client  *s3.Client
	dbClient  *dynamodb.Client
	bucket    string
	tableName string
)

type ImagePoint struct {
	ImageID   string  `json:"imageId" dynamodbav:"imageId"`
	Latitude  float64 `json:"latitude" dynamodbav:"latitude"`
	Longitude float64 `json:"longitude" dynamodbav:"longitude"`
	S3Key     string  `json:"s3Key" dynamodbav:"s3Key"`
}

func init() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
	}
	s3Client = s3.NewFromConfig(cfg)
	dbClient = dynamodb.NewFromConfig(cfg)
	bucket = os.Getenv("BUCKET_NAME")
	tableName = os.Getenv("TABLE_NAME")
}

func handler(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayProxyResponse, error) {
	// Enable CORS for localhost and your production domain
	headers := map[string]string{
		"Content-Type":                 "application/json",
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Methods": "GET, POST, OPTIONS",
	}

	if req.RequestContext.HTTP.Method == "OPTIONS" {
		return events.APIGatewayProxyResponse{StatusCode: 200, Headers: headers}, nil
	}

	switch req.RequestContext.HTTP.Method + " " + req.RequestContext.HTTP.Path {

	// 1. Get Upload URL
	case "POST /upload-url":
		type Request struct {
			Filename string `json:"filename"`
		}
		var body Request
		json.Unmarshal([]byte(req.Body), &body)

		key := fmt.Sprintf("%d-%s", time.Now().Unix(), body.Filename)
		presignClient := s3.NewPresignClient(s3Client)

		req, _ := presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		}, s3.WithPresignExpires(15*time.Minute))

		return jsonResponse(200, map[string]string{
			"uploadUrl": req.URL,
			"key":       key,
		}, headers), nil

	// 2. Get Map Points
	case "GET /images":
		out, err := dbClient.Scan(ctx, &dynamodb.ScanInput{
			TableName: aws.String(tableName),
		})
		if err != nil {
			return jsonResponse(500, map[string]string{"error": err.Error()}, headers), nil
		}

		var points []ImagePoint
		attributevalue.UnmarshalListOfMaps(out.Items, &points)

		return jsonResponse(200, points, headers), nil
	}

	return jsonResponse(404, map[string]string{"error": "Not Found"}, headers), nil
}

func jsonResponse(status int, body interface{}, headers map[string]string) events.APIGatewayProxyResponse {
	b, _ := json.Marshal(body)
	return events.APIGatewayProxyResponse{
		StatusCode: status,
		Headers:    headers,
		Body:       string(b),
	}
}

func main() {
	lambda.Start(handler)
}
