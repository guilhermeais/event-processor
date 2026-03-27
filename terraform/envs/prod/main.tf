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
}