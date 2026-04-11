variable "lambda_batch_size" {
  description = "Tamanho do batch para processar eventos da SQS"
  type        = number
  default     = 100
}

variable "lambda_max_batching_window_seconds" {
  description = "Janela máxima (segundos) para acumular mensagens antes da invocação da Lambda"
  type        = number
  default     = 5
}

variable "lambda_timeout_seconds" {
  description = "Timeout da Lambda em segundos"
  type        = number
  default     = 60
}

variable "lambda_xray_tracing_mode" {
  description = "Modo de tracing do AWS X-Ray para Lambda (Active ou PassThrough)"
  type        = string
  default     = "Active"
}
