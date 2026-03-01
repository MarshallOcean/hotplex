# ChatApps Ecosystem

## Bringing Agents to Where Users Live

The true power of an AI agent is realized when it interacts with users in their natural environment. **ChatApps** are the specialized adapters that bridge the HotPlex engine to mainstream communication platforms.

---

### Supported Platforms

We aim to cover the spectrum of enterprise and community platforms:

| Platform       | Type       | Status       | Features                           |
| :------------- | :--------- | :----------- | :--------------------------------- |
| **Slack**      | Enterprise | ✅ Production | Rich Markdown, Reactions, App Home |
| **DingTalk**   | Enterprise | 🔄 Beta       | Corporate approvals, AI Cards      |
| **Discord**    | Community  | 🔄 Beta       | Guild management, Threaded bots    |
| **Web Portal** | Custom     | ✅ Production | Full-Duplex UI, Custom branding    |

---

### How ChatApps Work

Unlike simple webhooks, HotPlex ChatApps maintain a **continuous duplex connection** to the engine. This allows for:

- **Real-time Thinking Updates**: Show the user precisely what the agent is "thinking" and which tools it is using.
- **Interactive Action Zones**: Embed buttons, selectors, and complex forms directly into the chat interface.
- **Cross-Platform State**: Start a session on Slack and resume it on the Web Portal without losing context.

---

### Binding Your First App

Connecting a platform to HotPlex is a simple "Binding" operation:

1.  **Register App**: Create an app entry on the target platform (e.g., Slack Developer Portal).
2.  **Configure Credentials**: Provide the API tokens to `hotplexd`.
3.  **Establish Bind**:
    ```bash
    hotplexd bind slack --name "my-agent-bot" --token "[SLACK_BOT_TOKEN]"
    ```

---

### Looking Ahead

Our vision is a **Write-Once-Deploy-Everywhere** experience for agents. You build the logic; HotPlex ensures the UI is perfectly adapted for the specific constraints and capabilities of each platform.

[Learn about Slack Integration](/guide/chatapps-slack) or [See the Ecosystem Gallery](/ecosystem/).
