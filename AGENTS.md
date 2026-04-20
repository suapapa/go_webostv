# Agents Guide: go_webostv

This project is a Go port of [PyWebOSTV](https://github.com/supersaiyanmode/PyWebOSTV). It implements the LG WebOS TV WebSocket protocol for remote control.

## Project Structure

- `discovery.go`: Implements SSDP discovery for WebOS devices. Uses `github.com/koron/go-ssdp`.
- `client.go`: The core WebSocket client. Handles:
    - Connection to `ws://host:3000/` or `wss://host:3001/`.
    - Message framing with unique IDs and callback/waiter mapping.
    - Registration (pairing) flow with the TV.
    - Subscription/Unsubscription management.
- `controls.go`: High-level API wrappers.
    - `MediaControl`: Volume, Play/Pause, Mute.
    - `SystemControl`: Power, Notifications, System Info.
    - `ApplicationControl`: App listing and launching.
    - `TvControl`: Channel management.
    - `InputControl`: Mouse movement and specialized button commands (via a separate `socketPath` WebSocket).
- `models.go`: Shared data structures for Apps, Inputs, and Audio outputs.
- `example/`: Contains usage demonstrations.

## Technical Notes for Agents

### Protocol Handling
- Every request to the TV requires a unique ID (UUID).
- The `Client.Request` method is a blocking wrapper around `SendMessage` for simple request-response cycles.
- For registration, use `Client.Register` which returns a status channel to handle the "Prompt accepted" and "Registered" phases.

### Input Controls (Mouse/Buttons)
- Input controls (move, click, etc.) require a separate WebSocket connection obtained via `ssap://com.webos.service.networkinput/getPointerInputSocket`.
- The protocol for this connection is line-based text (`type:command\nkey:value\n\n`), not JSON.

### Porting Status
- The port aims for parity with PyWebOSTV.
- Complex argument extraction logic from Python (`arguments` helper) was simplified into standard Go method signatures.
- All `ssap://` URIs match the original library.

## Common Tasks
- **Adding a new command:** Identify the `ssap://` URI and payload from PyWebOSTV or other sources, then add a corresponding method to the relevant `*Control` struct in `controls.go`.
- **Debugging connections:** Check WebSocket handshakes and JSON unmarshaling in `client.go`.
