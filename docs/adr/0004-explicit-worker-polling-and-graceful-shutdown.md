# Explicit Worker Polling and Graceful Shutdown

Status: accepted

The Kubernetes Worker will own an explicit SQS long-polling loop, bounded processing concurrency, visibility management, message deletion, and graceful shutdown. It will stop polling on termination, wait for in-flight work until a deadline, and leave unfinished messages undeleted. This is necessary because Kubernetes does not provide Lambda's managed invocation and batch-failure contract.

## Worker State Flow

```text
STARTING -> READY -> STOPPING -> STOPPED
               |       |
               |       +--> stop ReceiveMessage
               |       +--> wait for in-flight work
               |       +--> deadline reached: leave unfinished messages
               |
               +--> poll -> bounded pool -> outcome -> acknowledgement
```

## Receive and Process Example

Assume `worker_concurrency=4` and SQS returns three messages:

1. Receive up to the configured batch size.
2. Put at most four messages in the processing pool.
3. For each message, call the shared processing core.
4. `ACK` calls `DeleteMessage` with that message's receipt handle.
5. `RETRY` does not call `DeleteMessage`.
6. `TO_DLQ` sends to the DLQ, then deletes only after the send succeeds.

The worker must retain the receipt handle associated with each individual message. A receipt handle from a later receive must not be used to delete an earlier receive.

## Visibility Timeout

The current queue default is 30 seconds. The implementation must choose one of these safe paths for each message:

```text
processing duration < visibility timeout - safety margin
                         OR
periodically ChangeMessageVisibility while processing
```

Visibility extension failures are not silent successes. The worker should record the failure and let the message be redelivered if it can no longer guarantee exclusive processing.

## Shutdown Example

```text
SIGTERM
  |
  +--> cancel receive context
  +--> no new messages accepted
  +--> in-flight ACK completes -> delete allowed
  +--> in-flight RETRY -> no delete
  +--> deadline -> cancel remaining work, no delete
```

Kubernetes `terminationGracePeriodSeconds` must be longer than the worker's configured shutdown deadline plus a small process-exit margin. The worker must not claim readiness while it is stopping.

## Considered Options

- Use an unbounded goroutine per received message: rejected because queue bursts could exhaust pod resources and make shutdown unpredictable.
- Delete messages before processing: rejected because process failure after deletion would lose events.
- Rely only on the current 30-second visibility timeout: rejected because long processing can cause concurrent redelivery.

## Consequences

The worker requires operational settings for concurrency, batch size, visibility extension, shutdown deadline, and queue permissions. At-least-once processing remains possible when a pod terminates during work.

The worker has more operational code than the Lambda adapter because it must implement lifecycle and acknowledgement behavior explicitly. That complexity is isolated in the worker adapter and does not change the processing core.

## Implementation Checklist

- [ ] Implement long polling with cancellation support.
- [ ] Bound in-flight messages with a configurable worker pool.
- [ ] Preserve each message's receipt handle through processing.
- [ ] Implement delete-after-`ACK` and delete-after-successful-DLQ-send.
- [ ] Implement visibility extension or enforce a documented processing limit.
- [ ] Handle `SIGTERM` and `SIGINT` with a bounded drain.
- [ ] Add readiness behavior that becomes false during shutdown.
- [ ] Test receive errors, delete errors, visibility errors, and termination during processing.

See also: [ADR 0003](0003-at-least-once-delivery-and-idempotent-persistence.md).
