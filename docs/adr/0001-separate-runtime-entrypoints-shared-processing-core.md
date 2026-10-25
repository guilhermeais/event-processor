# Separate Runtime Entrypoints with a Shared Processing Core

Status: accepted

The project will add a separate `cmd/worker` executable for Kubernetes while retaining `cmd/lambda` for AWS Lambda. Both entrypoints will use the same processing core for schema loading and caching, validation, persistence, and `ACK`/`RETRY`/`TO_DLQ` decisions. This preserves the existing Lambda contract while preventing business behavior from being duplicated in the worker.

## Boundary

```text
cmd/lambda/main.go                 cmd/worker/main.go
        |                                  |
        v                                  v
Lambda Consumer Adapter        Kubernetes Worker Adapter
        |                                  |
        +---------------+------------------+
                        v
                 Processing Core
          schemas -> validate -> persist
                        |
                 Processing Outcome
                 ACK | RETRY | TO_DLQ
```

The adapters own delivery mechanics. The core owns the meaning of processing an event.

## Composition Root

Dependency construction will be centralized in one shared composition root. The composition root assembles the Processing Core and its infrastructure dependencies; it does not decide how messages are acknowledged.

```text
cmd/lambda -----------------> Lambda adapter
                                  |
cmd/worker -----------------> SQS Worker adapter
                                  |
                                  v
                         shared composition root
                                  |
             +--------------------+--------------------+
             |                    |                    |
        Schema Provider       Validator             Persister
             |                    |                    |
             +--------------------+--------------------+
                                  |
                              Processor
```

The exact package name for the composition root remains an implementation decision. The important boundary is that both runtime entrypoints use the same assembly path.

The project will not introduce a generic `EntryPoint` interface. Lambda and the SQS Worker have different lifecycle and acknowledgement contracts:

```go
// Lambda adapter: invoked by the Lambda runtime.
Handle(ctx context.Context, event events.SQSEvent) events.SQSEventResponse

// SQS Worker adapter: owns a long-running lifecycle.
Run(ctx context.Context) error
```

The project will also avoid one factory type per constructor, such as `ValidatorFactory` or `LambdaFactory`, unless a real construction variation justifies it. A factory that only forwards arguments to an existing constructor adds indirection without creating a useful seam.

## Shared Contract

Both adapters must pass the same normalized message to the core:

```json
{
  "client_id": "tenant-123",
  "event_id": "evt-01",
  "event_type": "invoice.created",
  "body": {"invoice_id": "inv-99", "amount": 1200}
}
```

The core returns an explicit outcome rather than an adapter-specific error:

| Outcome | Meaning | Lambda adapter | Kubernetes adapter |
| --- | --- | --- | --- |
| `ACK` | Persisted successfully | Do not report batch failure | Delete SQS message |
| `RETRY` | Recoverable failure | Report this batch item failure | Leave message undeleted |
| `TO_DLQ` | Terminal failure | Send to DLQ and complete the item | Send to DLQ, then delete source |

## Implementation Rules

- `cmd/lambda` must continue to satisfy the existing `ReportBatchItemFailures` contract.
- `cmd/worker` must not import Lambda event types.
- The processing core must not import Lambda, Kubernetes, or container orchestration packages.
- AWS client construction and schema-cache construction should be reusable, while message acknowledgement remains adapter-owned.
- Shared dependency construction must go through the composition root rather than being duplicated in each `cmd/*` entrypoint.
- Tests for the core must not require Lambda or Kubernetes.
- Adapter tests must verify translation and acknowledgement independently.

## Example: Same Failure, Different Delivery Action

If DynamoDB throttles while processing `evt-01`, the core returns `RETRY` once. Lambda returns `evt-01` in `batchItemFailures`; Kubernetes keeps the receipt handle without calling `DeleteMessage`. Neither adapter changes the business decision.

## Considered Options

- Replace the Lambda entrypoint with a container-only process: rejected because it removes the existing deployment model and complicates rollback.
- Duplicate the processing flow in a Kubernetes handler: rejected because the two runtimes would eventually diverge.

## Consequences

Runtime-specific delivery behavior remains at the boundary: Lambda reports batch item failures, while Kubernetes owns polling and message deletion.

The codebase gains a clear seam for a second runtime, but shared initialization must be kept free of Lambda-only assumptions. Any change to validation, schema loading, persistence, or outcome semantics must be tested against both adapters.

## Implementation Checklist

- [ ] Extract shared dependency construction from `cmd/lambda/main.go`.
- [ ] Define the normalized message and processing outcome types.
- [ ] Implement `cmd/worker/main.go` using the shared construction path.
- [ ] Keep Lambda-specific batch response handling in the Lambda adapter.
- [ ] Add unit tests for shared outcomes and adapter-specific acknowledgement.

See also: [RFC 0001](../rfc/0001-lambda-and-kubernetes-consumers.md).
