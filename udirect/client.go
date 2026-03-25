package udirect

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/u00io/udirect/tcpconn"
)

type Client struct {
	mtx                      sync.Mutex
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

	inputData       []byte // buffer for incoming data
	inputDataOffset int    // offset for the input data buffer

	stats ClientStats

	OnConnected     func(client *Client)              // callback for connection established
	OnFrameReceived func(client *Client, data []byte) // callback for received data
	OnDisconnected  func(client *Client)              // callback for connection closed
}

type ClientStats struct {
	BytesSent     int64
	BytesReceived int64
}

const (
	defaultMaxInputBufferSize = 10 * 1024 * 1024 // 1MB
)

func NewClient(addr string, port int) *Client {
	var c Client
	c.addr = addr
	c.port = port
	c.maxInputBufferSize = defaultMaxInputBufferSize
	c.GenerateLocalPrivateKey()
	c.tcpClient = tcpconn.NewClient(c.onTcpClientConnected, c.onTcpClientReceived, c.onTcpClientDisconnected)
	c.currentLocalNonce = make([]byte, 8)
	rand.Read(c.currentLocalNonce)
	c.inputData = make([]byte, defaultMaxInputBufferSize)
	c.inputDataOffset = 0
	return &c
}

func newClientFromTcpClient(tcpClient *tcpconn.Client, onConnected func(*Client), onFrameReceived func(*Client, []byte), onDisconnected func(*Client)) *Client {
	var c Client
	c.tcpClient = tcpClient
	c.maxInputBufferSize = defaultMaxInputBufferSize
	c.OnConnected = onConnected
	c.OnFrameReceived = onFrameReceived
	c.OnDisconnected = onDisconnected
	c.currentLocalNonce = make([]byte, 8)
	rand.Read(c.currentLocalNonce)
	c.inputData = make([]byte, defaultMaxInputBufferSize)
	c.inputDataOffset = 0
	return &c
}

func (c *Client) GenerateLocalPrivateKey() error {
	privateKey, err := GeneratePrivateKey()
	if err != nil {
		return err
	}
	c.mtx.Lock()
	c.privateKey = privateKey
	c.publicKey = ed25519.PublicKey(privateKey[32:])
	c.mtx.Unlock()
	return nil
}

func (c *Client) SetLocalPrivateKey(privateKeyHex string) error {
	privateKey, err := hex.DecodeString(privateKeyHex)
	if err != nil || len(privateKey) != ed25519.PrivateKeySize {
		return fmt.Errorf("invalid private key hex string")
	}
	c.mtx.Lock()
	c.privateKey = privateKey
	c.publicKey = ed25519.PublicKey(privateKey[32:])
	c.mtx.Unlock()
	return nil
}

func (c *Client) SetMaxInputBufferSize(size int) {
	c.mtx.Lock()
	c.maxInputBufferSize = size
	c.mtx.Unlock()
}

func (c *Client) Start() {
	c.tcpClient.Start(c.addr, c.port)
}

func (c *Client) Stop() {
	c.tcpClient.Stop()
}

func (c *Client) ID() int64 {
	return c.tcpClient.ID()
}

func (c *Client) onTcpClientConnected(client *tcpconn.Client) {
	var err error

	c.mtx.Lock()
	c.transportPrivateKey, c.transportPublicKey, err = GenerateCurve25519KeyPair()
	if err != nil {
		c.mtx.Unlock()
		return
	}
	payload := make([]byte, 128)
	copy(payload[0:32], c.publicKey)
	copy(payload[32:64], c.transportPublicKey)
	signature := ed25519.Sign(ed25519.PrivateKey(c.privateKey), c.transportPublicKey)
	copy(payload[64:64+64], signature) // 64 bytes
	handShakeFrame := newFrame(0, payload)
	c.mtx.Unlock()

	err = client.Send(handShakeFrame.toBytes())
	if err != nil {
		return
	}
}

func (c *Client) onTcpClientDisconnected(client *tcpconn.Client) {
	c.mtx.Lock()
	c.aesKey = nil
	c.transportPrivateKey = nil
	c.transportPublicKey = nil
	c.remotePublicKey = nil
	c.transportRemotePublicKey = nil
	c.inputDataOffset = 0
	onDisconnected := c.OnDisconnected
	c.mtx.Unlock()
	if onDisconnected != nil {
		onDisconnected(c)
	}
}

func (c *Client) onTcpClientReceived(client *tcpconn.Client, data []byte) {
	c.mtx.Lock()
	if c.inputDataOffset+len(data) > c.maxInputBufferSize {
		c.mtx.Unlock()
		client.Stop()
		return
	}
	copy(c.inputData[c.inputDataOffset:], data)
	c.inputDataOffset += len(data)
	if c.inputDataOffset < 4 {
		c.mtx.Unlock()
		return
	}

	minFrameLength := uint32(16)
	maxFrameLength := uint32(c.maxInputBufferSize)
	frameLength := binary.LittleEndian.Uint32(c.inputData[0:4])
	if frameLength > maxFrameLength {
		c.mtx.Unlock()
		client.Stop()
		return
	}
	if frameLength < minFrameLength {
		c.mtx.Unlock()
		client.Stop()
		return
	}

	receivedFrames := make([]*frame, 0)
	offset := 0
	for {
		if c.inputDataOffset-offset < 4 {
			break
		}
		frameLength = binary.LittleEndian.Uint32(c.inputData[offset : offset+4])
		if frameLength > maxFrameLength {
			c.mtx.Unlock()
			client.Stop()
			return
		}
		if frameLength < minFrameLength {
			c.mtx.Unlock()
			client.Stop()
			return
		}
		if uint32(c.inputDataOffset) < frameLength {
			break
		}

		frameData := make([]byte, frameLength)
		copy(frameData, c.inputData[offset:offset+int(frameLength)])
		offset += int(frameLength)
		frame, err := frameFromBytes(frameData)
		if err != nil {
			continue
		}
		receivedFrames = append(receivedFrames, frame)
	}
	if offset < c.inputDataOffset {
		copy(c.inputData[0:], c.inputData[offset:c.inputDataOffset])
	}
	c.inputDataOffset -= offset
	c.mtx.Unlock()

	for _, frame := range receivedFrames {
		switch frame.Type {
		case 0: // handshake
			if len(frame.Payload) == 128 {
				c.mtx.Lock()
				if c.aesKey != nil {
					c.mtx.Unlock()
					continue
				}
				c.mtx.Unlock()

				remotePublicKey := frame.Payload[0:32]
				transportRemotePublicKey := frame.Payload[32:64]
				signature := frame.Payload[64:128]
				if !ed25519.Verify(ed25519.PublicKey(remotePublicKey), transportRemotePublicKey, signature) {
					continue
				}
				c.mtx.Lock()
				c.remotePublicKey = remotePublicKey
				c.transportRemotePublicKey = transportRemotePublicKey
				c.mtx.Unlock()
				aesKey, err := deriveAESKey(c.transportPrivateKey, c.transportRemotePublicKey)
				if err != nil {
					continue
				}
				c.mtx.Lock()
				c.aesKey = aesKey
				onConnected := c.OnConnected
				c.mtx.Unlock()
				fmt.Println("Handshake successful, AES key derived:", hex.EncodeToString(c.aesKey))
				if onConnected != nil {
					onConnected(c)
				}
			}
		case 1: // encrypted data
			c.mtx.Lock()
			aesKey := make([]byte, len(c.aesKey))
			copy(aesKey, c.aesKey)
			expectedRemoteNonce := make([]byte, len(c.expectedRemoteNonce))
			copy(expectedRemoteNonce, c.expectedRemoteNonce)
			onFrameReceived := c.OnFrameReceived
			c.mtx.Unlock()

			if len(aesKey) != 32 {
				continue
			}

			decryptedData, err := decryptAESGCM(frame.Payload, aesKey)
			if err != nil {
				continue
			}

			if len(decryptedData) >= 16 {
				nonce := decryptedData[0:8]
				nextNonce := decryptedData[8:16]
				data := decryptedData[16:]
				if len(expectedRemoteNonce) > 0 {
					if !equalBytes(nonce, expectedRemoteNonce) {
						client.Stop()
						continue
					}
					c.mtx.Lock()
					c.expectedRemoteNonce = nextNonce
					c.mtx.Unlock()
				} else {
					c.mtx.Lock()
					c.expectedRemoteNonce = nextNonce
					c.mtx.Unlock()
				}

				if onFrameReceived != nil {
					onFrameReceived(c, data)
				}
			}
		}
	}
}

func (c *Client) Send(data []byte) error {
	// Payload format:
	// 0. nonce [8 bytes]
	// 1. next nonce [8 bytes]
	// 2. data [N bytes]

	// getting AES key and current nonce with lock
	aesKey := make([]byte, 0)
	// Wait until AES key is derived
	for i := 0; i < 10; i++ {
		c.mtx.Lock()
		if len(c.aesKey) == 32 {
			aesKey = make([]byte, len(c.aesKey))
			copy(aesKey, c.aesKey)
			c.mtx.Unlock()
			break
		}
		c.mtx.Unlock()
		if len(aesKey) == 32 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if len(aesKey) == 0 {
		return fmt.Errorf("AES key not derived yet")
	}

	payloadRaw := make([]byte, 8+8+len(data))
	c.mtx.Lock()
	copy(payloadRaw[0:8], c.currentLocalNonce) // current nonce
	// fill c.currentLocalNonce with random bytes for next use
	_, err := rand.Read(c.currentLocalNonce)
	if err != nil {
		c.mtx.Unlock()
		return err
	}
	copy(payloadRaw[8:16], c.currentLocalNonce) // next nonce
	c.mtx.Unlock()
	copy(payloadRaw[16:], data)
	encryptedPayload, err := encryptAESGCM(payloadRaw, aesKey)
	if err != nil {
		return err
	}
	frame := newFrame(1, encryptedPayload)
	return c.tcpClient.Send(frame.toBytes())
}
