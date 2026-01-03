package main

import (
	"context"
	_ "embed"
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

//go:embed views/index.html
var indexHTML string

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
}

func init() {
	// Initialize AWS clients once
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
	method := req.RequestContext.HTTP.Method
	path := req.RequestContext.HTTP.Path

	// 1. Serve HTML
	if method == "GET" && (path == "/" || path == "") {
		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Headers:    map[string]string{"Content-Type": "text/html"},
			Body:       indexHTML,
		}, nil
	}

	// 2. Get Map Points (Called by HTMX)
	if method == "GET" && path == "/points" {
		out, err := dbClient.Scan(ctx, &dynamodb.ScanInput{
			TableName: aws.String(tableName),
		})
		if err != nil {
			return jsonResponse(500, map[string]string{"error": err.Error()}), nil
		}

		var points []ImagePoint
		attributevalue.UnmarshalListOfMaps(out.Items, &points)
		return jsonResponse(200, points), nil
	}

	// 3. Generate Upload URL (Called by AlpineJS)
	if method == "POST" && path == "/upload-url" {
		type Request struct {
			Filename string `json:"filename"`
		}
		var body Request
		json.Unmarshal([]byte(req.Body), &body)

		key := fmt.Sprintf("%d-%s", time.Now().Unix(), body.Filename)
		presignClient := s3.NewPresignClient(s3Client)

		presignedReq, _ := presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		}, s3.WithPresignExpires(15*time.Minute))

		return jsonResponse(200, map[string]string{
			"uploadUrl": presignedReq.URL,
			"key":       key,
		}), nil
	}

	return jsonResponse(404, map[string]string{"error": "Not Found"}), nil
}

func jsonResponse(status int, body interface{}) events.APIGatewayProxyResponse {
	b, _ := json.Marshal(body)
	return events.APIGatewayProxyResponse{
		StatusCode: status,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       string(b),
	}
}

func main() {
	lambda.Start(handler)
}
