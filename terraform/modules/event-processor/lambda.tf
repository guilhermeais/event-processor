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
