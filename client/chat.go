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

func connectToEchoServer(serverURL string, username string, password string) error {
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
	// Send password immediately after username for authentication
	if err := c.WriteMessage(websocket.TextMessage, []byte(password)); err != nil {
		return fmt.Errorf("Failed to send password to server: %v", err)
	}

	readyCh := make(chan struct{})
	errCh := make(chan error, 1)

	fmt.Println("-------------------------------------------")
	fmt.Println("Waiting for authentication...")

	// Goroutine to read incoming messages
	go func() {
		for {
			messageType, data, err := c.ReadMessage()
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
				log.Println(err)
				return
			}
			if err != nil {
				errCh <- err
				return
			}
			if messageType == websocket.TextMessage {
				message := string(data)
				if message == "[[AUTH_OK]]" {
					// Signal that we can start sending
					select {
					case <-readyCh:
						// already closed
					default:
						close(readyCh)
					}
					fmt.Println("Authenticated. Listening for messages...")
					continue
				}
				if strings.HasPrefix(message, "ERROR:") {
					fmt.Println(message)
					errCh <- fmt.Errorf(message)
					return
				}
				// Print normal chat messages
				fmt.Printf("\r%s%s\nEnter Message : ", "\u001b[K", message)
			}
		}
	}()

	// Wait until authenticated or error
	select {
	case <-readyCh:
		// proceed
	case err := <-errCh:
		return err
	}

	fmt.Println("-------------------------------------------")
	fmt.Println("You can chat now.")

	// Read for terminal input
	reader := bufio.NewReader(os.Stdin)

	// Infinite loop so that user can keep sending messages
	for {
		fmt.Print("Enter Message : ")
		text, _ := reader.ReadString('\n')
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		if err := c.WriteMessage(websocket.TextMessage, []byte(text)); err != nil {
			return fmt.Errorf("failed to send message: %v", err)
		}
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

func getPassword() string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter your password: ")
	password, _ := reader.ReadString('\n')
	return strings.TrimSpace(password)
}