package uapi

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/u00io/udirect/forms"
	"github.com/u00io/udirect/stats"
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

func NewClient(addr string, port int, privateKeyHex string) *Client {
	stats.Inc("uapi.client_new")
	var c Client
	c.udirectClient = udirect.NewClient(addr, port, privateKeyHex)
	c.udirectClient.OnFrameReceived = c.onFrameReceived
	c.pendingCalls = make(map[string]chan *forms.Form)
	return &c
}

func (c *Client) Start() {
	stats.Inc("uapi.client_start")
	c.udirectClient.Start()
}

func (c *Client) Stop() {
	stats.Inc("uapi.client_stop")
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
		stats.Inc("uapi.client_parse_error")
		return
	}

	trId := form.GetFieldString("_TRID")
	if trId == "" {
		stats.Inc("uapi.client_no_trid")
		return
	}

	c.mtx.Lock()
	ch, ok := c.pendingCalls[trId]
	c.mtx.Unlock()
	if !ok {
		stats.Inc("uapi.client_no_pending_call")
		return
	}

	select {
	case ch <- form:
	default:
	}
}

func (c *Client) RemotePublicKey() string {
	return c.udirectClient.RemotePublicKey()
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
		stats.Inc("uapi.client_send_error")
		return nil, err
	}

	select {
	case response := <-responseCh:
		{
			stats.Inc("uapi.client_call_success")
			return response, nil
		}
	case <-time.After(3 * time.Second):
		{
			stats.Inc("uapi.client_call_timeout")
			return nil, ErrCallTimeout
		}
	}

}
