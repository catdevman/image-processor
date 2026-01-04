# Project Overview: Image Processor

A serverless AWS application built with Go that extracts geolocation data from uploaded images and displays them on a map.

## 🏗 Architecture
- **Language:** Go 1.25.5
- **Infrastructure:** Terraform (AWS)
- **Runtime:** AWS Lambda (provided.al2023, arm64)
- **Frontend:** HTML/HTMX/Alpine.js (embedded in Go binary)

## 📁 Component Breakdown

### 1. Web Frontend & API (`cmd/web`)
- **Type:** Lambda (Monolith) + API Gateway
- **Routes:**
    - `GET /`: Serves the embedded `index.html`.
    - `GET /points`: Returns image locations from DynamoDB.
    - `POST /upload-url`: Generates S3 pre-signed URLs for direct browser uploads.
- **Environment Variables:** `BUCKET_NAME`, `TABLE_NAME`.

### 2. Image Processor (`cmd/processor`)
- **Type:** Event-driven Lambda
- **Trigger:** S3 `s3:ObjectCreated:*` events (filtered for `.jpg`).
- **Logic:**
    - Downloads image from S3.
    - Uses `github.com/rwcarlsen/goexif` to extract GPS Lat/Long.
    - Stores metadata (`imageId`, `s3Key`, `latitude`, `longitude`) in DynamoDB.
- **Environment Variables:** `TABLE_NAME`.

### 3. Shared Logic (`internal/processor`)
- **`Handler`**: Orchestrates the processing flow.
- **`ExifGeoParser`**: Implementation of EXIF extraction logic.
- **Interfaces**: Defines `S3API` and `DynamoDBAPI` for testability.

### 4. Storage & Infrastructure (`main.tf`)
- **S3 Bucket**: Stores uploaded images. Configured with CORS for frontend uploads.
- **DynamoDB Table**: `ImageLocations` (Hash Key: `imageId`).
- **API Gateway**: HTTP API acting as a proxy to the Web Lambda.

## 🔄 Workflow
1. **Upload:** User selects a file in the browser.
2. **Pre-sign:** Browser requests a `PUT` URL from `cmd/web`.
3. **S3 Put:** Browser uploads the file directly to S3.
4. **Trigger:** S3 triggers `cmd/processor`.
5. **Process:** Processor extracts EXIF and saves to DynamoDB.
6. **Visualize:** Frontend calls `GET /points` and displays markers on the map.

## 🛠 Local Development
- **`cmd/local`**: A CLI tool to manually trigger the processing logic on an existing S3 object for testing purposes.
- **Testing**: `internal/processor/handler_test.go` (if implemented).

## 🚀 Deployment
- Build Go binaries for `arm64` Linux.
- Zip into `web.zip` and `processor.zip`.
- Run `terraform apply`.
