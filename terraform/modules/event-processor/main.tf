terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

resource "aws_dynamodb_table" "schemas" {
  name           = local.schema_table_name
  billing_mode   = "PAY_PER_REQUEST"
  hash_key       = "event_type"

  attribute {
    name = "event_type"
    type = "S"
  }

  tags = local.tags
}

resource "aws_dynamodb_table" "events" {
  name           = local.events_table_name
  billing_mode   = "PAY_PER_REQUEST"
  hash_key       = "client_id"
  range_key      = "event_id"

  attribute {
    name = "client_id"
    type = "S"
  }

  attribute {
    name = "event_id"
    type = "S"
  }

  tags = local.tags
}

resource "aws_sqs_queue" "event_dlq" {
  name = local.event_dlq_name

  tags = local.tags
}

resource "aws_sqs_queue" "event_queue" {
  name                       = local.event_queue_name
  visibility_timeout_seconds = var.visibility_timeout_seconds

  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.event_dlq.arn
    maxReceiveCount     = var.max_receive_count
  })

  tags = local.tags
}

resource "aws_iam_role" "lambda_exec_role" {
  name = local.role_name

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

  tags = local.tags
}

resource "aws_iam_role_policy" "lambda_app_permissions" {
  name = "event_processor_permissions"
  role = aws_iam_role.lambda_exec_role.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "SQSAccess"
        Effect = "Allow"
        Action = [
          "sqs:ReceiveMessage",
          "sqs:DeleteMessage",
          "sqs:GetQueueAttributes"
        ]
        Resource = aws_sqs_queue.event_queue.arn
      },
      {
        Sid    = "DynamoDBAccess"
        Effect = "Allow"
        Action = [
          "dynamodb:PutItem",
          "dynamodb:GetItem",
          "dynamodb:UpdateItem",
          "dynamodb:Query",
          "dynamodb:Scan"
        ]
        Resource = [
          aws_dynamodb_table.schemas.arn,
          aws_dynamodb_table.events.arn
        ]
      }
    ]
  })
}

resource "aws_iam_role_policy_attachment" "lambda_logs" {
  role       = aws_iam_role.lambda_exec_role.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

resource "aws_lambda_function" "event_processor" {
  function_name = local.lambda_name
  filename      = var.lambda_zip_path

  source_code_hash = filebase64sha256(var.lambda_zip_path)

  role    = aws_iam_role.lambda_exec_role.arn
  handler = var.lambda_handler
  runtime = var.lambda_runtime

  environment {
    variables = {
      SCHEMA_TABLE_NAME = aws_dynamodb_table.schemas.name
      EVENTS_TABLE_NAME = aws_dynamodb_table.events.name
      DLQ_URL           = aws_sqs_queue.event_dlq.url
    }
  }

  tags = local.tags
}

resource "aws_lambda_event_source_mapping" "sqs_trigger" {
  event_source_arn        = aws_sqs_queue.event_queue.arn
  function_name           = aws_lambda_function.event_processor.arn
  batch_size              = var.lambda_batch_size
  function_response_types = ["ReportBatchItemFailures"]
}