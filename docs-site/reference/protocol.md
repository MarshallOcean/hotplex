# Protocol Specification

## The Duplex Messaging Protocol (DMP)

HotPlex uses a specialized messaging protocol over WebSockets, designed specifically for the unpredictable and iterative nature of AI agent interactions. The **Duplex Messaging Protocol (DMP)** ensures that both the user and the agent can remain synchronized in terms of state and context.

---

### Message Structure

All DMP messages are JSON-serialized and follow a strict schema:

```json
{
  "id": "msg_01JHGTVR",
  "type": "think | action | output | system",
  "timestamp": "2026-03-01T08:21:00Z",
  "payload": { ... },
  "metadata": { ... }
}
```

---

### Event Lifecycle

A typical interaction follows this lifecycle:

1.  **Handshake**: The client connects and provides credentials.
2.  **Input Pulse**: The user sends a prompt via a ChatApp.
3.  **Thought Cycle**: The engine emits one or more `think` events.
4.  **Action Pulse**: The engine emits an `action` event.
5.  **Observation**: The tool returns a result, which is fed back into the Thought Cycle.
6.  **Resolution**: The engine emits an `output` event.

---

### Guaranteed Delivery & State Sync

The DMP is designed for **Resilience**:

- **Sequence Tracking**: Every message includes a sequence ID to ensure correct ordering even over high-latency connections.
- **Heartbeats**: The protocol maintains a low-level heartbeat to detect connection drops and trigger immediate state cleanup or reconnection logic.
- **State Checkpoints**: At critical points in the lifecycle, the engine performs a "State Checkpoint," ensuring that if the connection is lost, the agent can resume from the exact last "thought."

---

### Technical Implementation

The DMP is implemented as a high-performance event loop within the HotPlex core engine. While we are standardizing the internal representation, the wire format remains stable via JSON over WebSockets.

[View the Server Implementation on GitHub](https://github.com/hrygo/hotplex/blob/main/cmd/hotplexd/main.go)
