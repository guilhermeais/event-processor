resource "aws_s3_bucket" "datalake_raw_zone" {
  bucket = local.data_lake_raw_bucket_name

  tags = {
    Name        = local.data_lake_raw_bucket_name
    Environment = var.environment
  }
}

resource "aws_s3_bucket_versioning" "datalake_raw_versioning" {
  bucket = aws_s3_bucket.datalake_raw_zone.id
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_lifecycle_configuration" "datalake_raw_zone" {
  bucket = aws_s3_bucket.datalake_raw_zone.id
  rule {
    id     = "expire_raw_data"
    status = "Enabled"
    filter {
        prefix = "raw/"     
    }
    expiration {
      days = var.s3_datalake_raw_zone_expiration_days
    }
  }
}

resource "aws_s3_bucket" "datalake_silver" {
  bucket = local.data_lake_silver_bucket_name
}

resource "aws_s3_object" "glue_script_upload" {
  bucket = aws_s3_bucket.datalake_raw_zone.id
  key    = "scripts/process_dynamo_cdc.py"
  source = "../../../data/glue/jobs/process_dynamo_cdc.py"
  etag   = filemd5("../../../data/glue/jobs/process_dynamo_cdc.py")
}