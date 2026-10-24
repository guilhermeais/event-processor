# At-Least-Once Delivery with Idempotent Persistence

Status: accepted

The event processor will keep SQS at-least-once delivery semantics and will treat duplicate delivery as an expected condition. DynamoDB persistence will continue to use the `client_id#event_id` identity and conditional write behavior so a redelivery cannot create a second stored event. This is required for safe retries, visibility-timeout expiration, concurrent consumers, and process restarts.

## Processing Timeline

```text
ReceiveMessage
      |
      v
Validate and persist
      |
      +--> persistence succeeds --> DeleteMessage
      |
      +--> recoverable failure --> no delete --> redelivery
      |
      +--> terminal failure --> SendMessage(DLQ) --> DeleteMessage(source)
```

The delete is deliberately last. A consumer crash before deletion may cause the same message to be received again; the persistence operation must make that safe.

## Idempotency Example

For this message:

```text
client_id = tenant-123
event_id  = evt-01
storage id = tenant-123#evt-01
```

| Attempt | Event table operation | Result |
| --- | --- | --- |
| 1 | Conditional put: id does not exist | Event is stored |
| 2 after redelivery | Conditional put: id already exists | No second event is created |

The second attempt must be treated as safe completion from the perspective of queue handling. It must not send a valid duplicate to the DLQ merely because the idempotency condition reports that the first attempt already persisted it.

## Crash Scenarios

| Crash point | Expected result |
| --- | --- |
| Before persistence | Message becomes visible and is retried |
| After persistence, before delete | Message is redelivered; conditional write prevents duplication |
| During delete | Message may be redelivered; processing remains idempotent |
| After successful delete | No further delivery is expected from that receive |
| After DLQ send, before source delete | Source may redeliver; DLQ handling must avoid creating an uncontrolled duplicate if possible |

The last case requires the implementation to make the DLQ transition observable and to define how duplicate terminal handling is diagnosed. SQS itself does not provide an atomic "send to DLQ and delete source" operation.

## Considered Options

- Attempt exactly-once processing in the consumers: rejected because SQS visibility and process failure boundaries cannot provide that guarantee.
- Deduplicate only in the worker: rejected because Lambda and Kubernetes share the same domain behavior and both must be safe independently.

## Consequences

Successful processing must be safe to repeat, and message deletion must occur only after the corresponding outcome has been durably handled.

Exactly-once processing is not a valid system guarantee. The guarantee is instead: no event is intentionally lost before successful acknowledgement, and repeated valid events do not create multiple persisted records for the same `client_id#event_id`.

## Implementation Checklist

- [ ] Preserve the conditional write key and behavior in the shared persister.
- [ ] Ensure both Lambda and worker delete only after durable handling.
- [ ] Treat an already-persisted idempotent event as successful completion.
- [ ] Add tests for crash-equivalent redelivery: process twice, assert one stored item.
- [ ] Emit `client_id`, `event_id`, attempt context, and consumer runtime in logs.

See also: [ADR 0002](0002-concurrent-consumers-on-one-sqs-queue.md) and [ADR 0004](0004-explicit-worker-polling-and-graceful-shutdown.md).
