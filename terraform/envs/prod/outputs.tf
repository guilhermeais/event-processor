output "schemas_table_name" {
  description = "Nome final da tabela de schemas"
  value       = module.event_processor.schemas_table_name
}

output "events_table_name" {
  description = "Nome final da tabela de eventos"
  value       = module.event_processor.events_table_name
}

output "dlq_url" {
  description = "URL da Dead Letter Queue"
  value       = module.event_processor.dlq_url
}

output "queue_url" {
  description = "URL da Queue"
  value       = module.event_processor.queue_url
}

output "lambda_name" {
  description = "Nome da função Lambda"
  value       = module.event_processor.lambda_name
}

output "event_topic_arn" {
  description = "ARN do tópico SNS de eventos"
  value       = module.event_processor.event_topic_arn
}

output "s3_raw_bucket_name" {
  description = "Nome do bucket S3 para a Raw Zone"
  value       = module.event_processor.s3_raw_bucket_name
}

output "s3_silver_bucket_name" {
  description = "Nome do bucket S3 para a Silver Zone"
  value       = module.event_processor.s3_silver_bucket_name
}

output "glue_job_name" {
  description = "Nome do job do Glue para processar o CDC"
  value       = module.event_processor.glue_job_name
}

output "glue_crawler_name" {
  description = "Nome do crawler do Glue para a Silver Zone"
  value       = module.event_processor.glue_crawler_name
}

output "glue_iam_role_name" {
  description = "Nome da role do IAM para o Glue"
  value       = module.event_processor.glue_iam_role_name
}