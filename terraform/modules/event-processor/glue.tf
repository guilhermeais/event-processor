resource "aws_glue_catalog_database" "datalake_db" {
  name = local.glue_catalog_database_name
}
resource "aws_glue_job" "process_cdc_job" {
  name     = local.glue_job_name
  role_arn = aws_iam_role.glue_service_role.arn
  glue_version = "4.0" # Spark 3.3 / Python 3
  worker_type  = "G.1X"
  number_of_workers = 2

  command {
    script_location = "s3://${aws_s3_bucket.datalake_raw_zone.bucket}/${aws_s3_object.glue_script_upload.key}"
    python_version  = "3"
  }

  default_arguments = {
    "--S3_RAW_PATH"    = "s3://${aws_s3_bucket.datalake_raw_zone.bucket}/raw/"
    "--S3_SILVER_PATH" = "s3://${aws_s3_bucket.datalake_silver.bucket}/silver/"
    "--job-bookmark-option"              = "job-bookmark-enable"
    "--enable-continuous-cloudwatch-log" = "true"
    "--enable-continuous-log-filter"     = "true"
    "--continuous-log-logGroup"          = "/aws-glue/jobs/${local.glue_job_name}"
  }
}
resource "aws_glue_crawler" "silver_crawler" {
  database_name = aws_glue_catalog_database.datalake_db.name
  name          = local.glue_crawler_name
  role          = aws_iam_role.glue_service_role.arn

  s3_target {
    path = "s3://${aws_s3_bucket.datalake_silver.bucket}/silver/"
  }

  configuration = jsonencode({
    Version = 1.0
    CrawlerOutput = {
      Partitions = { AddOrUpdateBehavior = "InheritFromTable" }
      Tables = { TableThreshold = 1 }
    }
  })
}