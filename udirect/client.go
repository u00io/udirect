package udirect

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"github.com/u00io/udirect/tcpconn"
)

// Client represents a client connection to the server.
// Like TLS connection uses encryption, but Ed25519 is used for authentication and key exchange.
// AES-256-GCM is used for encryption and decryption.
// The client can connect to the server and send data to the server, and receive data from the server.
// The client can also disconnect from the server.
// The client can be configured to automatically reconnect to the server if the connection is lost.
// Based on ./tcpconn package

type Client struct {
	tcpClient                *tcpconn.Client
	addr                     string
	port                     int
	privateKey               []byte // Ed25519 private key
	publicKey                []byte // Ed25519 public key
	remotePublicKey          []byte // Ed25519 public key of the server
	transportPrivateKey      []byte // Ed25519 private key for transport
	transportPublicKey       []byte // Ed25519 public key for transport
	transportRemotePublicKey []byte // Ed25519 public key of the server for transport

	currentLocalNonce   []byte // nonce for AES-GCM (8 bytes)
	expectedRemoteNonce []byte // nonce for AES-GCM from remote side (8 bytes)

	maxInputBufferSize int // maximum size of the input buffer, default to 1MB

	aesKey []byte // AES-256-GCM key derived from the shared secret

	inputData []byte // buffer for incoming data

	OnConnected     func(client *Client)              // callback for connection established
	OnFrameReceived func(client *Client, data []byte) // callback for received data
	OnDisconnected  func(client *Client)              // callback for connection closed
}

func NewClient(privateKeyHex string, addr string, port int) *Client {
	var c Client
	privateKey, err := hex.DecodeString(privateKeyHex)
	if err != nil || len(privateKey) != ed25519.PrivateKeySize {
		privateKey, _ = GeneratePrivateKey()
	}
	c.addr = addr
	c.port = port
	c.maxInputBufferSize = 1024 * 1024 // 1MB
	c.privateKey = privateKey
	c.publicKey = ed25519.PublicKey(privateKey[32:])
	c.tcpClient = tcpconn.NewClient()
	c.tcpClient.OnConnected = c.onTcpClientConnected
	c.tcpClient.OnDisconnected = c.onTcpClientDisconnected
	c.tcpClient.OnReceived = c.onTcpClientReceived
	return &c
}

func newClientFromTcpClient(tcpClient *tcpconn.Client, privateKey []byte, onConnected func(*Client), onFrameReceived func(*Client, []byte), onDisconnected func(*Client)) *Client {
	var c Client
	c.privateKey = privateKey
	c.publicKey = ed25519.PublicKey(privateKey[32:])
	c.tcpClient = tcpClient
	c.maxInputBufferSize = 1024 * 1024 // 1MB
	c.OnConnected = onConnected
	c.OnFrameReceived = onFrameReceived
	c.OnDisconnected = onDisconnected
	return &c
}

func (c *Client) Start() {
	c.tcpClient.Start(c.addr, c.port)
}

func (c *Client) ID() int64 {
	return c.tcpClient.ID()
}

func (c *Client) onTcpClientConnected(client *tcpconn.Client) {
	fmt.Println("Client::onTcpClientConnected")
	// send handshake:
	// 0. public key of the client [32 bytes]
	// 1. transport public key and it is nonce as well [32 bytes]
	// 2. signature of the transport public key [64 bytes]
	// Total 96 bytes
	var err error
	c.transportPrivateKey, c.transportPublicKey, err = GenerateCurve25519KeyPair()
	if err != nil {
		return
	}
	payload := make([]byte, 128)
	copy(payload[0:32], c.publicKey)
	copy(payload[32:64], c.transportPublicKey)
	signature := ed25519.Sign(ed25519.PrivateKey(c.privateKey), c.transportPublicKey)
	copy(payload[64:64+64], signature) // 64 bytes
	handShakeFrame := newFrame(0, payload)
	err = c.tcpClient.Send(handShakeFrame.toBytes())
	if err != nil {
		return
	}
}

func (c *Client) onTcpClientDisconnected(client *tcpconn.Client) {
	fmt.Println("Client::onTcpClientDisconnected")
	c.aesKey = nil
	c.transportPrivateKey = nil
	c.transportPublicKey = nil

	if c.OnDisconnected != nil {
		c.OnDisconnected(c)
	}
}

func (c *Client) onTcpClientReceived(client *tcpconn.Client, data []byte) {
	fmt.Println("Client::onTcpClientReceived, data length:", len(data))
	if len(c.inputData)+len(data) > c.maxInputBufferSize {
		// drop the data if the input buffer is full
		return
	}
	c.inputData = append(c.inputData, data...)
	if len(c.inputData) < 4 {
		return
	}
	frameLength := binary.LittleEndian.Uint32(c.inputData[0:4])
	for len(c.inputData) >= int(frameLength) {
		frameData := make([]byte, frameLength)
		copy(frameData, c.inputData[0:frameLength])
		c.inputData = c.inputData[frameLength:]
		frame, err := frameFromBytes(frameData)
		if err != nil {
			continue
		}
		switch frame.Type {
		case 0: // handshake response
			if len(frame.Payload) == 128 {
				c.remotePublicKey = frame.Payload[0:32]
				c.transportRemotePublicKey = frame.Payload[32:64]
				signature := frame.Payload[64:128]
				if !ed25519.Verify(ed25519.PublicKey(c.remotePublicKey), c.transportRemotePublicKey, signature) {
					continue
				}
				c.aesKey, err = deriveAESKey(c.transportPrivateKey, c.transportRemotePublicKey)
				if err != nil {
					continue
				}
				fmt.Println("Handshake successful, AES key derived:", hex.EncodeToString(c.aesKey))
				if c.OnConnected != nil {
					c.OnConnected(c)
				}
			}
		case 1: // encrypted data
			decryptedData, err := decryptAESGCM(frame.Payload, c.aesKey)
			if err != nil {
				continue
			}

			if len(decryptedData) >= 16 {
				nonce := decryptedData[0:8]
				nextNonce := decryptedData[8:16]
				data := decryptedData[16:]
				if len(c.expectedRemoteNonce) > 0 {
					if !equalBytes(nonce, c.expectedRemoteNonce) {
						continue
					}
				} else {
					c.expectedRemoteNonce = nextNonce
				}

				if c.OnFrameReceived != nil {
					c.OnFrameReceived(c, data)
				}
			}
		}
	}
}

func (c *Client) Send(data []byte) error {
	// Payload format:
	// 0. nonce [32 bytes]
	// 1. next nonce [32 bytes]
	// 2. data [N bytes]
	if len(c.aesKey) == 0 {
		return fmt.Errorf("AES key is not derived yet")
	}
	payloadRaw := make([]byte, 32+len(data))
	copy(payloadRaw[0:8], c.currentLocalNonce) // current nonce
	// fill c.currentLocalNonce with random bytes for next use
	_, err := rand.Read(c.currentLocalNonce)
	if err != nil {
		return err
	}
	copy(payloadRaw[8:16], c.currentLocalNonce) // next nonce
	copy(payloadRaw[16:], data)
	encryptedPayload, err := encryptAESGCM(payloadRaw, c.aesKey)
	if err != nil {
		return err
	}
	frame := newFrame(1, encryptedPayload)
	return c.tcpClient.Send(frame.toBytes())
}
