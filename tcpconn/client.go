package tcpconn

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

type Client struct {
	id            int64
	mtx           sync.Mutex
	mtxSend       sync.Mutex
	host          string
	port          int
	autoreconnect bool
	bufferSize    int

	conn *net.TCPConn

	started  bool
	stopping bool

	onConnected    func(client *Client)
	onReceived     func(client *Client, data []byte)
	onDisconnected func(client *Client)

	onConnectedCalled   bool
	isDialing           bool
	isOnReceivedRunning bool
}

var (
	ErrorNotConnected = errors.New("client not connected")
	ErrorSendFailed   = errors.New("failed to send data")
)

func NewClient(onConnected func(*Client), onReceived func(*Client, []byte), onDisconnected func(*Client)) *Client {
	var c Client
	c.id = getNextClientID()
	c.bufferSize = 64 * 1024
	c.onConnected = onConnected
	c.onReceived = onReceived
	c.onDisconnected = onDisconnected
	if c.onConnected == nil {
		c.onConnected = func(*Client) {}
	}
	if c.onReceived == nil {
		c.onReceived = func(*Client, []byte) {}
	}
	if c.onDisconnected == nil {
		c.onDisconnected = func(*Client) {}
	}
	return &c
}

func newConnectedClient(conn *net.TCPConn, onConnected func(*Client), onReceived func(*Client, []byte), onDisconnected func(*Client)) *Client {
	var c Client
	c.id = getNextClientID()
	c.conn = conn
	c.host = conn.RemoteAddr().(*net.TCPAddr).IP.String()
	c.port = conn.RemoteAddr().(*net.TCPAddr).Port
	c.autoreconnect = false

	c.onConnected = onConnected
	c.onReceived = onReceived
	c.onDisconnected = onDisconnected
	if c.onConnected == nil {
		c.onConnected = func(*Client) {}
	}
	if c.onReceived == nil {
		c.onReceived = func(*Client, []byte) {}
	}
	if c.onDisconnected == nil {
		c.onDisconnected = func(*Client) {}
	}

	c.onConnectedCalled = false
	go c.thWork()
	return &c
}

func (c *Client) SetBufferSize(size int) {
	c.mtx.Lock()
	c.bufferSize = size
	c.mtx.Unlock()
}

func (c *Client) Conn() *net.TCPConn {
	c.mtx.Lock()
	conn := c.conn
	c.mtx.Unlock()
	return conn
}

func (c *Client) ID() int64 {
	return c.id
}

func (c *Client) SetAutoReconnect(enabled bool) {
	c.mtx.Lock()
	c.autoreconnect = enabled
	c.mtx.Unlock()
}

func (c *Client) Start(addr string, port int) {
	c.mtx.Lock()
	started := c.started
	if started {
		c.mtx.Unlock()
		return
	}
	c.host = addr
	c.port = port
	c.autoreconnect = true
	c.onConnectedCalled = false
	c.mtx.Unlock()
	go c.thWork()
}

func (c *Client) Stop() error {
	c.mtx.Lock()
	started := c.started
	c.mtx.Unlock()
	if !started {
		return ErrorNotConnected
	}

	c.mtx.Lock()
	conn := c.conn
	c.stopping = true
	if conn != nil {
		c.autoreconnect = false
		conn.Close()
		c.conn = nil
	}

	//isOnReceivedRunning := c.isOnReceivedRunning
	c.mtx.Unlock()

	/*if !isOnReceivedRunning {
		for {
			c.mtx.Lock()
			started := c.started
			c.mtx.Unlock()
			if !started {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
	}*/

	return nil
}

func (c *Client) CloseConnection() {
	c.mtx.Lock()
	conn := c.conn
	if conn != nil {
		conn.Close()
		c.conn = nil
	}
	c.mtx.Unlock()
}

func (c *Client) checkConnection() *net.TCPConn {
	var err error
	c.mtx.Lock()
	conn := c.conn
	if conn != nil {
		c.mtx.Unlock()
		return conn
	}
	c.mtx.Unlock()
	if !c.autoreconnect {
		return nil
	}

	c.mtx.Lock()
	if c.isDialing {
		c.mtx.Unlock()
		return nil
	}
	c.isDialing = true
	c.mtx.Unlock()

	tcpAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", c.host, c.port))
	if err != nil {
		c.mtx.Lock()
		c.isDialing = false
		c.mtx.Unlock()
		return nil
	}

	conn, err = net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		fmt.Printf("Failed to connect to %s:%d: %v\n", c.host, c.port, err)
		c.mtx.Lock()
		c.isDialing = false
		c.mtx.Unlock()
		return nil
	}
	fmt.Printf("Connected to %s:%d\n", c.host, c.port)
	c.mtx.Lock()
	c.onConnectedCalled = false
	c.conn = conn
	c.isDialing = false
	c.mtx.Unlock()
	return conn
}

func (c *Client) Send(data []byte) error {
	var err error
	conn := c.checkConnection()
	if conn == nil {
		return ErrorNotConnected
	}

	c.mtxSend.Lock()
	sent := 0
	for sent < len(data) {
		n, err := conn.Write(data[sent:])
		if err != nil {
			c.mtxSend.Unlock()
			return err
		}
		if n <= 0 {
			c.mtxSend.Unlock()
			return ErrorSendFailed
		}
		sent += n
	}
	c.mtxSend.Unlock()
	return err
}

func (c *Client) thWork() {
	c.mtx.Lock()
	c.started = true
	buffer := make([]byte, c.bufferSize)
	c.mtx.Unlock()

	for {
		c.mtx.Lock()
		stopping := c.stopping
		bufferSize := c.bufferSize
		c.mtx.Unlock()
		if stopping {
			break
		}
		conn := c.checkConnection()
		if conn == nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		if len(buffer) != bufferSize {
			buffer = make([]byte, bufferSize)
		}

		c.mtx.Lock()
		onConnected := c.onConnected
		needCallOnConnected := !c.onConnectedCalled
		c.onConnectedCalled = true
		c.mtx.Unlock()
		if onConnected != nil && needCallOnConnected {
			onConnected(c)
		}

		n, err := conn.Read(buffer)
		if err != nil {
			conn.Close()
			c.mtx.Lock()
			c.conn = nil
			onDisconnected := c.onDisconnected
			c.mtx.Unlock()
			if onDisconnected != nil {
				onDisconnected(c)
			}
			continue
		}

		receviedData := buffer[:n]
		c.onReceived(c, receviedData)
	}
	c.mtx.Lock()
	c.started = false
	c.stopping = false
	c.mtx.Unlock()
}
