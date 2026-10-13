package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
)

const (
	sendBufferSize = 64
	pingInterval   = 25 * time.Second
)

// Client represents an active WebSocket connection attached to a specific collaborator and room.
type Client struct {
	id        uuid.UUID
	userID    uuid.UUID
	userEmail string
	projectID uuid.UUID
	conn      *websocket.Conn
	room      *Room
	sendChan  chan []byte
	ctx       context.Context
	cancel    context.CancelFunc
	closeOnce sync.Once
}

// NewClient creates a new Client instance.
func NewClient(
	userID uuid.UUID,
	userEmail string,
	projectID uuid.UUID,
	conn *websocket.Conn,
	room *Room,
) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{
		id:        uuid.New(),
		userID:    userID,
		userEmail: userEmail,
		projectID: projectID,
		conn:      conn,
		room:      room,
		sendChan:  make(chan []byte, sendBufferSize),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// ID returns the unique connection ID of this client.
func (c *Client) ID() uuid.UUID {
	return c.id
}

// UserID returns the user ID of the connected collaborator.
func (c *Client) UserID() uuid.UUID {
	return c.userID
}

// UserEmail returns the email of the connected collaborator.
func (c *Client) UserEmail() string {
	return c.userEmail
}

// ProjectID returns the project ID associated with this client.
func (c *Client) ProjectID() uuid.UUID {
	return c.projectID
}

// Room returns the project room this client belongs to.
func (c *Client) Room() *Room {
	return c.room
}

// Send queues a Message envelope to be written to the WebSocket connection.
func (c *Client) Send(msg Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal client message: %w", err)
	}

	if ctxErr := c.ctx.Err(); ctxErr != nil {
		return fmt.Errorf("client context done: %w", ctxErr)
	}

	select {
	case <-c.ctx.Done():
		return fmt.Errorf("client context done: %w", c.ctx.Err())
	case c.sendChan <- data:
		return nil
	default:
		return errors.New("client send channel buffer full")
	}
}

// Close gracefully closes the client connection and cancels its context.
func (c *Client) Close(status websocket.StatusCode, reason string) {
	c.closeOnce.Do(func() {
		c.cancel()
		if c.conn != nil {
			_ = c.conn.Close(status, reason)
		}
	})
}

// ReadMessage reads the next message from the WebSocket connection in a direct, blocking call.
func (c *Client) ReadMessage() (Message, error) {
	for {
		_, data, err := c.conn.Read(c.ctx)
		if err != nil {
			return Message{}, fmt.Errorf("read websocket message: %w", err)
		}

		var msg Message
		if unmarshalErr := json.Unmarshal(data, &msg); unmarshalErr != nil {
			continue
		}

		if msg.Type == MsgPing {
			c.replyPong()
			continue
		}

		return msg, nil
	}
}

func (c *Client) replyPong() {
	pongMsg, err := NewMessage(MsgPong, nil)
	if err == nil {
		_ = c.Send(pongMsg)
	}
}

// WritePump writes queued messages and periodic ping frames to the WebSocket connection.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingInterval)
	defer func() {
		ticker.Stop()
		c.Close(websocket.StatusNormalClosure, "write pump finished")
	}()

	for {
		select {
		case <-c.ctx.Done():
			return
		case data, ok := <-c.sendChan:
			if !ok || !c.writeMessage(data) {
				return
			}
		case <-ticker.C:
			if !c.sendPing() {
				return
			}
		}
	}
}

func (c *Client) writeMessage(data []byte) bool {
	return c.conn.Write(c.ctx, websocket.MessageText, data) == nil
}

func (c *Client) sendPing() bool {
	return c.conn.Ping(c.ctx) == nil
}
