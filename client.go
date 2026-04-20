package webostv

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	Signature = "eyJhbGdvcml0aG0iOiJSU0EtU0hBMjU2Iiwia2V5SWQiOiJ0ZXN0LXNpZ25pbm" +
		"ctY2VydCIsInNpZ25hdHVyZVZlcnNpb24iOjF9.hrVRgjCwXVvE2OOSpDZ58hR" +
		"+59aFNwYDyjQgKk3auukd7pcegmE2CzPCa0bJ0ZsRAcKkCTJrWo5iDzNhMBWRy" +
		"aMOv5zWSrthlf7G128qvIlpMT0YNY+n/FaOHE73uLrS/g7swl3/qH/BGFG2Hu4" +
		"RlL48eb3lLKqTt2xKHdCs6Cd4RMfJPYnzgvI4BNrFUKsjkcu+WD4OO2A27Pq1n" +
		"50cMchmcaXadJhGrOqH5YmHdOCj5NSHzJYrsW0HPlpuAx/ECMeIZYDh6RMqaFM" +
		"2DXzdKX9NmmyqzJ3o/0lkk/N97gfVRLW5hA29yeAwaCViZNCP8iC9aO0q9fQoj" +
		"oa7NQnAtw=="
)

var RegistrationPayload = map[string]interface{}{
	"forcePairing": false,
	"manifest": map[string]interface{}{
		"appVersion":      "1.1",
		"manifestVersion": 1,
		"permissions": []string{
			"LAUNCH", "LAUNCH_WEBAPP", "APP_TO_APP", "CLOSE", "TEST_OPEN", "TEST_PROTECTED",
			"CONTROL_AUDIO", "CONTROL_DISPLAY", "CONTROL_INPUT_JOYSTICK", "CONTROL_INPUT_MEDIA_RECORDING",
			"CONTROL_INPUT_MEDIA_PLAYBACK", "CONTROL_INPUT_TV", "CONTROL_POWER", "READ_APP_STATUS",
			"READ_CURRENT_CHANNEL", "READ_INPUT_DEVICE_LIST", "READ_NETWORK_STATE", "READ_RUNNING_APPS",
			"READ_TV_CHANNEL_LIST", "WRITE_NOTIFICATION_TOAST", "READ_POWER_STATE", "READ_COUNTRY_INFO",
			"READ_SETTINGS", "CONTROL_TV_SCREEN", "CONTROL_TV_STANBY", "CONTROL_FAVORITE_GROUP",
			"CONTROL_USER_INFO", "CHECK_BLUETOOTH_DEVICE", "CONTROL_BLUETOOTH", "CONTROL_TIMER_INFO",
			"STB_INTERNAL_CONNECTION", "CONTROL_RECORDING", "READ_RECORDING_STATE", "WRITE_RECORDING_LIST",
			"READ_RECORDING_LIST", "READ_RECORDING_SCHEDULE", "WRITE_RECORDING_SCHEDULE", "READ_STORAGE_DEVICE_LIST",
			"READ_TV_PROGRAM_INFO", "CONTROL_BOX_CHANNEL", "READ_TV_ACR_AUTH_TOKEN", "READ_TV_CONTENT_STATE",
			"READ_TV_CURRENT_TIME", "ADD_LAUNCHER_CHANNEL", "SET_CHANNEL_SKIP", "RELEASE_CHANNEL_SKIP",
			"CONTROL_CHANNEL_BLOCK", "DELETE_SELECT_CHANNEL", "CONTROL_CHANNEL_GROUP", "SCAN_TV_CHANNELS",
			"CONTROL_TV_POWER", "CONTROL_WOL",
		},
		"signatures": []map[string]interface{}{
			{
				"signature":        Signature,
				"signatureVersion": 1,
			},
		},
		"signed": map[string]interface{}{
			"appId":   "com.lge.test",
			"created": "20140509",
			"localizedAppNames": map[string]string{
				"":      "LG Remote App",
				"ko-KR": "리모컨 앱",
				"zxx-XX": "ЛГ Rэмotэ AПП",
			},
			"localizedVendorNames": map[string]string{
				"": "LG Electronics",
			},
			"permissions": []string{
				"TEST_SECURE", "CONTROL_INPUT_TEXT", "CONTROL_MOUSE_AND_KEYBOARD", "READ_INSTALLED_APPS",
				"READ_LGE_SDX", "READ_NOTIFICATIONS", "SEARCH", "WRITE_SETTINGS", "WRITE_SETTINGS",
				"WRITE_NOTIFICATION_ALERT", "CONTROL_POWER", "READ_CURRENT_CHANNEL", "READ_RUNNING_APPS",
				"READ_UPDATE_INFO", "UPDATE_FROM_REMOTE_APP", "READ_LGE_TV_INPUT_EVENTS", "READ_TV_CURRENT_TIME",
			},
			"serial":   "2f930e2d2cfe083771f68e4fe7bb07",
			"vendorId": "com.lge",
		},
	},
	"pairingType": "PROMPT",
}

type RegistrationStatus int

const (
	Prompted RegistrationStatus = iota + 1
	Registered
)

type Message struct {
	Type    string      `json:"type"`
	ID      string      `json:"id"`
	URI     string      `json:"uri,omitempty"`
	Payload interface{} `json:"payload,omitempty"`
}

type Client struct {
	URL  string
	conn *websocket.Conn

	waiters      map[string]chan *Message
	waiterLock   sync.Mutex
	subscribers  map[string]subscription
	subscriberLock sync.Mutex

	sendLock sync.Mutex

	done chan struct{}
}

type subscription struct {
	uri      string
	callback func(interface{})
}

func NewClient(host string, secure bool) *Client {
	var wsURL string
	if secure {
		wsURL = fmt.Sprintf("wss://%s:3001/", host)
	} else {
		wsURL = fmt.Sprintf("ws://%s:3000/", host)
	}

	return &Client{
		URL:         wsURL,
		waiters:     make(map[string]chan *Message),
		subscribers: make(map[string]subscription),
		done:        make(chan struct{}),
	}
}

func (c *Client) Connect() error {
	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 10 * time.Second

	conn, _, err := dialer.Dial(c.URL, nil)
	if err != nil {
		return err
	}
	c.conn = conn

	go c.readLoop()

	return nil
}

func (c *Client) Close() error {
	close(c.done)
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Client) readLoop() {
	defer func() {
		c.waiterLock.Lock()
		for id, ch := range c.waiters {
			close(ch)
			delete(c.waiters, id)
		}
		c.waiterLock.Unlock()
	}()

	for {
		select {
		case <-c.done:
			return
		default:
			_, data, err := c.conn.ReadMessage()
			if err != nil {
				return
			}

			var msg Message
			if err := json.Unmarshal(data, &msg); err != nil {
				continue
			}

			c.handleMessage(&msg)
		}
	}
}

func (c *Client) handleMessage(msg *Message) {
	c.waiterLock.Lock()
	ch, ok := c.waiters[msg.ID]
	c.waiterLock.Unlock()

	if ok {
		select {
		case ch <- msg:
		default:
		}
		return
	}

	c.subscriberLock.Lock()
	sub, ok := c.subscribers[msg.ID]
	c.subscriberLock.Unlock()

	if ok {
		sub.callback(msg.Payload)
	}
}

func (c *Client) Register(store map[string]string) (<-chan RegistrationStatus, <-chan error) {
	statusChan := make(chan RegistrationStatus)
	errChan := make(chan error, 1)

	payload := make(map[string]interface{})
	for k, v := range RegistrationPayload {
		payload[k] = v
	}

	if key, ok := store["client_key"]; ok {
		payload["client-key"] = key
	}

	go func() {
		defer close(statusChan)
		defer close(errChan)

		respChan, err := c.SendMessage("register", "", payload)
		if err != nil {
			errChan <- err
			return
		}

		for {
			select {
			case item := <-respChan:
				if item == nil {
					errChan <- errors.New("connection closed")
					return
				}

				p, ok := item.Payload.(map[string]interface{})
				if !ok {
					errChan <- errors.New("invalid payload")
					return
				}

				if p["pairingType"] == "PROMPT" {
					statusChan <- Prompted
				} else if item.Type == "registered" {
					store["client_key"] = p["client-key"].(string)
					statusChan <- Registered
					return
				} else if item.Type == "error" {
					errChan <- fmt.Errorf("registration error: %v", p["error"])
					return
				}
			case <-time.After(60 * time.Second):
				errChan <- errors.New("timeout")
				return
			}
		}
	}()

	return statusChan, errChan
}

func (c *Client) Request(uri string, payload interface{}, timeout time.Duration) (*Message, error) {
	respChan, err := c.SendMessage("request", uri, payload)
	if err != nil {
		return nil, err
	}

	select {
	case resp := <-respChan:
		if resp == nil {
			return nil, errors.New("connection closed")
		}
		if resp.Type == "error" {
			return nil, fmt.Errorf("error from tv: %v", resp.Payload)
		}
		return resp, nil
	case <-time.After(timeout):
		return nil, errors.New("timeout")
	}
}

func (c *Client) SendMessage(requestType, uri string, payload interface{}) (<-chan *Message, error) {
	id := uuid.New().String()
	msg := Message{
		Type:    requestType,
		ID:      id,
		Payload: payload,
	}
	if uri != "" {
		msg.URI = uri
	}

	ch := make(chan *Message, 10)
	c.waiterLock.Lock()
	c.waiters[id] = ch
	c.waiterLock.Unlock()

	c.sendLock.Lock()
	err := c.conn.WriteJSON(msg)
	c.sendLock.Unlock()

	if err != nil {
		c.waiterLock.Lock()
		delete(c.waiters, id)
		c.waiterLock.Unlock()
		return nil, err
	}

	return ch, nil
}

func (c *Client) Subscribe(uri string, callback func(interface{})) (string, error) {
	id := uuid.New().String()

	c.subscriberLock.Lock()
	c.subscribers[id] = subscription{
		uri:      uri,
		callback: callback,
	}
	c.subscriberLock.Unlock()

	_, err := c.SendMessage("subscribe", uri, nil)
	if err != nil {
		c.subscriberLock.Lock()
		delete(c.subscribers, id)
		c.subscriberLock.Unlock()
		return "", err
	}

	return id, nil
}

func (c *Client) Unsubscribe(id string) error {
	c.subscriberLock.Lock()
	sub, ok := c.subscribers[id]
	if ok {
		delete(c.subscribers, id)
	}
	c.subscriberLock.Unlock()

	if !ok {
		return errors.New("subscription not found")
	}

	_, err := c.SendMessage("unsubscribe", sub.uri, nil)
	return err
}
