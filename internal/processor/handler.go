package processor

import (
	"context"
	"io"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/rwcarlsen/goexif/exif"
)

// --- Interfaces ---

type S3API interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

type DynamoDBAPI interface {
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
}

type GeoParser interface {
	ExtractLatLong(r io.Reader) (float64, float64, error)
}

// --- Implementation ---

type ExifGeoParser struct{}

func (p *ExifGeoParser) ExtractLatLong(r io.Reader) (float64, float64, error) {
	x, err := exif.Decode(r)
	if err != nil {
		return 0, 0, err
	}
	return x.LatLong()
}

// --- Handler ---

type Handler struct {
	S3Client  S3API
	DBClient  DynamoDBAPI
	GeoParser GeoParser
	TableName string
}

func (h *Handler) Invoke(ctx context.Context, s3Event events.S3Event) error {
	for _, record := range s3Event.Records {
		bucket := record.S3.Bucket.Name
		key := record.S3.Object.URLDecodedKey

		resp, err := h.S3Client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
		if err != nil {
			log.Printf("S3 Get Error: %v", err)
			continue
		}
		defer resp.Body.Close()

		lat, long, err := h.GeoParser.ExtractLatLong(resp.Body)
		if err != nil {
			log.Printf("Geo Extract Error for %s: %v", key, err)
			continue
		}

		item, _ := attributevalue.MarshalMap(map[string]interface{}{
			"imageId":   key,
			"s3Key":     key,
			"latitude":  lat,
			"longitude": long,
		})

		_, err = h.DBClient.PutItem(ctx, &dynamodb.PutItemInput{
			TableName: aws.String(h.TableName),
			Item:      item,
		})
		if err != nil {
			log.Printf("DynamoDB Error: %v", err)
			continue
		}
	}
	return nil
}
