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

// AuthCredentials represents user login information
type AuthCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func connectToEchoServer(serverURL string, username string, password string) error {
	// Build WebSocket URL
	wsURL := url.URL{Scheme: "ws", Host: serverURL, Path: "/"}
	fmt.Printf("Connecting to %s\n", wsURL.String())

	// Establish connection
	conn, _, err := websocket.DefaultDialer.Dial(wsURL.String(), nil)
	if err != nil {
		return fmt.Errorf("[%s] ✗ Connection failed: %v", getTimestamp(), err)
	}
	defer conn.Close()

	fmt.Printf("[%s] ✓ Connected successfully\n", getTimestamp())

	// Prepare authentication payload - FIXED VERSION
	authData := map[string]string{
		"username": username,
		"password": password,
	}

	authJSON, err := json.Marshal(authData)
	if err != nil {
		return fmt.Errorf("Failed to serialize authentication data: %v", err)
	}

	// Debug: Show what we're sending
	fmt.Printf("Sending auth data: %s\n", string(authJSON))

	// Send authentication credentials
	err = conn.WriteMessage(websocket.TextMessage, authJSON)
	if err != nil {
		return fmt.Errorf("Failed to send credentials: %v", err)
	}

	fmt.Printf("[%s] ✓ Credentials sent for user: %s\n", getTimestamp(), username)
	fmt.Println("-------------------------------------------")
	fmt.Println("Awaiting server response...")

	// Wait for authentication response
	_, responseData, err := conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("[%s] ✗ Server response error: %v", getTimestamp(), err)
	}

	serverResponse := string(responseData)

	// Check if authentication failed
	if strings.HasPrefix(serverResponse, "ERROR:") {
		fmt.Printf("\r%s%s\n", "\033[K", serverResponse)
		return fmt.Errorf("Authentication rejected by server")
	}

	// Authentication successful
	fmt.Printf("\r%s%s\n", "\033[K", serverResponse)
	fmt.Printf("[%s] ✓ Authentication successful!\n", getTimestamp())
	fmt.Println("-------------------------------------------")
	fmt.Println("You can now send messages. Type /quit to exit.")
	fmt.Println()

	// Start background goroutine for receiving messages
	go receiveMessagesInBackground(conn)

	// Main loop for sending messages
	sendMessagesFromTerminal(conn)

	return nil
}


// receiveMessagesInBackground listens for incoming messages
func receiveMessagesInBackground(conn *websocket.Conn) {
	for {
		msgType, data, err := conn.ReadMessage()

		if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			fmt.Printf("\r%sServer closed the connection.\n", "\033[K")
			os.Exit(1)
			return
		}

		if err != nil {
			fmt.Printf("\r%sConnection error: %v\n", "\033[K", err)
			os.Exit(1)
			return
		}

		if msgType == websocket.TextMessage {
			message := string(data)
			// Clear current line and print message
			fmt.Printf("\r%s%s\nEnter Message: ", "\033[K", message)
		}
	}
}

// sendMessagesFromTerminal handles user input
func sendMessagesFromTerminal(conn *websocket.Conn) {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("Enter Message: ")
		if !scanner.Scan() {
			break
		}

		userInput := strings.TrimSpace(scanner.Text())

		// Check for quit command
		if userInput == "/quit" {
			fmt.Println("Disconnecting...")
			break
		}

		// Send non-empty messages
		if userInput != "" {
			err := conn.WriteMessage(websocket.TextMessage, []byte(userInput))
			if err != nil {
				fmt.Printf("Failed to send message: %v\n", err)
				break
			}
		}
	}
}

// getUsername prompts for and validates username
func getUsername() string {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Enter your username: ")
		username, _ := reader.ReadString('\n')
		username = strings.TrimSpace(username)

		if username == "" {
			fmt.Println("⚠ Username cannot be empty. Please try again.")
			continue
		}

		return username
	}
}

// getPassword prompts for and validates password
func getPassword() string {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Enter your password: ")
		password, _ := reader.ReadString('\n')
		password = strings.TrimSpace(password)

		if password == "" {
			fmt.Println("⚠ Password cannot be empty. Please try again.")
			continue
		}

		return password
	}
}

// getServerAddress prompts for server address with default
func getServerAddress() string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter Server Address (default: localhost:8080): ")
	address, _ := reader.ReadString('\n')
	address = strings.TrimSpace(address)

	if address == "" {
		return "localhost:8080"
	}

	return address
}

// getTimestamp returns formatted current time
func getTimestamp() string {
	return time.Now().Format("02/01/2006 03:04:05 PM")
}
