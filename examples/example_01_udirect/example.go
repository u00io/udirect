package example01udirect

import (
	"fmt"
	"time"

	"github.com/u00io/udirect/udirect"
)

func runClient() {
	client := udirect.NewClient("127.0.0.1", "", 13245)
	client.OnConnected = func(client *udirect.Client) {
		println("Connected to server")
	}
	client.OnFrameReceived = func(client *udirect.Client, frame []byte) {
		println("Received frame from server, length:", len(frame))
	}
	client.OnDisconnected = func(client *udirect.Client) {
		println("Disconnected from server")
	}
	client.Start()

	for {
		err := client.Send([]byte("123"))
		if err != nil {
			fmt.Println("Send error:", err)
		}
		time.Sleep(1000 * time.Millisecond)
	}
}

func runServer() {
	srv := udirect.NewServer()
	srv.OnConnected = func(client *udirect.Client) {
		fmt.Println("Client connected:", client.ID())
	}

	srv.OnFrameReceived = func(client *udirect.Client, frame []byte) {
		fmt.Println("Received frame from client:", client.ID(), "Frame length:", len(frame))
		fmt.Println("Frame:", string(frame))
	}

	srv.OnDisconnected = func(client *udirect.Client) {
		fmt.Println("Client disconnected:", client.ID())
	}

	srv.Start(13245)

	for {
		time.Sleep(100 * time.Millisecond)
	}
}

func Run() {
	go runServer()
	go runClient()

	for {
		time.Sleep(100 * time.Millisecond)
	}

}
