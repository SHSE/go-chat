package transport

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

const MessageServerIsShuttingDown = "Server is shutting down"

type client struct {
	id       int
	conn     net.Conn
	messages chan string
	closing  chan struct{}
}

func newClient(id int, conn net.Conn) client {
	return client{id, conn, make(chan string, 8), make(chan struct{}, 1)}
}

func (c client) close() {
	select {
	case c.closing <- struct{}{}:
	default:
	}
}

func (c client) send(message string) {
	c.messages <- message
}

func (c client) receiveCommands(server *Server) {
	reader := bufio.NewReader(c.conn)

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

		server.commands <- Command{c.id, name, args}
	}

	server.disconnected <- c
}

func (c client) deliverMessages() {
	writer := bufio.NewWriter(c.conn)

	for {
		select {
		case <-c.closing:
			c.conn.Close()
			return

		case message := <-c.messages:
			writer.WriteString(message + "\n")

		buffering:
			for {
				select {
				case message, ok := <-c.messages:
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
	commands     chan Command
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
		make(chan Command),
		make(chan client),
		make(chan client),
		make(chan struct{}, 1),
		metrics,
	}
}

func (s *Server) Close() {
	s.closing <- struct{}{}
	<-s.done
}

func (s *Server) Run(ctx context.Context, address string, handler CommandHandler) error {
	listener, err := net.Listen("tcp", address)

	if err != nil {
		return err
	}

	s.logger.Info("Server started", zap.String("address", address))

	var clientCounter = 0

	go s.dispatch(handler)

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
			s.logger.Error("Failed to accept connection", zap.Error(err))
			conn.Close()
			continue
		}

		clientCounter++

		client := newClient(clientCounter, conn)

		s.connected <- client
	}

	listener.Close()
	s.Close()

	s.logger.Info("Shutdown completed")

	return nil
}

func (s *Server) SendTo(clientId int, data string) {
	value, found := s.connections.Load(clientId)

	if found {
		value.(client).send(data)
	}
}

func (s *Server) shutdown() bool {
	if s.count == 0 {
		return false
	}

	select {
	case <-s.commands:
	case <-s.disconnected:
		s.count--

		if s.count == 0 {
			return false
		}
	}
	return true
}

func (s *Server) dispatch(handler CommandHandler) {
	connected := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "connected_clients",
		Help: "Number of Connected clients."})

	commandTime := prometheus.NewSummary(prometheus.SummaryOpts{
		Name: "command_time",
		Help: "Command duration.",
	})

	s.metrics.MustRegister(connected)
	s.metrics.MustRegister(commandTime)

	for {
		select {
		case client := <-s.connected:
			s.connections.Store(client.id, client)
			s.count++
			handler.Connected(client.id)
			go client.deliverMessages()
			go client.receiveCommands(s)
			connected.Inc()

		case client := <-s.disconnected:
			s.connections.Delete(client.id)
			s.count--
			handler.Disconnected(client.id)
			client.close()
			connected.Dec()

		case command := <-s.commands:
			started := time.Now()
			err := handler.Command(command)

			if err != nil {
				s.SendTo(command.ClientId, "error "+err.Error())
			} else {
				s.SendTo(command.ClientId, "ok")
			}

			commandTime.Observe(time.Since(started).Seconds())

		case <-s.closing:
			s.connections.Range(func(key, value interface{}) bool {
				client := value.(client)
				client.messages <- MessageServerIsShuttingDown
				client.close()
				return true
			})

			for s.shutdown() {
			}

			s.done <- struct{}{}
			return
		}
	}
}
