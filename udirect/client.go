package udirect

import "github.com/u00io/udirect/tcpconn"

// Client represents a client connection to the server.
// Like TLS connection uses encryption, but Ed25519 is used for authentication and key exchange.
// AES-256-GCM is used for encryption and decryption.
// The client can connect to the server and send data to the server, and receive data from the server.
// The client can also disconnect from the server.
// The client can be configured to automatically reconnect to the server if the connection is lost.
// Based on ./tcpconn package

type Client struct {
	tcpClient       *tcpconn.Client
	privateKey      []byte // Ed25519 private key
	publicKey       []byte // Ed25519 public key
	remotePublicKey []byte // Ed25519 public key of the server
	aesKey          []byte // AES-256-GCM key derived from the shared secret
}

func NewClient(privateKey []byte) *Client {
	var c Client
	c.privateKey = privateKey
	c.tcpClient = tcpconn.NewClient()
	return &c
}
