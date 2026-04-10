locals {
  schema_table_name = "${var.project_name}-${var.environment}-${var.schema_table_name}"
  events_table_name = "${var.project_name}-${var.environment}-${var.events_table_name}"
  event_topic_name  = "${var.project_name}-${var.environment}-${var.event_topic_name}"
  event_queue_name  = "${var.project_name}-${var.environment}-${var.event_queue_name}"
  event_dlq_name    = "${var.project_name}-${var.environment}-${var.event_dlq_name}"
  lambda_name       = "${var.project_name}-${var.environment}-lambda"
  role_name         = "${var.project_name}-${var.environment}-lambda-role"
  data_lake_raw_bucket_name = "${var.project_name}-${var.environment}-datalake-raw-zone"
  data_lake_silver_bucket_name = "${var.project_name}-${var.environment}-datalake-silver-zone"
  aws_kinesis_stream_name = "${var.project_name}-${var.environment}-dynamodb-cdc-stream"
  firehose_delivery_stream_name = "${var.project_name}-${var.environment}-event-firehose"
  firehose_role_name = "${var.project_name}-${var.environment}-firehose-delivery-role"
  firehose_policy_name = "${var.project_name}-${var.environment}-firehose-delivery-policy"
  glue_catalog_database_name = "${var.project_name}-${var.environment}_datalake_catalog_db"
  glue_job_name = "${var.project_name}-${var.environment}-process-dynamo-cdc-job"
  glue_crawler_name = "${var.project_name}-${var.environment}-silver-layer-crawler"
  
  tags = merge(
    var.tags,
    {
      Environment = var.environment
      Project     = var.project_name
    }
  )
}