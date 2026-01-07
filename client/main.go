package main

import (
	"fmt"
)

func main() {
	// 1. Ask for Server Address first
	serverURL := getServerAddress()

	// 2. Then Ask for Username
	username := getUsername()

	if username == "" {
		fmt.Println("Username cannot be empty")
		return
	}

	fmt.Println("Connecting to server...")

	// 3. Connect using the gathered info
	err := connectToEchoServer(serverURL, username)
	if err != nil {
		fmt.Println(err)
		return
	}
}