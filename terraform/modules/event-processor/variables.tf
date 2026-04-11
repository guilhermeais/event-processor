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

  validation {
    condition     = var.lambda_batch_size >= 1 && var.lambda_batch_size <= 10000
    error_message = "lambda_batch_size deve estar entre 1 e 10000 para fila SQS standard."
  }
}

variable "lambda_max_batching_window_seconds" {
  description = "Janela máxima (segundos) para acumular mensagens antes de invocar a Lambda"
  type        = number
  default     = 5

  validation {
    condition     = var.lambda_max_batching_window_seconds >= 0 && var.lambda_max_batching_window_seconds <= 300
    error_message = "lambda_max_batching_window_seconds deve estar entre 0 e 300."
  }
}

variable "lambda_timeout_seconds" {
  description = "Timeout da Lambda em segundos"
  type        = number
  default     = 30

  validation {
    condition     = var.lambda_timeout_seconds >= 1 && var.lambda_timeout_seconds <= 900
    error_message = "lambda_timeout_seconds deve estar entre 1 e 900."
  }
}

variable "lambda_xray_tracing_mode" {
  description = "Modo de tracing do AWS X-Ray para Lambda (Active ou PassThrough)"
  type        = string
  default     = "Active"

  validation {
    condition     = contains(["Active", "PassThrough"], var.lambda_xray_tracing_mode)
    error_message = "lambda_xray_tracing_mode deve ser Active ou PassThrough."
  }
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