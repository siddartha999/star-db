package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
)

// Reads and prints the connection's request
func readConnection(connection net.Conn) {
	buffer := make([]byte, 2048)
	_, err := connection.Read(buffer)
	if err != nil {
		fmt.Println("Error reading connection's request", connection, err)
	}
	fmt.Printf("Read %d bytes", len(buffer))
	reader := bufio.NewReader(strings.NewReader(string(buffer)))
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				fmt.Println("Error processing the connection's request: ", connection, err)
			}
			break
		}
		if line == "" {
			fmt.Println("End of parsing client request")
			break
		}
		fmt.Println(line)
	}
}

// Processes the connection and returns back a response
func processConnection(connection net.Conn) {
	fmt.Println("Processing connection: ", connection)
	readConnection(connection)
	connection.Write([]byte("HTTP/1.1 200 OK\r\n\r\nHello\r\n"))
	connection.Write([]byte("Thank you for reaching out\r\n"))
}

func main() {
	// Starting a TCP-server
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()
	fmt.Println("Opened a TCP listener: ", &listener)
	for {
		fmt.Println("Awaiting a new client connection")
		connection, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Connection accepted: ", connection)
		processConnection(connection)
		connection.Close()
		fmt.Println("Connection processed and closed: ", connection)
	}
}
