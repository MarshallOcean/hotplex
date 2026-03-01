# API Reference

## Building with the HotPlex Runtime

The HotPlex API is designed for high-performance agentic interactions. We provide two primary interfaces: a **RESTful Control Plane** for configuration and management, and a **Streaming Data Plane** for real-time agent execution.

---

### Authentication

All API requests must include a Bearer token in the `Authorization` header:

```http
Authorization: Bearer [HOTPLEX_API_KEY]
```

---

### The REST Control Plane

Manage your sessions, agents, and bindings programmatically.

| Endpoint           | Method | Description                                          |
| :----------------- | :----- | :--------------------------------------------------- |
| `/session`     | `POST` | Create a new stateful agent session.                 |
| `/session/:id` | `GET`  | Retrieve the current state and context of a session. |
| `/v1/bindings`     | `GET`  | List all active ChatApp bindings.                    |
| `/v1/hooks`        | `POST` | Register a new custom hook endpoint.                 |

#### Example: Create a Session
```json
// POST /session
{
  "name": "coding-assistant",
  "template": "standard-oracle",
  "metadata": {
    "project": "hotplex-docs"
  }
}
```

---

### The Streaming Data Plane

For real-time agent interactions, we use a **Duplex WebSocket** connection. This is the "Data Plane" where the agent's thinking and action cycles occur.

#### URI Pattern
`ws://[HOTPLEX_HOST]/ws/v1/agent`

#### Key Message Types
- `think`: The agent is processing information.
- `action`: The agent is calling a tool or performing an action.
- `output`: Final or intermediate output for the user.
- `error`: Diagnostic information.

---

### Discover the SDKs

While the API is raw and powerful, we recommend using our official SDKs for a more idiomatic developer experience:

- [Go SDK](/sdks/go-sdk)
- [Python SDK](/sdks/python-sdk)
- [TypeScript SDK](/sdks/typescript-sdk)
