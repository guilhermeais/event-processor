locals {
  schema_table_name = "${var.project_name}-${var.environment}-${var.schema_table_name}"
  events_table_name = "${var.project_name}-${var.environment}-${var.events_table_name}"
  event_queue_name  = "${var.project_name}-${var.environment}-${var.event_queue_name}"
  event_dlq_name    = "${var.project_name}-${var.environment}-${var.event_dlq_name}"
  lambda_name       = "${var.project_name}-${var.environment}-lambda"
  role_name         = "${var.project_name}-${var.environment}-lambda-role"

  tags = merge(
    var.tags,
    {
      Environment = var.environment
      Project     = var.project_name
    }
  )
}