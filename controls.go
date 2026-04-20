package webostv

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

// Control is the base struct for all controls.
type Control struct {
	Client *Client
}

// MediaControl provides methods to control media and volume.
type MediaControl struct {
	Control
}

func (m *MediaControl) VolumeUp(ctx context.Context) error {
	_, err := m.Client.Request(ctx, "ssap://audio/volumeUp", nil)
	return err
}

func (m *MediaControl) VolumeDown(ctx context.Context) error {
	_, err := m.Client.Request(ctx, "ssap://audio/volumeDown", nil)
	return err
}

func (m *MediaControl) GetVolume(ctx context.Context) (*VolumeInfo, error) {
	resp, err := m.Client.Request(ctx, "ssap://audio/getVolume", nil)
	if err != nil {
		return nil, err
	}
	var info VolumeInfo
	if err := json.Unmarshal(resp.Payload, &info); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}
	return &info, nil
}

func (m *MediaControl) SetVolume(ctx context.Context, volume int) error {
	payload := map[string]interface{}{"volume": volume}
	_, err := m.Client.Request(ctx, "ssap://audio/setVolume", payload)
	return err
}

func (m *MediaControl) Mute(ctx context.Context, mute bool) error {
	payload := map[string]interface{}{"mute": mute}
	_, err := m.Client.Request(ctx, "ssap://audio/setMute", payload)
	return err
}

func (m *MediaControl) Play(ctx context.Context) error {
	_, err := m.Client.Request(ctx, "ssap://media.controls/play", nil)
	return err
}

func (m *MediaControl) Pause(ctx context.Context) error {
	_, err := m.Client.Request(ctx, "ssap://media.controls/pause", nil)
	return err
}

func (m *MediaControl) Stop(ctx context.Context) error {
	_, err := m.Client.Request(ctx, "ssap://media.controls/stop", nil)
	return err
}

func (m *MediaControl) Rewind(ctx context.Context) error {
	_, err := m.Client.Request(ctx, "ssap://media.controls/rewind", nil)
	return err
}

func (m *MediaControl) FastForward(ctx context.Context) error {
	_, err := m.Client.Request(ctx, "ssap://media.controls/fastForward", nil)
	return err
}

func (m *MediaControl) GetAudioOutput(ctx context.Context) (*AudioOutputSource, error) {
	resp, err := m.Client.Request(ctx, "ssap://audio/getSoundOutput", nil)
	if err != nil {
		return nil, err
	}
	var out AudioOutputSource
	if err := json.Unmarshal(resp.Payload, &out); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}
	return &out, nil
}

func (m *MediaControl) SetAudioOutput(ctx context.Context, output string) error {
	payload := map[string]interface{}{"output": output}
	_, err := m.Client.Request(ctx, "ssap://audio/changeSoundOutput", payload)
	return err
}

func (m *MediaControl) SubscribeVolume(callback func(*VolumeInfo)) (string, error) {
	return m.Client.Subscribe("ssap://audio/getVolume", func(p json.RawMessage) {
		var info VolumeInfo
		if err := json.Unmarshal(p, &info); err == nil {
			callback(&info)
		}
	})
}

// SystemControl provides methods for system-level operations.
type SystemControl struct {
	Control
}

func (s *SystemControl) PowerOff(ctx context.Context) error {
	_, err := s.Client.Request(ctx, "ssap://system/turnOff", nil)
	return err
}

func (s *SystemControl) ScreenOff(ctx context.Context) error {
	payload := map[string]interface{}{"standbyMode": "active"}
	_, err := s.Client.Request(ctx, "ssap://com.webos.service.tvpower/power/turnOffScreen", payload)
	return err
}

func (s *SystemControl) ScreenOn(ctx context.Context) error {
	payload := map[string]interface{}{"standbyMode": "active"}
	_, err := s.Client.Request(ctx, "ssap://com.webos.service.tvpower/power/turnOnScreen", payload)
	return err
}

func (s *SystemControl) Info(ctx context.Context) (*SWInformation, error) {
	resp, err := s.Client.Request(ctx, "ssap://com.webos.service.update/getCurrentSWInformation", nil)
	if err != nil {
		return nil, err
	}
	var info SWInformation
	if err := json.Unmarshal(resp.Payload, &info); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}
	return &info, nil
}

func (s *SystemControl) Notify(ctx context.Context, message string, iconBytes []byte, iconExt string) error {
	payload := map[string]interface{}{
		"message": message,
	}
	if iconBytes != nil {
		payload["iconData"] = base64.StdEncoding.EncodeToString(iconBytes)
		payload["iconExtension"] = iconExt
	}
	_, err := s.Client.Request(ctx, "ssap://system.notifications/createToast", payload)
	return err
}

// ApplicationControl provides methods to manage applications.
type ApplicationControl struct {
	Control
}

func (a *ApplicationControl) ListApps(ctx context.Context) ([]Application, error) {
	resp, err := a.Client.Request(ctx, "ssap://com.webos.applicationManager/listApps", nil)
	if err != nil {
		return nil, err
	}
	var p struct {
		Apps []Application `json:"apps"`
	}
	if err := json.Unmarshal(resp.Payload, &p); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}
	return p.Apps, nil
}

func (a *ApplicationControl) Launch(ctx context.Context, appID string, contentID string, params map[string]interface{}) error {
	payload := map[string]interface{}{
		"id": appID,
	}
	if contentID != "" {
		payload["contentId"] = contentID
	}
	if params != nil {
		payload["params"] = params
	}
	_, err := a.Client.Request(ctx, "ssap://system.launcher/launch", payload)
	return err
}

func (a *ApplicationControl) GetForegroundApp(ctx context.Context) (string, error) {
	resp, err := a.Client.Request(ctx, "ssap://com.webos.applicationManager/getForegroundAppInfo", nil)
	if err != nil {
		return "", err
	}
	var p struct {
		AppID string `json:"appId"`
	}
	if err := json.Unmarshal(resp.Payload, &p); err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}
	return p.AppID, nil
}

func (a *ApplicationControl) Close(ctx context.Context, appID string) error {
	payload := map[string]interface{}{"id": appID}
	_, err := a.Client.Request(ctx, "ssap://system.launcher/close", payload)
	return err
}

// TvControl provides methods for TV channel management.
type TvControl struct {
	Control
}

func (t *TvControl) ChannelUp(ctx context.Context) error {
	_, err := t.Client.Request(ctx, "ssap://tv/channelUp", nil)
	return err
}

func (t *TvControl) ChannelDown(ctx context.Context) error {
	_, err := t.Client.Request(ctx, "ssap://tv/channelDown", nil)
	return err
}

func (t *TvControl) GetCurrentChannel(ctx context.Context) (*ChannelInfo, error) {
	resp, err := t.Client.Request(ctx, "ssap://tv/getCurrentChannel", nil)
	if err != nil {
		return nil, err
	}
	var info ChannelInfo
	if err := json.Unmarshal(resp.Payload, &info); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}
	return &info, nil
}

func (t *TvControl) ChannelList(ctx context.Context) ([]ChannelInfo, error) {
	resp, err := t.Client.Request(ctx, "ssap://tv/getChannelList", nil)
	if err != nil {
		return nil, err
	}
	var p struct {
		ChannelList []ChannelInfo `json:"channelList"`
	}
	if err := json.Unmarshal(resp.Payload, &p); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}
	return p.ChannelList, nil
}

func (t *TvControl) SetChannel(ctx context.Context, channelID string) error {
	payload := map[string]interface{}{"channelId": channelID}
	_, err := t.Client.Request(ctx, "ssap://tv/openChannel", payload)
	return err
}

// SourceControl provides methods to manage input sources.
type SourceControl struct {
	Control
}

func (s *SourceControl) ListSources(ctx context.Context) ([]InputSource, error) {
	resp, err := s.Client.Request(ctx, "ssap://tv/getExternalInputList", nil)
	if err != nil {
		return nil, err
	}
	var p struct {
		Devices []InputSource `json:"devices"`
	}
	if err := json.Unmarshal(resp.Payload, &p); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}
	return p.Devices, nil
}

func (s *SourceControl) SetSource(ctx context.Context, sourceID string) error {
	payload := map[string]interface{}{"inputId": sourceID}
	_, err := s.Client.Request(ctx, "ssap://tv/switchInput", payload)
	return err
}

// InputControl provides methods for mouse and keyboard input.
type InputControl struct {
	Control
	mouseConn   *websocket.Conn
	mouseConnMu sync.Mutex
}

func (i *InputControl) Type(ctx context.Context, text string) error {
	payload := map[string]interface{}{"text": text, "replace": 0}
	_, err := i.Client.Request(ctx, "ssap://com.webos.service.ime/insertText", payload)
	return err
}

func (i *InputControl) Delete(ctx context.Context, count int) error {
	payload := map[string]interface{}{"count": count}
	_, err := i.Client.Request(ctx, "ssap://com.webos.service.ime/deleteCharacters", payload)
	return err
}

func (i *InputControl) Enter(ctx context.Context) error {
	_, err := i.Client.Request(ctx, "ssap://com.webos.service.ime/sendEnterKey", nil)
	return err
}

func (i *InputControl) ConnectInput(ctx context.Context) error {
	i.mouseConnMu.Lock()
	defer i.mouseConnMu.Unlock()

	resp, err := i.Client.Request(ctx, "ssap://com.webos.service.networkinput/getPointerInputSocket", nil)
	if err != nil {
		return err
	}
	var p struct {
		SocketPath string `json:"socketPath"`
	}
	if err := json.Unmarshal(resp.Payload, &p); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}

	if p.SocketPath == "" {
		return errors.New("unable to connect to mouse: empty socket path")
	}

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, p.SocketPath, nil)
	if err != nil {
		return fmt.Errorf("failed to dial mouse socket: %w", err)
	}
	i.mouseConn = conn
	return nil
}

func (i *InputControl) DisconnectInput() error {
	i.mouseConnMu.Lock()
	defer i.mouseConnMu.Unlock()

	if i.mouseConn != nil {
		err := i.mouseConn.Close()
		i.mouseConn = nil
		return err
	}
	return nil
}

func (i *InputControl) sendMouseCommand(cmd string, params ...interface{}) error {
	i.mouseConnMu.Lock()
	defer i.mouseConnMu.Unlock()

	if i.mouseConn == nil {
		return errors.New("mouse not connected")
	}
	parts := []string{"type:" + cmd}
	for j := 0; j < len(params); j += 2 {
		parts = append(parts, fmt.Sprintf("%v:%v", params[j], params[j+1]))
	}
	payload := strings.Join(parts, "\n") + "\n\n"
	return i.mouseConn.WriteMessage(websocket.TextMessage, []byte(payload))
}

func (i *InputControl) Move(dx, dy int, drag int) error {
	return i.sendMouseCommand("move", "dx", dx, "dy", dy, "down", drag)
}

func (i *InputControl) Click() error {
	return i.sendMouseCommand("click")
}

func (i *InputControl) Scroll(dx, dy int) error {
	return i.sendMouseCommand("scroll", "dx", dx, "dy", dy)
}

func (i *InputControl) Button(name string) error {
	return i.sendMouseCommand("button", "name", name)
}

// Specific buttons
func (i *InputControl) Left() error         { return i.Button("LEFT") }
func (i *InputControl) Right() error        { return i.Button("RIGHT") }
func (i *InputControl) Up() error           { return i.Button("UP") }
func (i *InputControl) Down() error         { return i.Button("DOWN") }
func (i *InputControl) Home() error         { return i.Button("HOME") }
func (i *InputControl) Back() error         { return i.Button("BACK") }
func (i *InputControl) Menu() error         { return i.Button("MENU") }
func (i *InputControl) OK() error           { return i.Button("ENTER") }
func (i *InputControl) Dash() error         { return i.Button("DASH") }
func (i *InputControl) Info() error         { return i.Button("INFO") }
func (i *InputControl) Exit() error         { return i.Button("EXIT") }
func (i *InputControl) Mute() error         { return i.Button("MUTE") }
func (i *InputControl) Red() error          { return i.Button("RED") }
func (i *InputControl) Green() error        { return i.Button("GREEN") }
func (i *InputControl) Yellow() error       { return i.Button("YELLOW") }
func (i *InputControl) Blue() error         { return i.Button("BLUE") }
func (i *InputControl) VolumeUp() error     { return i.Button("VOLUMEUP") }
func (i *InputControl) VolumeDown() error   { return i.Button("VOLUMEDOWN") }
func (i *InputControl) ChannelUp() error    { return i.Button("CHANNELUP") }
func (i *InputControl) ChannelDown() error  { return i.Button("CHANNELDOWN") }
func (i *InputControl) Play() error         { return i.Button("PLAY") }
func (i *InputControl) Pause() error        { return i.Button("PAUSE") }
func (i *InputControl) Stop() error         { return i.Button("STOP") }
func (i *InputControl) Rewind() error       { return i.Button("REWIND") }
func (i *InputControl) FastForward() error  { return i.Button("FASTFORWARD") }
