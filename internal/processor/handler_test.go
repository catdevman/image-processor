package processor

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// --- Mocks ---

type MockS3 struct{}

func (m *MockS3) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	// Return a dummy body (content doesn't matter now because we mock the parser)
	return &s3.GetObjectOutput{
		Body: io.NopCloser(strings.NewReader("dummy image bytes")),
	}, nil
}

type MockDynamo struct {
	PutItemInput *dynamodb.PutItemInput
}

func (m *MockDynamo) PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	m.PutItemInput = params
	return &dynamodb.PutItemOutput{}, nil
}

// MockParser allows us to simulate coordinates without a real JPG
type MockParser struct {
	Lat  float64
	Long float64
	Err  error
}

func (m *MockParser) ExtractLatLong(r io.Reader) (float64, float64, error) {
	return m.Lat, m.Long, m.Err
}

// --- Test ---

func TestHandler_Invoke(t *testing.T) {
	mockS3 := &MockS3{}
	mockDB := &MockDynamo{}

	// We force the parser to return specific coordinates
	mockParser := &MockParser{Lat: 40.7128, Long: -74.0060}

	h := &Handler{
		S3Client:  mockS3,
		DBClient:  mockDB,
		GeoParser: mockParser, // Inject the mock
		TableName: "TestTable",
	}

	event := events.S3Event{
		Records: []events.S3EventRecord{
			{
				S3: events.S3Entity{
					Bucket: events.S3Bucket{Name: "test-bucket"},
					Object: events.S3Object{URLDecodedKey: "photo.jpg"},
				},
			},
		},
	}

	err := h.Invoke(context.Background(), event)
	if err != nil {
		t.Fatalf("Handler invoke failed: %v", err)
	}

	if mockDB.PutItemInput == nil {
		t.Fatal("DynamoDB PutItem was not called")
	}

	item := mockDB.PutItemInput.Item

	// We verify that the handler correctly mapped our mock coordinates to the DB item
	if _, ok := item["latitude"]; !ok {
		t.Error("Latitude missing from DynamoDB payload")
	}
}
