# Architecture

This page names the programs that make up the app, where each one runs and how they meet.
The publisher's computer captures and encodes a screen.
One relay carries it.
Each viewer's computer decodes it.
No stream travels between two computers directly.

```
 publisher's computer            relay host                  viewer's computer
┌────────────────────┐        ┌──────────────┐        ┌────────────────────┐
│ shell              │        │ Caddy        │        │ shell              │
│   control API      │        │ groupd       │        │   control API      │
│   frame channel    │        │ discordd     │        │   frame channel    │
│                    │        │ watchd       │        │                    │
│ backend ───────────┼stream─►│ MediaMTX     ├stream─►│ backend            │
└────────────────────┘        └──────────────┘        └────────────────────┘
```

Both computers run the same two programs.
One computer can publish and watch at once.

## On each computer

| Program | Does | Page |
| --- | --- | --- |
| backend (Go) | captures, encodes, publishes and decodes, and decides every value, option and figure | [ipc-api.md](ipc-api.md) |
| shell (Avalonia) | draws what the backend holds, asks it to act and writes every word on screen | [avalonia/README.md](../avalonia/README.md) |

The backend decides what is true.
The shell decides how to say it.
Control API: gRPC over a local socket, with six methods (`Hello`, `Read`, `Watch`, `Ask`, `Set`, `Do`).
`Set` names a state to make true, so a second call with the same state changes nothing.
The two sides meet on identifiers such as `hevc_nvenc` and `srt`, which the shell turns into words.
Frame channel: decoded frames stay in shared GPU memory.
The backend lends a frame to the shell, which draws it and hands it back.

## On the relay host

| Program | Does | Page |
| --- | --- | --- |
| MediaMTX | takes every stream in and serves it to each viewer over the protocol that viewer asked for | [network-architecture.md](network-architecture.md) |
| groupd | makes group keys, signs tokens, lists a group's streams and holds members' presence leases | [auth-flow.md](auth-flow.md) |
| discordd | turns a Discord voice channel into a group | [discord-mode.md](discord-mode.md) |
| watchd | carries each viewer's own figures to the stream's publisher | [viewer-reports.md](viewer-reports.md) |
| Caddy | terminates TLS for every HTTP leg on port 443 | [network-architecture.md](network-architecture.md) |

A group key stays secret.
A stream's path stays public, its prefix derived from the key.
MediaMTX checks a token at the handshake, so only members publish or watch.
Each stream crosses two legs: publish (publisher to relay) and watch (relay to viewer).
The settings name one protocol per leg.
Publish takes `srt`, `rtsp`, `rtmp` or `webrtc`.
Watch adds `hls` and `moq`.

## Facts live in tables

Codecs, transports and the formats each leg carries sit in tables in the backend ([domain-model.md](domain-model.md)).
Every rule, every greyed option and every offered value derives from them.

## Where the code lives

| Directory | Holds |
| --- | --- |
| `api/` | the contract between backend and shell |
| `backend/` | the backend, groupd and discordd |
| `avalonia/` | the shell |
| `deploy/` | what the relay host runs |
| `packaging/` | every installer and package |
| `tools/` | probes for a running relay or backend, and the renderer for the comparison crops |

How code takes shape: [development-principles.md](development-principles.md).
How screens take shape: [design-principles.md](design-principles.md).
