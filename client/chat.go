package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// connectWebsocket connects to the server and performs authentication, returning the connection.
func connectWebsocket(serverURL string, username string, password string) (*websocket.Conn, error) {
	u := url.URL{Scheme: "ws", Host: serverURL, Path: "/"}

	// Open the websocket connection
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %v", err)
	}

	// Create authentication JSON
	authData := map[string]string{
		"username": username,
		"password": password,
	}
	authJSON, err := json.Marshal(authData)
	if err != nil {
		c.Close()
		return nil, fmt.Errorf("failed to encode auth data: %v", err)
	}

	// Send authentication message
	err = c.WriteMessage(websocket.TextMessage, authJSON)
	if err != nil {
		c.Close()
		return nil, fmt.Errorf("failed to send auth data: %v", err)
	}

	// Wait for authentication response
	messageType, data, err := c.ReadMessage()
	if err != nil {
		c.Close()
		return nil, fmt.Errorf("connection error during auth: %v", err)
	}

	if messageType == websocket.TextMessage {
		message := string(data)
		if strings.HasPrefix(message, "ERROR:") {
			c.Close()
			return nil, fmt.Errorf("authentication failed: %s", message)
		}
	} else {
		// Optional: Handle non-text messages if expected, but for auth typically we expect text confirmation
	}

	// Return the successful connection
	return c, nil
}

func getTimestamp() string {
	return time.Now().Format("02/01/2006 03:04:05 PM")
}
