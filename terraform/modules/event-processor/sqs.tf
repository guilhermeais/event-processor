resource "aws_sqs_queue" "event_dlq" {
  name = local.event_dlq_name

  tags = local.tags
}

resource "aws_sqs_queue" "event_queue" {
  name                       = local.event_queue_name
  visibility_timeout_seconds = var.visibility_timeout_seconds

  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.event_dlq.arn
    maxReceiveCount     = var.max_receive_count
  })

  tags = local.tags
}
