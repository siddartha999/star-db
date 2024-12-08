package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"syscall"

	constants "github.com/siddartha999/star-db/config"
)

// Processes the connection and returns back a response
func processConnection(connection net.Conn) {
	logFile, err := os.OpenFile(constants.LogFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		fmt.Println("Error opening log file")
		log.Fatal(err)
	}
	defer logFile.Close()

	buffer := make([]byte, 2048)
	bytesRead, e := connection.Read(buffer)
	if e != nil {
		fmt.Println("Error reading connection's request", connection, err)
	}
	fmt.Printf("Read %d bytes", bytesRead)
	logFile.WriteString(fmt.Sprintf("Reading %d bytes for the connection: %s\n", bytesRead, connection.RemoteAddr().String()))
	reader := bufio.NewReader(strings.NewReader(string(buffer)))
	for {
		request, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				fmt.Println("Error processing the connection's request: ", connection, err)
			}
			break
		}
		if request == "" {
			break
		}
		if len(request) >= 2 && request[len(request)-2] == '\r' {
			connection.Write([]byte("+OK\r\n"))
		} else { // Invalid request format
			connection.Write([]byte("-ERR invalid request format. Request does not end in CRLF.\r\n"))
		}
		fmt.Println(request)
		logFile.WriteString(request)
	}
	logFile.WriteString(fmt.Sprintf("Completed reading %d bytes for the connection: %s\n", bytesRead, connection.RemoteAddr().String()))
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
	kQueueFD, err := syscall.Kqueue()
	if err != nil {
		fmt.Println("Error creating a Kqueue(epoll equivalent) file descriptor")
		log.Fatal(err)
	}
	defer syscall.Close(kQueueFD)
	fmt.Println("Kqueue file descriptor: ", uint64(kQueueFD))

	// Retrieve file descriptor for the Network socket
	listenerFile, err := listener.(*net.TCPListener).File()
	if err != nil {
		fmt.Println("Error reading file descriptor for the network socket on port: ", listener.Addr().String())
		log.Fatal(err)
	}
	fmt.Println("Network socket file descriptor: ", uint64(listenerFile.Fd()))
	defer listenerFile.Close()

	// Create an event for Kqueue to monitor on the Network socket
	event := syscall.Kevent_t{
		Ident:  uint64(listenerFile.Fd()),
		Filter: syscall.EVFILT_READ,
		Flags:  syscall.EV_ADD | syscall.EV_ENABLE,
		Fflags: 0,
		Data:   0,
		Udata:  nil,
	}
	// Registering the event to get notified of new connections to the Network socket
	_, err = syscall.Kevent(kQueueFD, []syscall.Kevent_t{event}, nil, nil)
	if err != nil {
		fmt.Println("Error initializing kqueue")
		log.Fatal(err)
	}

	events := make([]syscall.Kevent_t, 1)
	connectionsMap := make(map[int]net.Conn)
	// An infinite Kqueue poller.
	for {
		fmt.Println("Polling KQueue for events")
		//Retrieving registered events from kqueue
		n, err := syscall.Kevent(kQueueFD, nil, events, &syscall.Timespec{Nsec: 1})
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
				if events[i].Ident == uint64(listenerFile.Fd()) {
					fmt.Println("Awaiting a new client connection")
					connection, err := listener.Accept()
					if err != nil {
						log.Fatal(err)
					}
					connectionFile, err := connection.(*net.TCPConn).File()
					if err != nil {
						fmt.Printf("Unable to extract connection's %s file descriptor", connection.RemoteAddr().String())
						continue
					}
					// Create a new read event for the connection. The connection's Receiver queue is to be monitored.
					connectionEvent := syscall.Kevent_t{
						Ident:  uint64(connectionFile.Fd()),
						Filter: syscall.EVFILT_READ,
						Flags:  syscall.EV_ADD | syscall.EV_ENABLE | syscall.EV_CLEAR,
					}
					// Register the event with the kqueue. kqueue will notify the main thread when data from the connection is available.
					_, err = syscall.Kevent(kQueueFD, []syscall.Kevent_t{connectionEvent}, nil, nil)
					if err != nil {
						fmt.Printf("Unable to monitor connection's %s file descriptor", connection.RemoteAddr().String())
						continue
					}
					fmt.Println("Connection accepted: ", connection)
					connectionsMap[int(connectionFile.Fd())] = connection
				} else {
					currentConnectionFD := int(events[i].Ident)
					currentConnection := connectionsMap[currentConnectionFD]
					processConnection(currentConnection)
					fmt.Println("Connection processed: ", currentConnection.RemoteAddr().String())
				}
			}
			fmt.Printf("Processed %d events returned by Kqueue", n)
		}
	}
}
