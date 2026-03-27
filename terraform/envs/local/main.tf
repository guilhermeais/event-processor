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
      sqs      = var.localstack_endpoint
      iam      = var.localstack_endpoint
    }
  }
}

module "event_processor" {
  source = "../../modules/event-processor"
}