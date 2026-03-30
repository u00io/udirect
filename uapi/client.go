package uapi

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/u00io/udirect/forms"
	"github.com/u00io/udirect/udirect"
)

type Client struct {
	udirectClient *udirect.Client
	mtx           sync.Mutex
	pendingCalls  map[string]chan *forms.Form
}

var (
	ErrCallTimeout = errors.New("call timeout")
)

func NewClient(addr string, port int) *Client {
	var c Client
	c.udirectClient = udirect.NewClient(addr, port)
	c.udirectClient.OnFrameReceived = c.onFrameReceived
	c.pendingCalls = make(map[string]chan *forms.Form)
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
	form, err := forms.ParseForm(frameData)
	if err != nil {
		return
	}

	trId := form.GetFieldString("_TRID")
	if trId == "" {
		return
	}

	c.mtx.Lock()
	ch, ok := c.pendingCalls[trId]
	c.mtx.Unlock()
	if !ok {
		return
	}

	select {
	case ch <- form:
	default:
	}
}

func (c *Client) Call(function string, form *forms.Form) (*forms.Form, error) {
	if form == nil {
		form = forms.NewForm()
	}

	trId := c.generateTrId()
	responseCh := make(chan *forms.Form, 1)

	c.mtx.Lock()
	c.pendingCalls[trId] = responseCh
	c.mtx.Unlock()

	defer func() {
		c.mtx.Lock()
		delete(c.pendingCalls, trId)
		c.mtx.Unlock()
	}()

	form.SetFieldString("_FN", function)
	form.SetFieldString("_TRID", trId)

	err := c.udirectClient.Send(form.Serialize())
	if err != nil {
		return nil, err
	}

	select {
	case response := <-responseCh:
		return response, nil
	case <-time.After(3 * time.Second):
		return nil, ErrCallTimeout
	}

}
