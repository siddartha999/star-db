package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"syscall"
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
	// Listening to a Port
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()
	fmt.Println("Opened a TCP listener: ", &listener)

	// Initializing Kqueue (epoll equivalent for MACOS)
	fd, err := syscall.Kqueue()
	if err != nil {
		fmt.Println("Error creating a Kqueue(epoll equivalent) file descriptor")
		log.Fatal(err)
	}
	defer syscall.Close(fd)
	fmt.Println("Kqueue file descriptor: ", uint64(fd))

	// Retrieve file descriptor for the Network socket
	file, err := listener.(*net.TCPListener).File()
	if err != nil {
		fmt.Println("Error reading file descriptor for the network socket on port: ", listener.Addr().String())
		log.Fatal(err)
	}
	fmt.Println("Network socket file descriptor: ", uint64(file.Fd()))

	// Create an event for Kqueue to monitor on the Network socket
	event := syscall.Kevent_t{
		Ident:  uint64(file.Fd()),
		Filter: syscall.EVFILT_READ,
		Flags:  syscall.EV_ADD | syscall.EV_ENABLE,
		Fflags: 0,
		Data:   0,
		Udata:  nil,
	}
	events := make([]syscall.Kevent_t, 10)

	// An infinite Kqueue poller.
	for {
		fmt.Println("Polling KQueue for events")
		n, err := syscall.Kevent(fd, []syscall.Kevent_t{event}, events, &syscall.Timespec{Nsec: 1000000})
		if err != nil {
			if err == syscall.EINTR {
				//Interrupted system call, keep on going
				continue
			}
			fmt.Println("Error polling Kqueue for events")
			log.Fatal(err)
		}
		fmt.Printf("Received %d events from Kqueue", n)

		if n > 0 {
			fmt.Printf("Processing %d events returned by Kqueue", n)
			for i := 0; i < n; i++ {
				if events[i].Ident == uint64(file.Fd()) {
					fmt.Println("Awaiting a new client connection")
					connection, err := listener.Accept()
					if err != nil {
						log.Fatal(err)
					}
					fmt.Println("Connection accepted: ", connection)
					processConnection(connection)
					fmt.Println("Connection processed: ", connection)
				}
			}
			fmt.Printf("Processed %d events returned by Kqueue", n)
		}
	}
}
