package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"
)

type Session struct {
	conn   net.Conn
	reader *bufio.Reader
	writer *bufio.Writer
	input  chan string
	output chan bool
	done   chan error
	events chan<- string
	closed bool
}

func NewSession(address string, events chan<- string) (client *Session, err error) {
	var conn net.Conn

	for i := 0; i < 3; i++ {
		conn, err = net.Dial("tcp", address)

		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}

		break
	}

	if err != nil {
		return
	}

	client = &Session{
		conn,
		bufio.NewReader(conn),
		bufio.NewWriter(conn),
		make(chan string),
		make(chan bool, 2),
		make(chan error, 2),
		events,
		false,
	}

	go func() {
		for {
			resp, err := client.reader.ReadString('\n')

			if err != nil {
				client.done <- err
				break
			}

			if resp == "ok\n" {
				client.output <- true
			} else if strings.Index(resp, "error ") == 0 {
				client.output <- false
			} else {
				client.events <- resp
			}
		}
	}()

	go func() {
		for message := range client.input {
			if _, err = client.writer.WriteString(message + "\n"); err != nil {
				client.done <- err
				break
			}

			if err = client.writer.Flush(); err != nil {
				client.done <- err
				break
			}
		}
	}()

	return
}

func (session *Session) SendCommand(name string, args []string) (bool, error) {
	session.input <- fmt.Sprintf("%s %s", name, strings.Join(args, " "))

	select {
	case resp := <-session.output:
		return resp, nil
	case err := <-session.done:
		return false, err
	}
}

func (session *Session) Close() {
	if session.closed {
		return
	}

	session.conn.Close()
	close(session.input)
	close(session.output)
	close(session.events)
	session.closed = true
}
