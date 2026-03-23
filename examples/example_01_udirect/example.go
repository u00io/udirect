package example01udirect

import (
	"fmt"
	"sync"
	"time"

	"github.com/u00io/udirect/udirect"
)

var dataToSend = make([]byte, 10*1024*1024)

type Stat struct {
	TotalReceived int64
	TotalSent     int64
}

var mtx sync.Mutex
var srvStat Stat

func runClient() {
	client := udirect.NewClient("127.0.0.1", 13245)
	client.SetMaxInputBufferSize(200 * 1024 * 1024)
	client.OnConnected = func(client *udirect.Client) {
	}
	client.OnFrameReceived = func(client *udirect.Client, frame []byte) {
		client.Send(frame)
	}
	client.OnDisconnected = func(client *udirect.Client) {
	}
	client.Start()

	for {
		err := client.Send([]byte(dataToSend))
		if err != nil {
			fmt.Println("Error sending data:", err)
		}
	}
}

func runServer() {
	srv := udirect.NewServer()
	srv.SetMaxInputBufferSize(200 * 1024 * 1024)
	srv.OnConnected = func(client *udirect.Client) {
		fmt.Println("Client connected:", client.ID())
	}

	srv.OnFrameReceived = func(client *udirect.Client, frame []byte) {
		mtx.Lock()
		srvStat.TotalReceived += int64(len(frame))
		srvStat.TotalSent += int64(len(frame))
		mtx.Unlock()
	}

	srv.OnDisconnected = func(client *udirect.Client) {
		fmt.Println("Client disconnected:", client.ID())
	}

	srv.Start(13245)

	for {
		time.Sleep(1000 * time.Millisecond)
	}
}

func Run() {
	go runServer()
	go runClient()

	lastSrvTotalReceived := srvStat.TotalReceived
	lastSrvTotalSent := srvStat.TotalSent
	dtLastSrvStat := time.Now()

	for {
		time.Sleep(1000 * time.Millisecond)
		mtx.Lock()
		recvBytes := srvStat.TotalReceived - lastSrvTotalReceived
		sentBytes := srvStat.TotalSent - lastSrvTotalSent
		lastSrvTotalReceived = srvStat.TotalReceived
		lastSrvTotalSent = srvStat.TotalSent

		rcvSpeed := int64(float64(recvBytes) / time.Since(dtLastSrvStat).Seconds())
		sndSpeed := int64(float64(sentBytes) / time.Since(dtLastSrvStat).Seconds())
		dtLastSrvStat = time.Now()

		fmt.Println("RCV", rcvSpeed/(1024*1024), "MB/s,", "SND", sndSpeed/(1024*1024), "MB/s")
		mtx.Unlock()
	}

}
