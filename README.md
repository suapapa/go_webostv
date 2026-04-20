# go_webostv

[![Go Reference](https://pkg.go.dev/badge/github.com/suapapa/go_webostv.svg)](https://pkg.go.dev/github.com/suapapa/go_webostv)

A Go port of [PyWebOSTV](https://github.com/supersaiyanmode/PyWebOSTV), a library to remote control LG WebOS TVs.

## Features

- SSDP Discovery of TVs on the network.
- WebSocket-based connection and registration (pairing).
- Media Controls (volume, playback, etc.).
- System Controls (notification, power off, etc.).
- Application Controls (list apps, launch, close).
- TV Controls (channel change, program info).
- Input Controls (mouse movement, button clicks).
- Subscription support (volume, etc.).

## Installation

```bash
go get github.com/suapapa/go_webostv
```

## Usage

See `example/main.go` for a complete example.

### Basic Connection

```go
client := webostv.NewClient("192.168.1.10", false)
err := client.Connect()
if err != nil {
    log.Fatal(err)
}

store := make(map[string]string) // Persist this to avoid re-pairing
statusChan, errChan := client.Register(store)
// Handle statusChan and errChan
```

### Media Control

```go
media := &webostv.MediaControl{Control: webostv.Control{Client: client}}
media.VolumeUp()
```

### Input Control (Mouse/Buttons)

```go
input := &webostv.InputControl{Control: webostv.Control{Client: client}}
err := input.ConnectInput()
if err == nil {
    input.Move(10, 10, 0)
    input.Click()
    input.DisconnectInput()
}
```

## Credits

- This library is a port of [PyWebOSTV](https://github.com/supersaiyanmode/PyWebOSTV).
- Credits to the original author Srivatsan Iyer and contributors.
