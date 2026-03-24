package uapi

import (
	"github.com/u00io/udirect/forms"
	"github.com/u00io/udirect/udirect"
)

type IProcessor interface {
	Process(form *forms.Form) (*forms.Form, error)
}

type Server struct {
	udirectServer *udirect.Server
	processor     IProcessor
}

func NewServer() *Server {
	var c Server
	c.udirectServer = udirect.NewServer()
	c.udirectServer.OnFrameReceived = c.OnFrameReceived
	return &c
}

func (c *Server) SetProcessor(processor IProcessor) {
	c.processor = processor
}

func (c *Server) Start(port int) error {
	return c.udirectServer.Start(port)
}

func (c *Server) Stop() error {
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
	form, err := forms.ParseForm(frameData)
	if err != nil {
		return
	}
	trId := form.GetFieldString("_TRID")
	function := form.GetFieldString("_FN")
	responseForm, err := c.processor.Process(form)
	if err != nil {
		return
	}
	responseForm.SetFieldString("_TRID", trId)
	responseForm.SetFieldString("_FN", function)
	responseFrameData := responseForm.Serialize()
	client.Send(responseFrameData)
}
