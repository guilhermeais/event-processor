resource "aws_kinesis_stream" "dynamodb_cdc_stream" {
  name             = local.aws_kinesis_stream_name
  shard_count      = var.kinesis_shard_count
  retention_period = var.kinesis_retention_period

  tags = local.tags
}

resource "aws_dynamodb_kinesis_streaming_destination" "cdc_export" {
  stream_arn = aws_kinesis_stream.dynamodb_cdc_stream.arn
  table_name = local.events_table_name
}