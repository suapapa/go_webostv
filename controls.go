package webostv

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type Control struct {
	Client *Client
}

// MediaControl
type MediaControl struct {
	Control
}

func (m *MediaControl) VolumeUp() error {
	_, err := m.Client.Request("ssap://audio/volumeUp", nil, 5*time.Second)
	return err
}

func (m *MediaControl) VolumeDown() error {
	_, err := m.Client.Request("ssap://audio/volumeDown", nil, 5*time.Second)
	return err
}

func (m *MediaControl) GetVolume() (map[string]interface{}, error) {
	resp, err := m.Client.Request("ssap://audio/getVolume", nil, 5*time.Second)
	if err != nil {
		return nil, err
	}
	p, ok := resp.Payload.(map[string]interface{})
	if !ok {
		return nil, errors.New("invalid response")
	}
	return p, nil
}

func (m *MediaControl) SetVolume(volume int) error {
	payload := map[string]interface{}{"volume": volume}
	_, err := m.Client.Request("ssap://audio/setVolume", payload, 5*time.Second)
	return err
}

func (m *MediaControl) Mute(mute bool) error {
	payload := map[string]interface{}{"mute": mute}
	_, err := m.Client.Request("ssap://audio/setMute", payload, 5*time.Second)
	return err
}

func (m *MediaControl) Play() error {
	_, err := m.Client.Request("ssap://media.controls/play", nil, 5*time.Second)
	return err
}

func (m *MediaControl) Pause() error {
	_, err := m.Client.Request("ssap://media.controls/pause", nil, 5*time.Second)
	return err
}

func (m *MediaControl) Stop() error {
	_, err := m.Client.Request("ssap://media.controls/stop", nil, 5*time.Second)
	return err
}

func (m *MediaControl) Rewind() error {
	_, err := m.Client.Request("ssap://media.controls/rewind", nil, 5*time.Second)
	return err
}

func (m *MediaControl) FastForward() error {
	_, err := m.Client.Request("ssap://media.controls/fastForward", nil, 5*time.Second)
	return err
}

func (m *MediaControl) ListAudioOutputSources() []AudioOutputSource {
	sources := []string{"tv_speaker", "external_speaker", "soundbar", "bt_soundbar", "tv_external_speaker"}
	res := make([]AudioOutputSource, len(sources))
	for i, s := range sources {
		res[i] = AudioOutputSource{Source: s}
	}
	return res
}

func (m *MediaControl) GetAudioOutput() (*AudioOutputSource, error) {
	resp, err := m.Client.Request("ssap://audio/getSoundOutput", nil, 5*time.Second)
	if err != nil {
		return nil, err
	}
	p, ok := resp.Payload.(map[string]interface{})
	if !ok {
		return nil, errors.New("invalid response")
	}
	return &AudioOutputSource{Source: p["soundOutput"].(string)}, nil
}

func (m *MediaControl) SetAudioOutput(source AudioOutputSource) error {
	payload := map[string]interface{}{"output": source.Source}
	_, err := m.Client.Request("ssap://audio/changeSoundOutput", payload, 5*time.Second)
	return err
}

// Subscriptions
func (m *MediaControl) SubscribeVolume(callback func(map[string]interface{})) (string, error) {
	return m.Client.Subscribe("ssap://audio/getVolume", func(p interface{}) {
		if data, ok := p.(map[string]interface{}); ok {
			callback(data)
		}
	})
}

// SystemControl
type SystemControl struct {
	Control
}

func (s *SystemControl) PowerOff() error {
	_, err := s.Client.Request("ssap://system/turnOff", nil, 5*time.Second)
	return err
}

func (s *SystemControl) ScreenOff() error {
	payload := map[string]interface{}{"standbyMode": "active"}
	_, err := s.Client.Request("ssap://com.webos.service.tvpower/power/turnOffScreen", payload, 5*time.Second)
	return err
}

func (s *SystemControl) ScreenOn() error {
	payload := map[string]interface{}{"standbyMode": "active"}
	_, err := s.Client.Request("ssap://com.webos.service.tvpower/power/turnOnScreen", payload, 5*time.Second)
	return err
}

func (s *SystemControl) Info() (map[string]interface{}, error) {
	resp, err := s.Client.Request("ssap://com.webos.service.update/getCurrentSWInformation", nil, 5*time.Second)
	if err != nil {
		return nil, err
	}
	return resp.Payload.(map[string]interface{}), nil
}

func (s *SystemControl) Notify(message string, iconBytes []byte, iconExt string) error {
	payload := map[string]interface{}{
		"message": message,
	}
	if iconBytes != nil {
		payload["iconData"] = base64.StdEncoding.EncodeToString(iconBytes)
		payload["iconExtension"] = iconExt
	}
	_, err := s.Client.Request("ssap://system.notifications/createToast", payload, 5*time.Second)
	return err
}

// ApplicationControl
type ApplicationControl struct {
	Control
}

func (a *ApplicationControl) ListApps() ([]Application, error) {
	resp, err := a.Client.Request("ssap://com.webos.applicationManager/listApps", nil, 5*time.Second)
	if err != nil {
		return nil, err
	}
	p := resp.Payload.(map[string]interface{})
	appsData := p["apps"].([]interface{})
	apps := make([]Application, len(appsData))
	for i, v := range appsData {
		data := v.(map[string]interface{})
		apps[i] = Application{
			ID:    data["id"].(string),
			Title: data["title"].(string),
			Icon:  data["icon"].(string),
			Data:  data,
		}
	}
	return apps, nil
}

func (a *ApplicationControl) Launch(app Application, contentID string, params map[string]interface{}) error {
	payload := map[string]interface{}{
		"id": app.ID,
	}
	if contentID != "" {
		payload["contentId"] = contentID
	}
	if params != nil {
		payload["params"] = params
	}
	_, err := a.Client.Request("ssap://system.launcher/launch", payload, 5*time.Second)
	return err
}

func (a *ApplicationControl) GetCurrent() (string, error) {
	resp, err := a.Client.Request("ssap://com.webos.applicationManager/getForegroundAppInfo", nil, 5*time.Second)
	if err != nil {
		return "", err
	}
	p := resp.Payload.(map[string]interface{})
	return p["appId"].(string), nil
}

func (a *ApplicationControl) Close(launchInfo map[string]interface{}) error {
	_, err := a.Client.Request("ssap://system.launcher/close", launchInfo, 5*time.Second)
	return err
}

// TvControl
type TvControl struct {
	Control
}

func (t *TvControl) ChannelUp() error {
	_, err := t.Client.Request("ssap://tv/channelUp", nil, 5*time.Second)
	return err
}

func (t *TvControl) ChannelDown() error {
	_, err := t.Client.Request("ssap://tv/channelDown", nil, 5*time.Second)
	return err
}

func (t *TvControl) GetCurrentChannel() (map[string]interface{}, error) {
	resp, err := t.Client.Request("ssap://tv/getCurrentChannel", nil, 5*time.Second)
	if err != nil {
		return nil, err
	}
	return resp.Payload.(map[string]interface{}), nil
}

func (t *TvControl) ChannelList() (map[string]interface{}, error) {
	resp, err := t.Client.Request("ssap://tv/getChannelList", nil, 5*time.Second)
	if err != nil {
		return nil, err
	}
	return resp.Payload.(map[string]interface{}), nil
}

func (t *TvControl) SetChannelWithID(channelID string) error {
	payload := map[string]interface{}{"channelId": channelID}
	_, err := t.Client.Request("ssap://tv/openChannel", payload, 5*time.Second)
	return err
}

func (t *TvControl) GetCurrentProgram() (map[string]interface{}, error) {
	resp, err := t.Client.Request("ssap://tv/getChannelProgramInfo", nil, 5*time.Second)
	if err != nil {
		return nil, err
	}
	return resp.Payload.(map[string]interface{}), nil
}

// SourceControl
type SourceControl struct {
	Control
}

func (s *SourceControl) ListSources() ([]InputSource, error) {
	resp, err := s.Client.Request("ssap://tv/getExternalInputList", nil, 5*time.Second)
	if err != nil {
		return nil, err
	}
	p := resp.Payload.(map[string]interface{})
	sourcesData := p["devices"].([]interface{})
	sources := make([]InputSource, len(sourcesData))
	for i, v := range sourcesData {
		data := v.(map[string]interface{})
		sources[i] = InputSource{
			ID:    data["id"].(string),
			Label: data["label"].(string),
			Data:  data,
		}
	}
	return sources, nil
}

func (s *SourceControl) SetSource(source InputSource) error {
	payload := map[string]interface{}{"inputId": source.ID}
	_, err := s.Client.Request("ssap://tv/switchInput", payload, 5*time.Second)
	return err
}

// InputControl
type InputControl struct {
	Control
	mouseConn *websocket.Conn
}

func (i *InputControl) Type(text string) error {
	payload := map[string]interface{}{"text": text, "replace": 0}
	_, err := i.Client.Request("ssap://com.webos.service.ime/insertText", payload, 5*time.Second)
	return err
}

func (i *InputControl) Delete(count int) error {
	payload := map[string]interface{}{"count": count}
	_, err := i.Client.Request("ssap://com.webos.service.ime/deleteCharacters", payload, 5*time.Second)
	return err
}

func (i *InputControl) Enter() error {
	_, err := i.Client.Request("ssap://com.webos.service.ime/sendEnterKey", nil, 5*time.Second)
	return err
}

func (i *InputControl) ConnectInput() error {
	resp, err := i.Client.Request("ssap://com.webos.service.networkinput/getPointerInputSocket", nil, 5*time.Second)
	if err != nil {
		return err
	}
	p := resp.Payload.(map[string]interface{})
	sockPath := p["socketPath"].(string)
	if sockPath == "" {
		return errors.New("unable to connect to mouse")
	}

	conn, _, err := websocket.DefaultDialer.Dial(sockPath, nil)
	if err != nil {
		return err
	}
	i.mouseConn = conn
	return nil
}

func (i *InputControl) DisconnectInput() error {
	if i.mouseConn != nil {
		return i.mouseConn.Close()
	}
	return nil
}

func (i *InputControl) sendMouseCommand(cmd string, params ...interface{}) error {
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
func (i *InputControl) Left() error  { return i.Button("LEFT") }
func (i *InputControl) Right() error { return i.Button("RIGHT") }
func (i *InputControl) Up() error    { return i.Button("UP") }
func (i *InputControl) Down() error  { return i.Button("DOWN") }
func (i *InputControl) Home() error  { return i.Button("HOME") }
func (i *InputControl) Back() error  { return i.Button("BACK") }
func (i *InputControl) Menu() error  { return i.Button("MENU") }
func (i *InputControl) OK() error    { return i.Button("ENTER") }
func (i *InputControl) Dash() error  { return i.Button("DASH") }
func (i *InputControl) Info() error  { return i.Button("INFO") }
func (i *InputControl) Exit() error  { return i.Button("EXIT") }
func (i *InputControl) Mute() error  { return i.Button("MUTE") }
func (i *InputControl) Red() error   { return i.Button("RED") }
func (i *InputControl) Green() error { return i.Button("GREEN") }
func (i *InputControl) Yellow() error { return i.Button("YELLOW") }
func (i *InputControl) Blue() error  { return i.Button("BLUE") }
func (i *InputControl) VolumeUp() error { return i.Button("VOLUMEUP") }
func (i *InputControl) VolumeDown() error { return i.Button("VOLUMEDOWN") }
func (i *InputControl) ChannelUp() error { return i.Button("CHANNELUP") }
func (i *InputControl) ChannelDown() error { return i.Button("CHANNELDOWN") }
func (i *InputControl) Play() error { return i.Button("PLAY") }
func (i *InputControl) Pause() error { return i.Button("PAUSE") }
func (i *InputControl) Stop() error { return i.Button("STOP") }
func (i *InputControl) Rewind() error { return i.Button("REWIND") }
func (i *InputControl) FastForward() error { return i.Button("FASTFORWARD") }
