# Getting Started

## Experience HotPlex in 5 Minutes

Welcome! This guide is designed to get your first HotPlex agent up and running as quickly as possible. We focus on the developer experience, ensuring you see results without wading through deep technical specs.

---

### 1. Installation

HotPlex is distributed as a single, high-performance binary. Choose the method that fits your workflow:

::: code-group
```bash [Binary Installation]
# Download the latest release from GitHub
curl -sfL https://raw.githubusercontent.com/hrygo/hotplex/main/scripts/install.sh | sh
```

```bash [Go Install]
# Build from source using the latest version
go install github.com/hrygo/hotplex/cmd/hotplexd@latest
```
:::

---

### 2. Launch the Daemon

The `hotplexd` daemon is the heart of the system. Start it in development mode to begin:

```bash
hotplexd --help
```

> [!TIP]
> In dev mode, HotPlex uses an in-memory state store and opens a local administration portal at `http://localhost:8080`.

---

### 3. Deploy Your First Agent

HotPlex comes with a "Standard Oracle" template to help you get started. Deploy it to see the system in action:

```bash
# Create a new session
hotplexd session create --name "my-first-agent"

# Bind a ChatApp (e.g., Slack)
hotplexd bind slack --channel "general"
```

---

### Next Steps

Now that you've crossed the threshold, it's time to explore the depth of the platform:

| Path          | Goal                       | Link                                         |
| :------------ | :------------------------- | :------------------------------------------- |
| **Architect** | Understand the core engine | [Architecture Overview](/guide/architecture) |
| **Developer** | Build custom behaviors     | [Hooks & SDKs](/guide/hooks)                 |
| **Operator**  | Deploy to production       | [Production Guide](/guide/deployment)        |

---

> "We handle the state, you handle the soul." — The HotPlex Team
