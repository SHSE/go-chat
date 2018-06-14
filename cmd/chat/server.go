package main

import (
	"bufio"
	"context"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"net"
	"strings"
	"sync"
	"time"
)

type client struct {
	id       int
	conn     net.Conn
	messages chan string
	closing  chan struct{}
}

func newClient(id int, conn net.Conn) client {
	return client{id, conn, make(chan string, 8), make(chan struct{}, 1)}
}

func (client client) close() {
	select {
	case client.closing <- struct{}{}:
	default:
	}
}

func (client client) send(message string) {
	client.messages <- message
}

func (client client) receiveCommands(server *Server) {
	reader := bufio.NewReader(client.conn)

	for {
		text, err := reader.ReadString('\n')

		if err != nil {
			break
		}

		data := strings.TrimSpace(text)

		if data == "" {
			continue
		}

		parts := strings.Split(data, " ")
		name, args := parts[0], parts[1:]

		server.commands <- command{client.id, name, args}
	}

	server.disconnected <- client
}

func (client client) deliverMessages() {
	writer := bufio.NewWriter(client.conn)

	for {
		select {
		case <-client.closing:
			client.conn.Close()
			return

		case message := <-client.messages:
			writer.WriteString(message + "\n")

		buffering:
			for {
				select {
				case message, ok := <-client.messages:
					if !ok {
						break buffering
					}

					writer.WriteString(message + "\n")
				default:
					break buffering
				}
			}

			writer.Flush()
		}
	}
}

type Server struct {
	closing      chan struct{}
	logger       *zap.Logger
	connections  sync.Map
	count        int
	commands     chan command
	connected    chan client
	disconnected chan client
	done         chan struct{}
	metrics      prometheus.Registerer
}

func NewServer(logger *zap.Logger, metrics prometheus.Registerer) *Server {
	return &Server{
		make(chan struct{}, 1),
		logger,
		sync.Map{},
		0,
		make(chan command),
		make(chan client),
		make(chan client),
		make(chan struct{}, 1),
		metrics,
	}
}

func (server *Server) Close() {
	server.closing <- struct{}{}
	<-server.done
}

func (server *Server) Run(ctx context.Context, address string, handler commandHandler) error {
	listener, err := net.Listen("tcp", address)

	if err != nil {
		return err
	}

	server.logger.Info("Server started", zap.String("address", address))

	var clientCounter = 0

	go server.dispatch(handler)

	go func() {
		<-ctx.Done()
		listener.Close()
	}()

	for {
		conn, err := listener.Accept()

		if ctx.Err() != nil {
			break
		}

		if err != nil {
			server.logger.Error("Failed to accept connection", zap.Error(err))
			conn.Close()
			continue
		}

		clientCounter++

		client := newClient(clientCounter, conn)

		server.connected <- client
	}

	listener.Close()
	server.Close()

	server.logger.Info("Shutdown completed")

	return nil
}

func (server *Server) sendTo(clientId int, data string) {
	value, found := server.connections.Load(clientId)

	if found {
		value.(client).send(data)
	}
}

func (server *Server) shutdown() bool {
	if server.count == 0 {
		return false
	}

	select {
	case <-server.commands:
	case <-server.disconnected:
		server.count--

		if server.count == 0 {
			return false
		}
	}
	return true
}

func (server *Server) dispatch(handler commandHandler) {
	connected := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "connected_clients",
		Help: "Number of connected clients."})

	commandTime := prometheus.NewSummary(prometheus.SummaryOpts{
		Name: "command_time",
		Help: "Command duration.",
	})

	server.metrics.MustRegister(connected)
	server.metrics.MustRegister(commandTime)

	for {
		select {
		case client := <-server.connected:
			server.connections.Store(client.id, client)
			server.count++
			handler.connected(client.id)
			go client.deliverMessages()
			go client.receiveCommands(server)
			connected.Inc()

		case client := <-server.disconnected:
			server.connections.Delete(client.id)
			server.count--
			handler.disconnected(client.id)
			client.close()
			connected.Dec()

		case command := <-server.commands:
			started := time.Now()
			err := handler.command(command)

			if err != nil {
				server.sendTo(command.clientId, "error "+err.Error())
			} else {
				server.sendTo(command.clientId, "ok")
			}

			commandTime.Observe(time.Since(started).Seconds())

		case <-server.closing:
			server.connections.Range(func(key, value interface{}) bool {
				client := value.(client)
				client.messages <- "Server is shutting down"
				client.close()
				return true
			})

			for server.shutdown() {
			}

			server.done <- struct{}{}
			return
		}
	}
}
