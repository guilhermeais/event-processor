package ports

import "errors"

var ErrRetryable = errors.New("retryable error")
var ErrInvalidPayload = errors.New("invalid payload")
