package uapi

import (
	"sync"

	"github.com/u00io/udirect/forms"
	"github.com/u00io/udirect/stats"
	"github.com/u00io/udirect/udirect"
)

type IProcessor interface {
	Process(client *udirect.Client, form *forms.Form) (*forms.Form, error)
}

type Server struct {
	udirectServer *udirect.Server
	mtx           sync.RWMutex
	processor     IProcessor
}

func NewServer(privateKeyHex string) *Server {
	stats.Inc("uapi.server_new")
	var c Server
	c.udirectServer = udirect.NewServer(privateKeyHex)
	c.udirectServer.OnFrameReceived = c.OnFrameReceived
	return &c
}

func (c *Server) SetProcessor(processor IProcessor) {
	c.mtx.Lock()
	c.processor = processor
	c.mtx.Unlock()
}

func (c *Server) Start(port int) error {
	stats.Inc("uapi.server_start")
	return c.udirectServer.Start(port)
}

func (c *Server) Stop() error {
	stats.Inc("uapi.server_stop")
	return c.udirectServer.Stop()
}

func (c *Server) OnConnected(client *udirect.Client) {
}

func (c *Server) OnDisconnected(client *udirect.Client) {
}

func (c *Server) OnFrameReceived(client *udirect.Client, frameData []byte) {
	go c.thProcessFrame(client, frameData)
}

func (c *Server) thProcessFrame(client *udirect.Client, frameData []byte) {
	stats.Inc("goroutine.uapi.server_process_frame")
	defer stats.Dec("goroutine.uapi.server_process_frame")

	form, err := forms.ParseForm(frameData)
	if err != nil {
		return
	}

	c.mtx.RLock()
	processor := c.processor
	c.mtx.RUnlock()
	if processor == nil {
		stats.Inc("uapi.server_no_processor")
		return
	}

	trId := form.GetFieldString("_TRID")
	function := form.GetFieldString("_FN")
	responseForm, err := processor.Process(client, form)
	if err != nil {
		stats.Inc("uapi.server_process_error")
		client.Stop()
		return
	}
	if responseForm == nil {
		responseForm = forms.NewForm()
	}
	responseForm.SetFieldString("_TRID", trId)
	responseForm.SetFieldString("_FN", function)
	responseFrameData := responseForm.Serialize()
	_ = client.Send(responseFrameData)
}
