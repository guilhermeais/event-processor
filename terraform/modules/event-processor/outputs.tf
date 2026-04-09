output "event_topic_arn" {
  description = "ARN do tópico SNS de eventos"
  value       = aws_sns_topic.events.arn
}
output "schemas_table_name" {
  description = "Nome final da tabela de schemas"
  value       = aws_dynamodb_table.schemas.name
}

output "events_table_name" {
  description = "Nome final da tabela de eventos"
  value       = aws_dynamodb_table.events.name
}

output "dlq_url" {
  description = "URL da Dead Letter Queue"
  value       = aws_sqs_queue.event_dlq.url
}

output "queue_url" {
  description = "URL da Queue"
  value       = aws_sqs_queue.event_queue.url
}

output "lambda_name" {
  description = "Nome da função Lambda"
  value       = aws_lambda_function.event_processor.function_name
}

output "s3_raw_bucket_name" {
  description = "Nome do bucket S3 para a Raw Zone"
  value       = aws_s3_bucket.datalake_raw_zone.bucket
}

output "s3_silver_bucket_name" {
  description = "Nome do bucket S3 para a Silver Zone"
  value       = aws_s3_bucket.datalake_silver.bucket
}

output "glue_job_name" {
  description = "Nome do job do Glue para processar o CDC"
  value       = aws_glue_job.process_cdc_job.name
}

output "glue_crawler_name" {
  description = "Nome do crawler do Glue para a Silver Zone"
  value       = aws_glue_crawler.silver_crawler.name
}

output "glue_iam_role_name" {
  description = "Nome da role do IAM para o Glue"
  value       = aws_iam_role.glue_service_role.name
}
