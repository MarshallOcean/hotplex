const WebSocket = require('ws');

/**
 * HotPlex WebSocket Full Lifecycle Demo
 * 
 * Demonstrates:
 * 1. Session Initialization (Cold Start)
 * 2. Hot-Multiplexing (Reuse of persistent process)
 * 3. State Persistence (Using marker files)
 * 4. Manual Session Termination
 */

const WS_URL = 'ws://localhost:8080/ws/v1/agent';
const SESSION_ID = 'ws-full-demo-session';

async function runStep(label, payload) {
    console.log(`\n\x1b[36m>>> STEP: ${label}\x1b[0m`);
    return new Promise((resolve, reject) => {
        const ws = new WebSocket(WS_URL);
        let completed = false;

        ws.on('open', () => {
            console.log(`[WS] Connected. Sending: ${payload.type}`);
            ws.send(JSON.stringify(payload));
        });

        ws.on('message', (data) => {
            const msg = JSON.parse(data);

            switch (msg.event) {
                case 'thinking':
                    process.stdout.write('🤔 ');
                    break;
                case 'answer':
                    process.stdout.write(msg.data.EventData);
                    break;
                case 'tool_use':
                    process.stdout.write(`\n🛠️  Tool: ${msg.data.EventData}\n`);
                    break;
                case 'session_stats':
                    console.log('\n\x1b[2m[STATS]\x1b[0m', JSON.stringify(msg.data, null, 2));
                    break;
                case 'completed':
                    console.log('\n✅ Task Completed.');
                    completed = true;
                    ws.close();
                    break;
                case 'stopped':
                    console.log(`\n🛑 Session ${msg.data.session_id} stopped.`);
                    completed = true;
                    ws.close();
                    break;
                case 'error':
                    console.error('\n❌ Server Error:', msg.data);
                    ws.close();
                    reject(msg.data);
                    break;
            }
        });

        ws.on('close', () => {
            if (completed) resolve();
            else reject(new Error('Connection closed unexpectedly'));
        });

        ws.on('error', reject);
    });
}

async function main() {
    console.log("=== HotPlex WebSocket Full Lifecycle Demo ===");

    try {
        // [1] Cold Start: Set some context inside the session
        await runStep("Cold Start - Setting Context", {
            type: "execute",
            session_id: SESSION_ID,
            prompt: "Remember my name is 'Agent Zero'. Just say 'Got it'.",
            work_dir: process.cwd()
        });

        // [2] Hot-Multiplexing: Verify the context is still there (Reuse)
        // This will be fast as the process is already running.
        await runStep("Hot-Multiplexing - Verifying Context", {
            type: "execute",
            session_id: SESSION_ID,
            prompt: "What is my name?",
            work_dir: process.cwd()
        });

        // [3] Persistence: Simulate a server "restart" or reconnection
        // The underlying process might still be running or can be resumed via markers.
        console.log("\n[Note] You can restart the hotplexd server now if you want to test deep persistence.");
        await runStep("Persistence - Reconnecting", {
            type: "execute",
            session_id: SESSION_ID,
            prompt: "Am I still 'Agent Zero'?",
            work_dir: process.cwd()
        });

        // [4] Termination: Explicitly kill the session
        await runStep("Manual Termination", {
            type: "stop",
            session_id: SESSION_ID,
            reason: "demo finished"
        });

        console.log("\n\x1b[32m=== Full Demo Complete ===\x1b[0m");
    } catch (err) {
        console.error("Demo failed:", err);
        console.log("\n\x1b[31mEnsure 'hotplexd' is running on localhost:8080\x1b[0m");
    }
}

main();
