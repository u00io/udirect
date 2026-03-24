package uapi

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/u00io/udirect/forms"
	"github.com/u00io/udirect/udirect"
)

type Client struct {
	udirectClient *udirect.Client
}

func NewClient(addr string, port int) *Client {
	var c Client
	c.udirectClient = udirect.NewClient(addr, port)
	return &c
}

func (c *Client) Start() {
	c.udirectClient.Start()
}

func (c *Client) Stop() {
	c.udirectClient.Stop()
}

func (c *Client) generateTrId() string {
	randomBytes := make([]byte, 8)
	rand.Read(randomBytes)
	return hex.EncodeToString(randomBytes)
}

func (c *Client) onFrameReceived(client *udirect.Client, frameData []byte) {
}

func (c *Client) Call(function string, form *forms.Form) (*forms.Form, error) {
	trId := c.generateTrId()
	form.SetFieldString("_FN", function)
	form.SetFieldString("_TRID", trId)

	c.udirectClient.Send(form.Serialize())

	return nil, nil

}
