package udirect

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/u00io/udirect/stats"
	"github.com/u00io/udirect/tcpconn"
)

type Server struct {
	mtx       sync.Mutex
	port      int
	tcpServer *tcpconn.Server

	maxInputBufferSize int

	privateKeyHex string
	privateKey    []byte // Ed25519 private key
	publicKey     []byte // Ed25519 public key

	clients map[int64]*Client

	OnConnected     func(*Client)
	OnFrameReceived func(*Client, []byte)
	OnDisconnected  func(*Client)
}

func NewServer(privateKeyHex string) *Server {
	stats.Inc("udirect.server_new")
	var c Server
	c.clients = make(map[int64]*Client)
	c.maxInputBufferSize = defaultMaxInputBufferSize
	if privateKeyHex != "" {
		c.SetLocalPrivateKey(privateKeyHex)
	} else {
		c.GenerateLocalPrivateKey()
	}
	return &c
}

func (c *Server) Start(port int) error {
	stats.Inc("udirect.server_start")
	c.tcpServer = tcpconn.NewServer(port, c.onTcpConnected, c.onTcpReceived, c.onTcpDisconnected)
	err := c.tcpServer.Start()
	return err
}

func (c *Server) Stop() error {
	stats.Inc("udirect.server_stop")
	return c.tcpServer.Stop()
}

func (c *Server) SetMaxInputBufferSize(size int) {
	c.mtx.Lock()
	c.maxInputBufferSize = size
	c.mtx.Unlock()
}

func (c *Server) GenerateLocalPrivateKey() error {
	stats.Inc("udirect.server_generate_local_private_key")
	privateKey, err := GeneratePrivateKey()
	if err != nil {
		return err
	}
	privateKeyHex := hex.EncodeToString(privateKey)
	c.SetLocalPrivateKey(privateKeyHex)
	return nil
}

func (c *Server) SetLocalPrivateKey(privateKeyHex string) error {
	stats.Inc("udirect.server_set_local_private_key")

	privateKey, err := hex.DecodeString(privateKeyHex)
	if err != nil || len(privateKey) != ed25519.PrivateKeySize {
		return fmt.Errorf("invalid private key hex string")
	}
	c.mtx.Lock()
	c.privateKeyHex = privateKeyHex
	c.privateKey = privateKey
	c.publicKey = ed25519.PublicKey(privateKey[32:])
	c.mtx.Unlock()
	return nil
}

func (c *Server) onTcpConnected(client *tcpconn.Client) {
	stats.Inc("udirect.server_tcp_connected")
	udirectClient := newClientFromTcpClient(client, c.OnClientConnected, c.OnClientFrameReceived, c.OnClientDisconnected)
	udirectClient.SetLocalPrivateKey(c.privateKeyHex)
	udirectClient.SetMaxInputBufferSize(c.maxInputBufferSize)
	c.mtx.Lock()
	c.clients[client.ID()] = udirectClient
	c.mtx.Unlock()
	udirectClient.onTcpClientConnected(client)
}

func (c *Server) onTcpReceived(client *tcpconn.Client, data []byte) {
	id := client.ID()
	c.mtx.Lock()
	udirectClient, ok := c.clients[id]
	c.mtx.Unlock()
	if ok && udirectClient != nil {
		udirectClient.onTcpClientReceived(client, data)
	}
}

func (c *Server) onTcpDisconnected(client *tcpconn.Client) {
	stats.Inc("udirect.server_tcp_disconnected")
	id := client.ID()
	c.mtx.Lock()
	udirectClient, ok := c.clients[id]
	c.mtx.Unlock()
	if ok && udirectClient != nil {
		udirectClient.onTcpClientDisconnected(client)
	}
	c.mtx.Lock()
	delete(c.clients, id)
	c.mtx.Unlock()
}

func (c *Server) OnClientConnected(client *Client) {
	stats.Inc("tcpconn.server_on_client_connected")
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
	stats.Inc("tcpconn.server_on_client_disconnected")
	c.mtx.Lock()
	onDisconnected := c.OnDisconnected
	c.mtx.Unlock()
	if onDisconnected != nil {
		onDisconnected(client)
	}
}
