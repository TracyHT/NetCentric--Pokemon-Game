package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

func main() {
	serverAddr, err := net.ResolveUDPAddr("udp", "localhost:8080")
	if err != nil {
		fmt.Println("Error resolving server address:", err)
		return
	}

	conn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		fmt.Println("Error dialing server:", err)
		return
	}
	defer conn.Close()

	reader := bufio.NewReader(os.Stdin)

	var username string
	for {
		fmt.Print("Enter your username: ")
		username, _ = reader.ReadString('\n')
		username = strings.TrimSpace(username)

		sendMessage(conn, "LOGIN:"+username)

		response := receiveMessage(conn)
		fmt.Println("Server response:", response)

		if strings.HasPrefix(response, "SUCCESS:") {
			fmt.Println("Welcome " + username + "!")
			break
		} else if strings.HasPrefix(response, "ERROR:") {
			fmt.Println("Login failed:", response)
		}
	}

	// Start a goroutine to continuously receive messages from the server
	go func() {
		for {
			msg := receiveMessage(conn)
			if msg != "" {
				fmt.Println(msg)
			}
		}
	}()

	// Main loop to send messages to the server
	for {
		text, _ := reader.ReadString('\n')
		sendMessage(conn, text)
	}
}

func sendMessage(conn *net.UDPConn, message string) {
	_, err := conn.Write([]byte(message))
	if err != nil {
		fmt.Println("Error sending message:", err)
	}
}

func receiveMessage(conn *net.UDPConn) string {
	buffer := make([]byte, 1024)
	n, _, err := conn.ReadFromUDP(buffer)
	if err != nil {
		fmt.Println("Error reading from UDP:", err)
		return ""
	}
	return strings.TrimSpace(string(buffer[:n]))
}
