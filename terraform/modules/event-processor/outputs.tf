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