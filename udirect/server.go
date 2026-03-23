package udirect

import (
	"sync"

	"github.com/u00io/udirect/tcpconn"
)

type Server struct {
	mtx        sync.Mutex
	port       int
	tcpServer  *tcpconn.Server
	privateKey []byte // Ed25519 private key

	clients map[int64]*Client

	OnConnected     func(*Client)
	OnFrameReceived func(*Client, []byte)
	OnDisconnected  func(*Client)
}

func NewServer() *Server {
	var c Server
	c.privateKey, _ = GeneratePrivateKey()
	c.clients = make(map[int64]*Client)
	return &c
}

func (c *Server) Start(port int) error {
	c.tcpServer = tcpconn.NewServer()
	c.tcpServer.OnConnected = c.onTcpConnected
	c.tcpServer.OnReceived = c.onTcpReceived
	c.tcpServer.OnDisconnected = c.onTcpDisconnected
	err := c.tcpServer.Start(port)
	return err
}

func (c *Server) Stop() error {
	return c.tcpServer.Stop()
}

func (c *Server) onTcpConnected(client *tcpconn.Client) {
	udirectClient := newClientFromTcpClient(client, c.privateKey, c.OnClientConnected, c.OnClientFrameReceived, c.OnClientDisconnected)
	c.mtx.Lock()
	c.clients[client.ID()] = udirectClient
	c.mtx.Unlock()
	udirectClient.onTcpClientConnected(client)
}

func (c *Server) onTcpReceived(client *tcpconn.Client, data []byte) {
	udirectClient, ok := c.clients[client.ID()]
	if ok && udirectClient != nil {
		udirectClient.onTcpClientReceived(client, data)
	}
}

func (c *Server) onTcpDisconnected(client *tcpconn.Client) {
	udirectClient, ok := c.clients[client.ID()]
	if ok && udirectClient != nil {
		udirectClient.onTcpClientDisconnected(client)
	}
	delete(c.clients, client.ID())
}

func (c *Server) OnClientConnected(client *Client) {
	c.mtx.Lock()
	onConnected := c.OnConnected
	c.mtx.Unlock()
	if onConnected != nil {
		onConnected(client)
	}
}

func (c *Server) OnClientFrameReceived(client *Client, data []byte) {
	c.mtx.Lock()
	onFrameReceived := c.OnFrameReceived
	c.mtx.Unlock()
	if onFrameReceived != nil {
		onFrameReceived(client, data)
	}
}

func (c *Server) OnClientDisconnected(client *Client) {
	c.mtx.Lock()
	onDisconnected := c.OnDisconnected
	c.mtx.Unlock()
	if onDisconnected != nil {
		onDisconnected(client)
	}
}
