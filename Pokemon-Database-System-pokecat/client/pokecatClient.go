package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

func main() {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		fmt.Println("Error connecting to server:", err.Error())
		return
	}
	defer conn.Close()

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter your name to register or login: ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)

	// Send the player's name to the server
	_, err = fmt.Fprintf(conn, "%s\n", name)
	if err != nil {
		fmt.Println("Error sending name to server:", err.Error())
		return
	}

	go readFromServer(conn)

	for {
		fmt.Println("Enter a command:")
		text, _ := reader.ReadString('\n')
		text = strings.TrimSpace(text)

		// Send command to the server
		_, err := fmt.Fprintf(conn, "%s\n", text)
		if err != nil {
			fmt.Println("Error sending command to server:", err.Error())
			return
		}

		// Exit condition
		if text == "exit" {
			fmt.Println("Exiting client...")
			return
		}
	}
}

func readFromServer(conn net.Conn) {
	reader := bufio.NewReader(conn)
	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading from server:", err.Error())
			return
		}
		fmt.Print(msg)
	}
}
