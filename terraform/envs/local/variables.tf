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

variable "aws_region" {
  description = "Região AWS"
  type        = string
  default     = "us-east-1"
}

variable "use_localstack" {
  description = "Se true, usa LocalStack em vez de AWS real"
  type        = bool
  default     = true
}

variable "localstack_endpoint" {
  description = "Endpoint do LocalStack (usado apenas se use_localstack = true)"
  type        = string
  default     = "http://localhost:4566"
}

variable "aws_access_key_id" {
  description = "AWS Access Key (deixar vazio para usar credenciais padrão em produção)"
  type        = string
  default     = "test"
  sensitive   = true
}

variable "aws_secret_access_key" {
  description = "AWS Secret Access Key (deixar vazio para usar credenciais padrão em produção)"
  type        = string
  default     = "test"
  sensitive   = true
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

variable "lambda_max_batching_window_seconds" {
  description = "Janela máxima (segundos) para acumular mensagens antes da invocação da Lambda"
  type        = number
  default     = 5
}

variable "lambda_timeout_seconds" {
  description = "Timeout da Lambda em segundos"
  type        = number
  default     = 30
}

variable "lambda_xray_tracing_mode" {
  description = "Modo de tracing do AWS X-Ray para Lambda (Active ou PassThrough)"
  type        = string
  default     = "Active"
}

variable "lambda_zip_path" {
  description = "Caminho para o arquivo ZIP da Lambda"
  type        = string
  default     = "../build/function.zip"
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

variable "tags" {
  description = "Tags comuns para todos os recursos"
  type        = map(string)
  default = {
    ManagedBy = "Terraform"
  }
}