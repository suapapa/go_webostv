package webostv

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	// DefaultHandshakeTimeout is the default timeout for websocket handshake.
	DefaultHandshakeTimeout = 10 * time.Second
	// DefaultWriteWait is the default time to wait for a message to be written.
	DefaultWriteWait = 10 * time.Second
)

var (
	ErrConnectionClosed = errors.New("connection closed")
	ErrTimeout          = errors.New("timeout")
	ErrInvalidResponse  = errors.New("invalid response from TV")
)

// RegistrationStatus represents the status during the registration process.
type RegistrationStatus int

const (
	Prompted RegistrationStatus = iota + 1
	Registered
)

// Message is the basic unit of communication with WebOS TV.
type Message struct {
	Type    string          `json:"type"`
	ID      string          `json:"id"`
	URI     string          `json:"uri,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// Client is a WebOS TV client.
type Client struct {
	url string
	ctx context.Context
	cancel context.CancelFunc

	conn *websocket.Conn
	connMu sync.Mutex

	waiters    map[string]chan *Message
	waiterMu   sync.RWMutex

	subscribers map[string]subscription
	subscriberMu sync.RWMutex

	writeMu sync.Mutex

	options clientOptions
}

type subscription struct {
	uri      string
	callback func(json.RawMessage)
}

type clientOptions struct {
	handshakeTimeout time.Duration
	secure           bool
}

// Option is a functional option for Client.
type Option func(*clientOptions)

// WithSecure sets whether to use a secure (WSS) connection.
func WithSecure(secure bool) Option {
	return func(o *clientOptions) {
		o.secure = secure
	}
}

// WithHandshakeTimeout sets the timeout for the websocket handshake.
func WithHandshakeTimeout(d time.Duration) Option {
	return func(o *clientOptions) {
		o.handshakeTimeout = d
	}
}

// NewClient creates a new WebOS TV client.
func NewClient(host string, opts ...Option) *Client {
	options := clientOptions{
		handshakeTimeout: DefaultHandshakeTimeout,
		secure:           false,
	}
	for _, opt := range opts {
		opt(&options)
	}

	var wsURL string
	if options.secure {
		if strings.Contains(host, ":") {
			wsURL = fmt.Sprintf("wss://%s/", host)
		} else {
			wsURL = fmt.Sprintf("wss://%s:3001/", host)
		}
	} else {
		if strings.Contains(host, ":") {
			wsURL = fmt.Sprintf("ws://%s/", host)
		} else {
			wsURL = fmt.Sprintf("ws://%s:3000/", host)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Client{
		url:         wsURL,
		ctx:         ctx,
		cancel:      cancel,
		waiters:     make(map[string]chan *Message),
		subscribers: make(map[string]subscription),
		options:     options,
	}
}

// Connect establishes a connection to the TV.
func (c *Client) Connect() error {
	c.connMu.Lock()
	defer c.connMu.Unlock()

	if c.conn != nil {
		return nil
	}

	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = c.options.handshakeTimeout

	conn, _, err := dialer.Dial(c.url, nil)
	if err != nil {
		return fmt.Errorf("failed to dial %s: %w", c.url, err)
	}
	c.conn = conn

	go c.readLoop()

	return nil
}

// Close closes the connection to the TV.
func (c *Client) Close() error {
	c.cancel()
	c.connMu.Lock()
	defer c.connMu.Unlock()

	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}

// URL returns the URL the client is connected to.
func (c *Client) URL() string {
	return c.url
}

func (c *Client) readLoop() {
	defer func() {
		c.waiterMu.Lock()
		for id, ch := range c.waiters {
			close(ch)
			delete(c.waiters, id)
		}
		c.waiterMu.Unlock()
		_ = c.Close()
	}()

	for {
		select {
		case <-c.ctx.Done():
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
	c.waiterMu.RLock()
	ch, ok := c.waiters[msg.ID]
	c.waiterMu.RUnlock()

	if ok {
		select {
		case ch <- msg:
		case <-c.ctx.Done():
		default:
			// channel full, drop message or handle overflow
		}
		return
	}

	c.subscriberMu.RLock()
	sub, ok := c.subscribers[msg.ID]
	c.subscriberMu.RUnlock()

	if ok {
		go sub.callback(msg.Payload)
	}
}

// Register performs the registration with the TV.
func (c *Client) Register(ctx context.Context, store map[string]string) (<-chan RegistrationStatus, <-chan error) {
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

		respChan, _, err := c.SendMessage(ctx, "", "register", "", payload)
		if err != nil {
			errChan <- fmt.Errorf("failed to send registration message: %w", err)
			return
		}

		for {
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			case <-c.ctx.Done():
				errChan <- ErrConnectionClosed
				return
			case item, ok := <-respChan:
				if !ok {
					errChan <- ErrConnectionClosed
					return
				}

				var p struct {
					PairingType string `json:"pairingType"`
					ClientKey   string `json:"client-key"`
					Error       string `json:"error"`
				}
				if err := json.Unmarshal(item.Payload, &p); err != nil {
					errChan <- fmt.Errorf("failed to unmarshal registration payload: %w", err)
					return
				}

				if p.PairingType == "PROMPT" {
					select {
					case statusChan <- Prompted:
					case <-ctx.Done():
						return
					}
				} else if item.Type == "registered" {
					store["client_key"] = p.ClientKey
					select {
					case statusChan <- Registered:
					case <-ctx.Done():
					}
					return
				} else if item.Type == "error" {
					errChan <- fmt.Errorf("registration error: %s", p.Error)
					return
				}
			}
		}
	}()

	return statusChan, errChan
}

// Request sends a request and waits for a single response.
func (c *Client) Request(ctx context.Context, uri string, payload interface{}) (*Message, error) {
	respChan, id, err := c.SendMessage(ctx, "", "request", uri, payload)
	if err != nil {
		return nil, err
	}
	defer func() {
		c.waiterMu.Lock()
		delete(c.waiters, id)
		c.waiterMu.Unlock()
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.ctx.Done():
		return nil, ErrConnectionClosed
	case resp, ok := <-respChan:
		if !ok {
			return nil, ErrConnectionClosed
		}
		if resp.Type == "error" {
			return nil, fmt.Errorf("%w: %s", ErrInvalidResponse, string(resp.Payload))
		}
		return resp, nil
	}
}

// SendMessage sends a message to the TV. If id is empty, a new UUID is generated.
func (c *Client) SendMessage(ctx context.Context, id, requestType, uri string, payload interface{}) (<-chan *Message, string, error) {
	if id == "" {
		id = uuid.New().String()
	}
	
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	msg := Message{
		Type:    requestType,
		ID:      id,
		Payload: rawPayload,
	}
	if uri != "" {
		msg.URI = uri
	}

	ch := make(chan *Message, 10)
	c.waiterMu.Lock()
	c.waiters[id] = ch
	c.waiterMu.Unlock()

	c.writeMu.Lock()
	err = c.conn.WriteJSON(msg)
	c.writeMu.Unlock()

	if err != nil {
		c.waiterMu.Lock()
		delete(c.waiters, id)
		c.waiterMu.Unlock()
		return nil, "", fmt.Errorf("failed to write JSON: %w", err)
	}

	return ch, id, nil
}

// Subscribe subscribes to a URI.
func (c *Client) Subscribe(uri string, callback func(json.RawMessage)) (string, error) {
	id := uuid.New().String()

	c.subscriberMu.Lock()
	c.subscribers[id] = subscription{
		uri:      uri,
		callback: callback,
	}
	c.subscriberMu.Unlock()

	_, _, err := c.SendMessage(context.Background(), id, "subscribe", uri, nil)
	if err != nil {
		c.subscriberMu.Lock()
		delete(c.subscribers, id)
		c.subscriberMu.Unlock()
		return "", fmt.Errorf("failed to send subscribe message: %w", err)
	}

	// For subscriptions, we don't want to use the waiter mechanism after the initial send,
	// because we want all responses (including the first one) to go to the subscriber callback.
	c.waiterMu.Lock()
	delete(c.waiters, id)
	c.waiterMu.Unlock()

	return id, nil
}

// Unsubscribe removes a subscription.
func (c *Client) Unsubscribe(id string) error {
	c.subscriberMu.Lock()
	sub, ok := c.subscribers[id]
	if ok {
		delete(c.subscribers, id)
	}
	c.subscriberMu.Unlock()

	if !ok {
		return errors.New("subscription not found")
	}

	_, _, err := c.SendMessage(context.Background(), "", "unsubscribe", sub.uri, nil)
	if err != nil {
		return fmt.Errorf("failed to send unsubscribe message: %w", err)
	}
	return nil
}
