package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	host    string
	port    int
	addr    string
	clients = make(map[*Client]bool)
	mutex   sync.Mutex
	line    string
	history []string
)

type Client struct {
	conn net.Conn
	name string
}

func main() {
	//fmt.Println(os.Args)
	if len(os.Args) == 1 {
		port = 8989
		StartServer(port)
	} else if len(os.Args) == 2 {
		port, err := strconv.Atoi(os.Args[1])
		if port < 1 || port > 65535 || err != nil {
			log.Fatal("Invalid port number")
			port = 8989
			fmt.Println("invalid port number, 8989 has been used")
		}
		StartServer(port)
	} else {
		fmt.Println("[USAGE]: ./TCPChat $port")
	}

}

func StartServer(port int) {
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		log.Fatalf("Failed to start server on %s: %v", addr, err)
		return
	}
	defer listener.Close()
	fmt.Printf("Server is listening on port %d\n", port)
	fmt.Println("Welcome to TCP-Chat!")
	f, e := os.Open("ascci.txt")
	if e != nil {
		fmt.Println("Error opening file:", e)
		os.Exit(0)
	}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line += scanner.Text() + "\n"
	}
	fmt.Println(line)
	for {
		// accept incoming connections
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection from client: %s", err)
		}
		for _, c := range history {
			conn.Write([]byte(c + "\n"))
		}
		// accepts TCP connections from clients and processes these
		// connections in separate goroutines
		go HandleConnection(conn)
	}
}

func HandleConnection(conn net.Conn) {
	defer conn.Close()
	conn.Write([]byte("Welcome to TCP-Chat!\n"))
	conn.Write([]byte(line))
	// conn.Write([]byte(line))
	// Display ASCII art to the user and get a name
	conn.Write([]byte("\n[ENTER YOUR NAME]:"))
	names, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil {
		fmt.Println("Failed to read client name:", err)
		return
	}

	name := strings.TrimSpace(string(names))
	if name == "" {
		conn.Write([]byte("Invalid name. Please enter a name:"))
		HandleConnection(conn)
		return
	}

	// Create a new client
	client := &Client{
		conn: conn,
		name: name,
	}

	// Add the client to the list of connected clients
	mutex.Lock()
	clients[client] = true
	mutex.Unlock()
	// Broadcast join message
	enterandleaves(fmt.Sprintf("%s has joined the chat...", name), conn)
	history = append(history, fmt.Sprintf("%s has joined the chat...", name))
	// Read and broadcast messages
	reader := bufio.NewReader(conn)
	for { // Read data from the client
		buffer, err := reader.ReadBytes('\n')
		if err != nil {
			mutex.Lock()
			delete(clients, client)
			mutex.Unlock()
			enterandleaves(fmt.Sprintf("%s has left the chat.", client.name), conn)
			history = append(history, fmt.Sprintf("%s has left the chat.", client.name))
			return
		}

		sendername := ""
		for client := range clients {
			if client.conn == conn {
				sendername = client.name
			}
		}
		if strings.TrimSpace(string(buffer)) != "" {
			// Broadcast the message to all other clients
			broadcastMessage(string(buffer), conn, sendername)
		}
	}
}

func broadcastMessage(message string, sender net.Conn, name string) {
	mutex.Lock()
	defer mutex.Unlock()

	// Create the timestamp once
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	formattedMessage := fmt.Sprintf("[%s][%s]: %s", timestamp, name, message)
	history = append(history, fmt.Sprintf("[%s][%s]: %s", timestamp, name, message))
	// Print the message to the server console once
	fmt.Print(formattedMessage)

	for client := range clients {
		if client.conn != sender {
			_, err := client.conn.Write([]byte(formattedMessage))
			if err != nil {
				// Log the error and clean up the client
				client.conn.Close()
				mutex.Lock()
				delete(clients, client)
				mutex.Unlock()
				enterandleaves(fmt.Sprintf("%s has left the chat.", client.name), client.conn)
				history = append(history, fmt.Sprintf("%s has left the chat.", client.name))
			}
		}
	}
}

func enterandleaves(message string, sender net.Conn) {
	mutex.Lock()
	defer mutex.Unlock()

	for client := range clients {
		if client.conn != sender {
			_, err := client.conn.Write([]byte(message + "\n"))
			if err != nil {
				// Log the error and clean up the client
				client.conn.Close()
				mutex.Lock()
				delete(clients, client)
				mutex.Unlock()
				// Notify other clients about the disconnection
				enterandleaves(fmt.Sprintf("%s has left the chat.", client.name), sender)
				history = append(history, fmt.Sprintf("%s has left the chat.", client.name))
			}
		}
	}
	// Print the message once after the broadcast
	fmt.Println(message)
}
