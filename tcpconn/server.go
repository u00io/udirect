package tcpconn

import (
	"errors"
	"net"
	"sync"
	"time"
)

type Server struct {
	mtx  sync.Mutex
	port int

	started                 bool
	startedThAccept         bool
	startedThCleanupClients bool

	stopping bool
	clients  map[int64]*Client
	listener *net.TCPListener

	bufferSize int

	// callbacks
	OnConnected    func(*Client)
	OnReceived     func(*Client, []byte)
	OnDisconnected func(*Client)
}

var (
	ErrorAlreadyStarted = errors.New("server already started")
	ErrorNotStarted     = errors.New("server not started")
)

func NewServer() *Server {
	var c Server
	c.clients = make(map[int64]*Client)
	c.bufferSize = 64 * 1024
	return &c
}

func (c *Server) Start(port int) error {
	c.mtx.Lock()
	started := c.started
	c.mtx.Unlock()
	if started {
		return ErrorAlreadyStarted
	}

	var listener *net.TCPListener
	var err error
	listener, err = net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4zero, Port: port})
	if err != nil {
		return err
	}

	c.mtx.Lock()
	c.listener = listener
	c.port = port
	c.started = true
	c.mtx.Unlock()
	go c.thCleanupClients()
	go c.thAccept()
	return nil
}

func (c *Server) Stop() error {
	// Check if the server is started
	c.mtx.Lock()
	started := c.started
	c.mtx.Unlock()
	if !started {
		return ErrorNotStarted
	}

	// Signal the server to stop accepting new connections and close the listener
	c.mtx.Lock()
	c.stopping = true
	c.listener.Close()
	c.listener = nil
	clients := c.clients
	c.clients = make(map[int64]*Client)
	c.mtx.Unlock()

	// Close all client connections
	for _, client := range clients {
		client.Stop()
	}

	// Wait for the worker and cleanup goroutines to finish
	for c.startedThAccept || c.startedThCleanupClients {
		time.Sleep(10 * time.Millisecond)
	}

	// Mark the server as stopped
	c.mtx.Lock()
	c.started = false
	c.mtx.Unlock()
	return nil
}

func (c *Server) SetBufferSize(size int) {
	c.mtx.Lock()
	c.bufferSize = size
	c.mtx.Unlock()
	clients := make([]*Client, 0)
	c.mtx.Lock()
	for _, client := range c.clients {
		clients = append(clients, client)
	}
	c.mtx.Unlock()
	for _, client := range clients {
		client.SetBufferSize(size)
	}
}

func (c *Server) thCleanupClients() {
	c.mtx.Lock()
	c.startedThCleanupClients = true
	c.mtx.Unlock()

	for {
		c.mtx.Lock()
		stopping := c.stopping
		c.mtx.Unlock()

		if stopping {
			break
		}

		clientToRemove := make([]*Client, 0)
		clients := make([]*Client, 0)
		c.mtx.Lock()
		for _, client := range c.clients {
			clients = append(clients, client)
		}
		c.mtx.Unlock()

		for _, client := range clients {
			if client.Conn() != nil {
				clientToRemove = append(clientToRemove, client)
			}
		}

		c.mtx.Lock()
		for _, client := range clientToRemove {
			delete(c.clients, client.id)
		}
		c.mtx.Unlock()

		time.Sleep(100 * time.Millisecond)
	}

	c.mtx.Lock()
	c.startedThCleanupClients = false
	c.mtx.Unlock()
}

func (c *Server) thAccept() {
	c.mtx.Lock()
	c.startedThAccept = true
	c.mtx.Unlock()

	for {
		c.mtx.Lock()
		listener := c.listener
		stopping := c.stopping
		c.mtx.Unlock()
		if stopping {
			break
		}
		if listener == nil {
			time.Sleep(10 * time.Millisecond)
			continue
		}

		conn, err := listener.AcceptTCP()
		if err != nil {
			time.Sleep(10 * time.Millisecond)
			continue
		}
		client := newConnectedClient(conn, c.onClientConnected, c.onClientReceived, c.onClientDisconnected)
		client.SetBufferSize(c.bufferSize)
		c.mtx.Lock()
		c.clients[client.id] = client
		c.mtx.Unlock()
	}

	c.mtx.Lock()
	c.startedThAccept = false
	c.mtx.Unlock()
}

func (c *Server) onClientConnected(client *Client) {
	c.mtx.Lock()
	onConnected := c.OnConnected
	c.mtx.Unlock()
	if onConnected != nil {
		onConnected(client)
	}
}

func (c *Server) onClientReceived(client *Client, data []byte) {
	c.mtx.Lock()
	onReceived := c.OnReceived
	c.mtx.Unlock()
	if onReceived != nil {
		onReceived(client, data)
	}
}

func (c *Server) onClientDisconnected(client *Client) {
	c.mtx.Lock()
	onDisconnected := c.OnDisconnected
	c.mtx.Unlock()
	if onDisconnected != nil {
		onDisconnected(client)
	}
}
