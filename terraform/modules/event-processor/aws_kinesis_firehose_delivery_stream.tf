resource "aws_kinesis_firehose_delivery_stream" "event_firehose" {
  name        = local.firehose_delivery_stream_name
  destination = "extended_s3"

  kinesis_source_configuration {
    kinesis_stream_arn = aws_kinesis_stream.dynamodb_cdc_stream.arn
    role_arn           = aws_iam_role.firehose_delivery_role.arn
  }

  extended_s3_configuration {
    role_arn   = aws_iam_role.firehose_delivery_role.arn
    bucket_arn = aws_s3_bucket.datalake_raw_zone.arn

    buffering_size     = 1
    buffering_interval = 60

    prefix              = "raw/year=!{timestamp:yyyy}/month=!{timestamp:MM}/day=!{timestamp:dd}/"
    error_output_prefix = "errors/year=!{timestamp:yyyy}/month=!{timestamp:MM}/day=!{timestamp:dd}/!{firehose:error-output-type}/"
  }

  tags = local.tags
}
