package main

import (
	"fmt"
)

func main() {
	// 1. Ask for Server Address
	serverURL := getServerAddress()

	// 2. Ask for Username
	username := getUsername()

	// 3. Ask for Password (NEW)
	password := getPassword()

	if username == "" {
		fmt.Println("Username cannot be empty")
		return
	}

	fmt.Println("Connecting to server...")

	// 4. Connect using the gathered info (include password)
	err := connectToEchoServer(serverURL, username, password)
	if err != nil {
		fmt.Println(err)
		return
	}
}