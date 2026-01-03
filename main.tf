terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = "us-east-1"
}

# --- 1. Storage (S3 & DynamoDB) ---

resource "aws_s3_bucket" "uploads" {
  bucket_prefix = "image-uploads-"
  force_destroy = true # For dev environments; remove for prod
}

resource "aws_dynamodb_table" "metadata" {
  name           = "ImageLocations"
  billing_mode   = "PAY_PER_REQUEST"
  hash_key       = "imageId"

  attribute {
    name = "imageId"
    type = "S"
  }
}

# --- 2. IAM Roles & Permissions ---

resource "aws_iam_role" "lambda_exec" {
  name = "image_processor_role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "lambda.amazonaws.com"
      }
    }]
  })
}

resource "aws_iam_policy" "lambda_policy" {
  name        = "image_processor_policy"
  description = "Allow Lambda to read S3 and write DynamoDB"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = [
          "logs:CreateLogGroup",
          "logs:CreateLogStream",
          "logs:PutLogEvents"
        ]
        Effect   = "Allow"
        Resource = "arn:aws:logs:*:*:*"
      },
      {
        Action   = ["s3:GetObject"]
        Effect   = "Allow"
        Resource = "${aws_s3_bucket.uploads.arn}/*"
      },
      {
        Action   = ["dynamodb:PutItem"]
        Effect   = "Allow"
        Resource = aws_dynamodb_table.metadata.arn
      }
    ]
  })
}

resource "aws_iam_role_policy_attachment" "attach_policy" {
  role       = aws_iam_role.lambda_exec.name
  policy_arn = aws_iam_policy.lambda_policy.arn
}

# --- 1. Web Lambda (The Monolith) ---

resource "aws_lambda_function" "web" {
  filename         = "web.zip" # Build cmd/web/main.go -> web.zip
  function_name    = "GoMapWeb"
  role             = aws_iam_role.lambda_exec.arn
  handler          = "bootstrap"
  runtime          = "provided.al2023"
  architectures    = ["arm64"]
  source_code_hash = filebase64sha256("web.zip")

  environment {
    variables = {
      TABLE_NAME  = aws_dynamodb_table.metadata.name
      BUCKET_NAME = aws_s3_bucket.uploads.id
    }
  }
}

# Web Lambda needs Scan (to show map) and S3 Put (to presign urls)
resource "aws_iam_role_policy" "web_policy" {
  role = aws_iam_role.lambda_exec.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action   = ["dynamodb:Scan", "s3:PutObject"]
        Effect   = "Allow"
        Resource = [aws_dynamodb_table.metadata.arn, "${aws_s3_bucket.uploads.arn}/*"]
      }
    ]
  })
}

# --- 2. API Gateway (Single Entry Point) ---

resource "aws_apigatewayv2_api" "http_api" {
  name          = "go-map-app"
  protocol_type = "HTTP"
}

resource "aws_apigatewayv2_stage" "default" {
  api_id      = aws_apigatewayv2_api.http_api.id
  name        = "$default"
  auto_deploy = true
}

resource "aws_apigatewayv2_integration" "web_integration" {
  api_id           = aws_apigatewayv2_api.http_api.id
  integration_type = "AWS_PROXY"
  integration_uri  = aws_lambda_function.web.invoke_arn
}

# Catch-all Route: Send everything to Go
resource "aws_apigatewayv2_route" "any" {
  api_id    = aws_apigatewayv2_api.http_api.id
  route_key = "$default" 
  target    = "integrations/${aws_apigatewayv2_integration.web_integration.id}"
}

resource "aws_lambda_permission" "api_gw_web" {
  statement_id  = "AllowAPIGatewayInvokeWeb"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.web.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.http_api.execution_arn}/*/*"
}

# --- 3. Lambda Function ---

resource "aws_lambda_function" "processor" {
  filename         = "processor.zip"
  function_name    = "ImageGeoProcessor"
  role             = aws_iam_role.lambda_exec.arn
  handler          = "bootstrap" # Must be 'bootstrap' for provided.al2023
  runtime          = "provided.al2023"
  architectures    = ["arm64"]
  source_code_hash = filebase64sha256("processor.zip")

  environment {
    variables = {
      TABLE_NAME = aws_dynamodb_table.metadata.name
    }
  }
}

# --- 4. Triggers & Wiring ---

# Allow S3 to call the Lambda
resource "aws_lambda_permission" "allow_s3" {
  statement_id  = "AllowExecutionFromS3"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.processor.function_name
  principal     = "s3.amazonaws.com"
  source_arn    = aws_s3_bucket.uploads.arn
}

# Create the trigger event
resource "aws_s3_bucket_notification" "bucket_notification" {
  bucket = aws_s3_bucket.uploads.id

  lambda_function {
    lambda_function_arn = aws_lambda_function.processor.arn
    events              = ["s3:ObjectCreated:*"]
    filter_suffix       = ".jpg" # Optional: Limit to jpgs
  }

  depends_on = [aws_lambda_permission.allow_s3]
}

# --- Outputs ---

output "bucket_name" {
  value = aws_s3_bucket.uploads.id
}
