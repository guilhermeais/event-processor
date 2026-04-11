provider "aws" {
  region     = var.aws_region
  access_key = var.use_localstack ? var.aws_access_key_id : null
  secret_key = var.use_localstack ? var.aws_secret_access_key : null

  s3_use_path_style           = var.use_localstack
  skip_credentials_validation = var.use_localstack
  skip_metadata_api_check     = var.use_localstack
  skip_requesting_account_id  = var.use_localstack

  dynamic "endpoints" {
    for_each = var.use_localstack ? [1] : []
    content {
      dynamodb = var.localstack_endpoint
      lambda   = var.localstack_endpoint
      sns      = var.localstack_endpoint
      sqs      = var.localstack_endpoint
      iam      = var.localstack_endpoint
    }
  }
}

module "event_processor" {
  source = "../../modules/event-processor"

  lambda_batch_size                  = var.lambda_batch_size
  lambda_max_batching_window_seconds = var.lambda_max_batching_window_seconds
  lambda_timeout_seconds             = var.lambda_timeout_seconds
  lambda_xray_tracing_mode           = var.lambda_xray_tracing_mode
}