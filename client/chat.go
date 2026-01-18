package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type Message struct {
	Sender    string
	Content   string
	Timestamp string
}

type Session struct {
	Conn     *websocket.Conn
	Username string
	Inbound  chan Message
	Errors   chan error
}

func NewSession(serverURL, username, password string) (*Session, error) {
	u := url.URL{Scheme: "ws", Host: serverURL, Path: "/"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("[Error] ✗ Failed to connect to server: %v", err)
	}

	// Create authentication JSON
	authData := map[string]string{
		"username": username,
		"password": password,
	}
	authJSON, err := json.Marshal(authData)
	if err != nil {
		return nil, fmt.Errorf("failed to create authentication data: %v", err)
	}
	
	err = c.WriteMessage(websocket.TextMessage, authJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to write to server: %v", err)
	}

	// Wait for authentication response
	messageType, data, err := c.ReadMessage()
	if err != nil {
		return nil, fmt.Errorf("connection error: %v", err)
	}

	if messageType == websocket.TextMessage {
		message := string(data)
		if strings.HasPrefix(message, "ERROR:") {
			return nil, fmt.Errorf(message)
		}
	}

	s := &Session{
		Conn:     c,
		Username: username,
		Inbound:  make(chan Message, 100),
		Errors:   make(chan error, 10),
	}

	go s.readLoop()

	return s, nil
}

func (s *Session) readLoop() {
	for {
		messageType, data, err := s.Conn.ReadMessage()
		if err != nil {
			s.Errors <- err
			return
		}

		if messageType == websocket.TextMessage {
			raw := string(data)
			msg := Message{
				Sender:    "Server",
				Content:   raw,
				Timestamp: time.Now().Format("03:04:05 PM"),
			}

			// Parse: 1/19/2026, 12:40:00 AM: Alice said: Hello
			parts := strings.SplitN(raw, ": ", 2)
			if len(parts) == 2 {
				// parts[0] is time
				msg.Timestamp = parts[0]
				// parts[1] is "Alice said: Hello"
				subParts := strings.SplitN(parts[1], " said: ", 2)
				if len(subParts) == 2 {
					msg.Sender = subParts[0]
					msg.Content = subParts[1]
				}
			}

			s.Inbound <- msg
		}
	}
}

func (s *Session) Send(content string) error {
	return s.Conn.WriteMessage(websocket.TextMessage, []byte(content))
}

func (s *Session) Close() {
	if s.Conn != nil {
		s.Conn.Close()
	}
}