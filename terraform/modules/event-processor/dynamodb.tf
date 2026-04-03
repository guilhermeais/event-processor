resource "aws_dynamodb_table" "schemas" {
  name         = local.schema_table_name
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "event_type"

  attribute {
    name = "event_type"
    type = "S"
  }

  tags = local.tags
}

resource "aws_dynamodb_table" "events" {
  name         = local.events_table_name
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "client_id"
  range_key    = "event_id"

  attribute {
    name = "client_id"
    type = "S"
  }

  attribute {
    name = "event_id"
    type = "S"
  }

  tags = local.tags
}
