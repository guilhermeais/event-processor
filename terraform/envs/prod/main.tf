terraform {
  backend "s3" {
    bucket = "terraform-state-274020916788-us-east-1-an"
    key    = "event-processor/prod.tfstate"
    region = "us-east-1"
  }
}

provider "aws" {
  region = "us-east-1"
}

module "event_processor" {
  source = "../../modules/event-processor"
  
  environment = "prod"
  lambda_batch_size                  = var.lambda_batch_size
  lambda_max_batching_window_seconds = var.lambda_max_batching_window_seconds
  lambda_timeout_seconds             = var.lambda_timeout_seconds
  lambda_xray_tracing_mode           = var.lambda_xray_tracing_mode
}