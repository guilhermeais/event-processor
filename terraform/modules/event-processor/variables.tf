variable "environment" {
  description = "Ambiente de execução (dev, staging, prod)"
  type        = string
  default     = "dev"

  validation {
    condition     = contains(["dev", "staging", "prod"], var.environment)
    error_message = "Environment deve ser dev, staging ou prod."
  }
}

variable "project_name" {
  description = "Nome do projeto para prefixo de recursos"
  type        = string
  default     = "event-processor"
}

variable "schema_table_name" {
  description = "Nome da tabela DynamoDB de schemas"
  type        = string
  default     = "schemas"
}

variable "events_table_name" {
  description = "Nome da tabela DynamoDB de eventos"
  type        = string
  default     = "events"
}

variable "event_topic_name" {
  description = "Nome do tópico SNS de entrada de eventos"
  type        = string
  default     = "event-topic"
}

variable "event_queue_name" {
  description = "Nome da fila SQS de entrada"
  type        = string
  default     = "event-queue"
}

variable "event_dlq_name" {
  description = "Nome da Dead Letter Queue"
  type        = string
  default     = "event-dlq"
}

variable "lambda_function_name" {
  description = "Nome da função Lambda"
  type        = string
  default     = "event-processor-lambda"
}

variable "lambda_role_name" {
  description = "Nome do IAM role para Lambda"
  type        = string
  default     = "event_processor_lambda_role"
}

variable "visibility_timeout_seconds" {
  description = "Visibility timeout da fila SQS (segundos)"
  type        = number
  default     = 30
}

variable "max_receive_count" {
  description = "Máximo de tentativas antes de enviar para DLQ"
  type        = number
  default     = 3
}

variable "lambda_batch_size" {
  description = "Tamanho do batch para procesar eventos da SQS"
  type        = number
  default     = 10
}

variable "lambda_zip_path" {
  description = "Caminho para o arquivo ZIP da Lambda"
  type        = string
  default     = "../../../build/function.zip"
}

variable "lambda_runtime" {
  description = "Runtime da Lambda"
  type        = string
  default     = "provided.al2023"
}

variable "lambda_handler" {
  description = "Handler da Lambda"
  type        = string
  default     = "bootstrap"
}

variable "s3_datalake_raw_zone_expiration_days" {
  description = "Número de dias para expiração de objetos no S3"
  type        = number
  default     = 60
}

variable "kinesis_shard_count" {
  description = "Número de shards para o Kinesis Stream"
  type        = number
  default     = 1
}

variable "kinesis_retention_period" {
  description = "Período de retenção do Kinesis Stream (em horas)"
  type        = number
  default     = 24
}

variable "tags" {
  description = "Tags comuns para todos os recursos"
  type        = map(string)
  default = {
    ManagedBy = "Terraform"
  }
}