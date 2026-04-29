---
title: Python Extensions
weight: 40
description: Building gollm extensions in Python using the gRPC proto stubs
---

Python extensions use the same gRPC protocol as Go extensions. The loader detects `.py` files and runs them with the configured Python interpreter, passing `GOLLM_SOCKET_PATH` as an environment variable. The extension is expected to listen on that Unix socket.

---

## Prerequisites

```bash
pip install grpcio grpcio-tools
```

---

## Generate Python Stubs

```bash
python -m grpc_tools.protoc \
  -I extensions/proto \
  --python_out=.gollm/extensions \
  --grpc_python_out=.gollm/extensions \
  extensions/proto/extension.proto
```

This deposits `extension_pb2.py` and `extension_pb2_grpc.py` alongside your script.

---

## Implement the Extension

```python
# .gollm/extensions/ticket_context.py
import os
import subprocess
import grpc
from concurrent import futures
import extension_pb2
import extension_pb2_grpc


class TicketContextServicer(extension_pb2_grpc.ExtensionServicer):
    def Name(self, request, context):
        return extension_pb2.NameResponse(name="ticket-context")

    def Tools(self, request, context):
        return extension_pb2.ToolsResponse(tools=[])

    def BeforePrompt(self, request, context):
        branch = subprocess.check_output(
            ["git", "rev-parse", "--abbrev-ref", "HEAD"], text=True
        ).strip()
        state = request.state or extension_pb2.AgentState()
        state.prompt += f"\n\n<branch>Current branch: {branch}</branch>"
        return extension_pb2.BeforePromptResponse(state=state)

    def BeforeToolCall(self, request, context):
        return extension_pb2.BeforeToolCallResponse(intercept=False)

    def AfterToolCall(self, request, context):
        return extension_pb2.AfterToolCallResponse(result=request.result)

    def ModifySystemPrompt(self, request, context):
        return extension_pb2.ModifySystemPromptResponse(
            modified_prompt=request.current_prompt
        )

    def AgentStart(self, request, context):
        return extension_pb2.Empty()

    def AgentEnd(self, request, context):
        return extension_pb2.Empty()

    def ModifyInput(self, request, context):
        return extension_pb2.ModifyInputResponse(action="continue", text=request.text)


def serve():
    socket_path = os.environ["GOLLM_SOCKET_PATH"]
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    extension_pb2_grpc.add_ExtensionServicer_to_server(TicketContextServicer(), server)
    server.add_insecure_port(f"unix:{socket_path}")
    server.start()
    server.wait_for_termination()


if __name__ == "__main__":
    serve()
```

Place the script in your extensions directory. `gollm` runs it as `python ticket_context.py` on startup.

---

## Available RPC Methods

Implement any subset of the `ExtensionServicer` methods. Unimplemented methods should return a sensible empty response (see the template above). The full list mirrors the Go plugin interface — see [Go Extensions](extensions-go/) for hook semantics.

| RPC | Purpose |
|---|---|
| `Name` | Return extension identifier |
| `Tools` | Return tool definitions |
| `ExecuteTool` | Execute a registered tool |
| `SessionStart` / `SessionEnd` | Session lifecycle |
| `AgentStart` / `AgentEnd` | Per-prompt lifecycle |
| `TurnStart` / `TurnEnd` | Per-LLM-turn lifecycle |
| `ModifyInput` | Transform or consume user input |
| `ModifySystemPrompt` | Augment the system prompt |
| `BeforePrompt` | Mutate model/provider/thinking |
| `ModifyContext` | Filter or inject LLM-bound messages |
| `BeforeProviderRequest` | Modify the raw completion request |
| `AfterProviderResponse` | Observe LLM output |
| `BeforeToolCall` | Intercept or block tool calls |
| `AfterToolCall` | Observe or modify tool results |
| `BeforeCompact` / `AfterCompact` | Compaction lifecycle |

---

## Tips

- **Logs go to stderr.** Python's `print()` goes to stdout, which is not read by the host. Use `sys.stderr.write()` or `logging` for debugging output.
- **Keep proto stubs in the same directory** as your script, or adjust `sys.path` before importing them.
- **Thread safety:** `grpc.server` with `ThreadPoolExecutor` handles concurrent RPC calls. If you maintain per-session state, use a lock or session-keyed dict.
