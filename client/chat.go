package main

import (
	"bufio"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

func connectToEchoServer(serverURL string, username string) error {
	u := url.URL{Scheme: "ws", Host: serverURL, Path: "/"}
	fmt.Printf("Connecting to %s\n", u.String())
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("[%s] ✗ Failed to connect to server: %v", getTimestamp(), err)

	}

	defer c.Close()

	fmt.Printf("[%s] ✓ Connected to server\n", getTimestamp())
	err = c.WriteMessage(websocket.TextMessage, []byte(username))
	if err != nil {
		return fmt.Errorf("Failed to write to server: %v", err)

	}
	fmt.Printf("[%s] ✓ Username sent: %s\n", getTimestamp(), username)
	fmt.Println("-------------------------------------------")
	fmt.Println("Listening for messages from server...")

	// Run a goroutine so incoming messages are received in background while user types
	go func() {
		for {
			messageType, data, err := c.ReadMessage()
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
				log.Println(err)
				return
			}
			if err != nil {
				fmt.Printf("error: %v\n", err)
				break
			}
			if messageType == websocket.TextMessage {
				timestamp := getTimestamp()
				message := string(data)
				fmt.Printf("\r%s[%s] Server -> Client: %s\nEnter Message : ", "\033[K", timestamp, message)
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
	fmt.Print("Enter your username: ")
	username, _ := reader.ReadString('\n')
	if len(username) > 0 && username[len(username)-1] == '\n' {
		username = username[:len(username)-1]
	}
	return username
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