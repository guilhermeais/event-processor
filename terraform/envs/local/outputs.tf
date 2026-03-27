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