---
title: Service Architecture
weight: 20
description: Protobuf boundary, in-process bufconn client, and session lookup strategies
categories: [internals]
tags: [agents]
---

`sharur` follows a **Strict Protobuf Internal Architecture**. Instead of UI modes calling Go functions directly, all interfaces are treated as clients of a central `AgentService`.

---

## Protobuf Boundary

The interface between the UI and the core is defined in `proto/sharur/v1/agent.proto`. This boundary ensures:

- **Consistency**: All modes (TUI, CLI, JSON, Remote gRPC) use the exact same code paths and logic.
- **Decoupling**: UI logic is completely isolated from agent state, session persistence, and provider adapters.
- **Interoperability**: Any gRPC-capable client can interact with a `sharur` service.

---

## In-Process Communication

For local CLI usage, `sharur` uses a specialized **In-Process Client** (`internal/service/client.go`). It uses `bufconn` to implement the `pb.AgentServiceClient` interface over an in-memory pipe. This provides the safety and structure of gRPC without the latency or configuration complexity of network ports.

---

## Backend Service (`internal/service`)

The `Service` struct implements `pb.AgentServiceServer`. It owns the `session.Manager` and manages the lifecycle of `agent.Agent` instances. It translates between internal agent events (Go channels) and Protobuf event streams.

---

## Session Loading Strategy

RPCs split into three lookup strategies:

| Strategy | Used by | Behaviour |
|---|---|---|
| `getOrCreate(id)` | `Prompt`, `NewSession` | Always returns an entry — creates a fresh agent if `id` is unknown, loading from disk if a matching session file exists |
| `loadIfExists(id)` | `GetState`, `GetMessages`, `ConfigureSession`, `ForkSession`, `CloneSession` | Returns the entry if it is in memory **or** can be loaded from disk; returns `NotFound` for completely unknown IDs |
| `lookup(id)` | `Steer`, `Abort`, `FollowUp`, `StreamEvents` | In-memory only — these only make sense for a currently-running agent |

This means a `/resume <id>` command can switch to any session ever saved to disk without a round-trip `NewSession` call: the first `GetMessages` or `GetState` call transparently loads it.
