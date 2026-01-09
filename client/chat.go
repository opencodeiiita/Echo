package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

func connectToEchoServer(serverURL string, username string, password string) error {
	u := url.URL{Scheme: "ws", Host: serverURL, Path: "/"}
	fmt.Printf("Connecting to %s\n", u.String())
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("[%s] ✗ Failed to connect to server: %v", getTimestamp(), err)

	}

	defer c.Close()

	fmt.Printf("[%s] ✓ Connected to server\n", getTimestamp())
	
	// Create authentication JSON
	authData := map[string]string{
		"username": username,
		"password": password,
	}
	authJSON, err := json.Marshal(authData)
	if err != nil {
		return fmt.Errorf("Failed to create authentication data: %v", err)
	}
	
	err = c.WriteMessage(websocket.TextMessage, authJSON)
	if err != nil {
		return fmt.Errorf("Failed to write to server: %v", err)

	}
	fmt.Printf("[%s] ✓ Authentication data sent for user: %s\n", getTimestamp(), username)
	fmt.Println("-------------------------------------------")
	fmt.Println("Waiting for authentication response...")

	// Wait for authentication response
	messageType, data, err := c.ReadMessage()
	if err != nil {
		return fmt.Errorf("[%s] ✗ Connection error: %v", getTimestamp(), err)
	}

	if messageType == websocket.TextMessage {
		message := string(data)
		if strings.HasPrefix(message, "ERROR:") {
			fmt.Printf("\r%s%s\n", "\033[K", message)
			return fmt.Errorf("Authentication failed: %s", message)
		} else {
			fmt.Printf("\r%s%s\n", "\033[K", message)
			fmt.Printf("[%s] ✓ Authentication successful!\n", getTimestamp())
		}
	}

	fmt.Println("Listening for messages from server...")

	// Run a goroutine so incoming messages are received in background while user types
	go func() {
		for {
			messageType, data, err := c.ReadMessage()
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				fmt.Printf("\r%sConnection closed by server.\n", "\033[K")
				os.Exit(1)
				return
			}
			if err != nil {
				fmt.Printf("\r%sConnection error: %v\n", "\033[K", err)
				os.Exit(1)
				break
			}
			if messageType == websocket.TextMessage {
				message := string(data)
				// We print the message directly to avoid double timestamps/prefixes
				fmt.Printf("\r%s%s\nEnter Message : ", "\033[K", message)
			}
		}
	}()

	// Read for terminal input
	reader := bufio.NewReader(os.Stdin)

	// Infinite loop so that user can keep sending messages
	for {
		fmt.Print("Enter Message : ")
		text, _ := reader.ReadString('\n')
		text = strings.TrimSpace(text)
		c.WriteMessage(websocket.TextMessage, []byte(text))
	}
	return nil

}

func getUsername() string {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Enter your username: ")
		username, _ := reader.ReadString('\n')
		username = strings.TrimSpace(username)
		
		if username == "" {
			fmt.Println("Username cannot be empty. Please try again.")
			continue
		}
		
		return username
	}
}

func getTimestamp() string {
	return time.Now().Format("02/01/2006 03:04:05 PM")
}

func getServerAddress() string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter Server Address (default: localhost:8080): ")
	address, _ := reader.ReadString('\n')
	address = strings.TrimSpace(address)
	
	// If the user just hits enter, default to localhost:8080
	if address == "" {
		return "localhost:8080"
	}
	return address
}

func getPassword() string {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Enter your password: ")
		password, _ := reader.ReadString('\n')
		password = strings.TrimSpace(password)
		
		if password == "" {
			fmt.Println("Password cannot be empty. Please try again.")
			continue
		}
		
		return password
	}
}