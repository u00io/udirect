package example02u00test

import (
	"fmt"
	"sync"
	"time"

	"github.com/u00io/udirect/udirect"
)

var dataToSend = make([]byte, 1*1024)

type Stat struct {
	ClientsSent     int64
	ServerProcessed int64
}

var mtx sync.Mutex
var srvStat Stat

func runClient(addr string) {
	client := udirect.NewClient(addr, 13245, "")
	client.SetMaxInputBufferSize(1 * 1024 * 1024)
	client.OnConnected = func(client *udirect.Client) {
	}
	client.OnFrameReceived = func(client *udirect.Client, frame []byte) {
	}
	client.OnDisconnected = func(client *udirect.Client) {
	}
	client.Start()

	for {
		err := client.Send([]byte(dataToSend))
		mtx.Lock()
		srvStat.ClientsSent++
		mtx.Unlock()
		if err != nil {
			fmt.Println("Error sending data:", err)
		}
	}
}

func runServer() {
	srv := udirect.NewServer("")
	srv.SetMaxInputBufferSize(100 * 1024)
	srv.OnConnected = func(client *udirect.Client) {
	}
	srv.OnFrameReceived = func(client *udirect.Client, frame []byte) {
		mtx.Lock()
		srvStat.ServerProcessed++
		mtx.Unlock()
	}
	srv.OnDisconnected = func(client *udirect.Client) {
	}
	srv.Start(13245)
	for {
		time.Sleep(1000 * time.Millisecond)
	}
}

func Run(server bool, addr string) {
	if server {
		go runServer()
	} else {
		go runClient(addr)
	}

	lastSrvTotalReceived := srvStat.ServerProcessed
	lastSrvTotalSent := srvStat.ClientsSent
	dtLastSrvStat := time.Now()

	for {
		time.Sleep(1000 * time.Millisecond)
		mtx.Lock()
		serverProcessed := srvStat.ServerProcessed - lastSrvTotalReceived
		clientSent := srvStat.ClientsSent - lastSrvTotalSent
		lastSrvTotalReceived = srvStat.ServerProcessed
		lastSrvTotalSent = srvStat.ClientsSent

		serverSpeed := int64(float64(serverProcessed) / time.Since(dtLastSrvStat).Seconds())
		clientSpeed := int64(float64(clientSent) / time.Since(dtLastSrvStat).Seconds())
		dtLastSrvStat = time.Now()

		fmt.Println("SERVER", serverSpeed, "CLIENTS", clientSpeed)
		mtx.Unlock()
	}
}
