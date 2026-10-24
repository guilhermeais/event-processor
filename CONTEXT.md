# Event Processing Context

This context defines the language used by the event processor when consuming, validating, and persisting tenant events.

## Event Flow

**Event**:
An input message containing a tenant identifier, event identifier, event type, and JSON payload for processing.
_Avoid_: Record, request, notification

**Event Type**:
The identifier that selects the schema used to validate an event payload.
_Avoid_: Topic, message type, route

**Schema Catalog**:
The authoritative collection of event schemas used by the event processor.
_Avoid_: Schema registry, contract cache

**Schema Cache**:
A local, compiled representation of the schema catalog used to validate events. It is temporary, has a TTL, and is refreshed from the schema catalog.
_Avoid_: Schema catalog, registry

## Processing Outcomes

**ACK**:
The event was valid and persisted successfully, so its source queue message can be deleted.
_Avoid_: Success, complete

**RETRY**:
Processing could not complete because of a recoverable failure, so the source queue message must remain available for another attempt.
_Avoid_: Temporary error, failure

**TO_DLQ**:
Processing determined that the event cannot succeed through another attempt, so it is sent to the dead-letter queue before the source message is removed.
_Avoid_: Reject, discard, poison message

**Dead-Letter Queue**:
The destination for events that are invalid or cannot be processed after the defined retry behavior.
_Avoid_: Error queue, failure queue

## Runtime Roles

**Lambda Consumer**:
The AWS Lambda input adapter that receives SQS batches through an Event Source Mapping and reports individual batch item failures.
_Avoid_: Lambda worker, Lambda service

**Kubernetes Worker**:
The long-running SQS input adapter that polls, processes, and deletes messages from a Kubernetes pod.
_Avoid_: K8S worker, pod consumer, Kubernetes Lambda

**Processing Core**:
The shared application logic that loads schemas, validates event payloads, applies processing decisions, and persists accepted events without depending on a runtime adapter.
_Avoid_: Worker logic, Lambda logic, handler

**Consumer**:
Either the Lambda Consumer or Kubernetes Worker. Consumers compete for messages when attached to the same SQS queue; they do not each receive a copy of every message.
_Avoid_: Processor, subscriber
