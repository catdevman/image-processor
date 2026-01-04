# Image Processor

A robust serverless application built with **Go** and **AWS** that extracts geolocation metadata from images and visualizes them on an interactive map.

![Architecture Diagram](https://img.shields.io/badge/Architecture-Serverless-orange?style=flat-square)
![Go Version](https://img.shields.io/badge/Go-1.25.5-blue?style=flat-square&logo=go)
![Infrastructure](https://img.shields.io/badge/IaC-Terraform-purple?style=flat-square&logo=terraform)

## 📖 Overview

This project demonstrates a cloud-native, event-driven architecture for processing image uploads. When a user uploads a photo (specifically JPGs) via the web interface, the application automatically:
1.  **Stores** the image securely in AWS S3.
2.  **Triggers** a Lambda function to analyze the file.
3.  **Extracts** EXIF GPS coordinates (latitude/longitude) using Go.
4.  **Indexes** the metadata in Amazon DynamoDB.
5.  **Displays** the location on a map via the web dashboard.

## 🏗 Architecture

The system is composed of several decoupled components:

*   **Web Frontend (Lambda Monolith):**
    *   Located in `cmd/web`.
    *   Serves the HTML/HTMX/Alpine.js UI.
    *   Generates S3 Pre-signed URLs for secure, direct-to-S3 uploads.
    *   Fetches processed location data from DynamoDB for the map display.
    *   Running behind an AWS HTTP API Gateway.

*   **Image Processor (Event Handler):**
    *   Located in `cmd/processor`.
    *   Triggered purely by `s3:ObjectCreated:*` events.
    *   Downloads the new image, parses EXIF data using `github.com/rwcarlsen/goexif`, and saves the result to DynamoDB.

*   **Infrastructure:**
    *   Defined in `main.tf`.
    *   Resources: S3 Bucket (Uploads), DynamoDB Table (Metadata), API Gateway, Lambda Functions, IAM Roles.

## 📂 Project Structure

```bash
├── cmd/
│   ├── api/        # (Optional) Separate API entry points
│   ├── local/      # CLI tool for testing processing logic locally
│   ├── processor/  # The background worker Lambda (S3 Trigger)
│   └── web/        # The frontend/API Lambda (HTTP Trigger)
├── internal/
│   └── processor/  # Shared business logic and interfaces
├── main.tf         # Terraform Infrastructure as Code
├── go.mod          # Go dependencies
└── PROJECT_OVERVIEW.md # Detailed architecture notes
```

## 🚀 Getting Started

### Prerequisites

- **Go**: Version 1.25.5 or later.
- **Terraform**: For provisioning AWS resources.
- **AWS CLI**: Configured with appropriate credentials.

### Build & Deploy

1.  **Prepare the Binaries:**
    AWS Lambda (`provided.al2023`) requires a binary named `bootstrap`. You need to build and zip both the web and processor services.

    ```bash
    # Build Web
    GOOS=linux GOARCH=arm64 go build -o bootstrap cmd/web/main.go
    zip web.zip bootstrap
    rm bootstrap

    # Build Processor
    GOOS=linux GOARCH=arm64 go build -o bootstrap cmd/processor/main.go
    zip processor.zip bootstrap
    rm bootstrap
    ```

2.  **Initialize Terraform:**
    *Note: You may need to adjust the `backend "s3"` configuration in `main.tf` to match your own state bucket or remove it for local state.*

    ```bash
    terraform init
    ```

3.  **Deploy Infrastructure:**

    ```bash
    terraform apply
    ```
    Confirm the apply to create the resources. Terraform will output the `bucket_name` and other details.

4.  **Access the Application:**
    Navigate to the API Gateway URL provided by the AWS Console (or add an output for it in Terraform) to view the web interface.

## 🛠 Local Development

This project uses [mise](https://mise.jdx.dev/) to manage environment variables and tasks. This ensures you have the correct Go version and AWS configuration without manual setup.

### Prerequisites
1.  **Install `mise`**: Follow the [official instructions](https://mise.jdx.dev/getting-started.html).
2.  **AWS Credentials**: Ensure your terminal session has active AWS credentials (e.g., via `aws sso login` or env vars).
3.  **Terraform State**: You must have run `terraform apply` at least once so `mise` can fetch the bucket name.

### Running the Server

Start the local development server with a single command:

```bash
make dev
```

(This runs `mise run dev` under the hood)

This will:
1.  Automatically fetch the `BUCKET_NAME` from your Terraform outputs.
2.  Set necessary environment variables (`TABLE_NAME`, `AWS_REGION`).
3.  Start the Go web server on `http://localhost:8080`.

You can now visit `http://localhost:8080` to upload images and see them placed on the map.

## 🧪 Testing

Run the standard Go test suite:

```bash
go test ./internal/...
```
