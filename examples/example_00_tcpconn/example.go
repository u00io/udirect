package example00tcpconn

import (
	"fmt"
	"sync"
	"time"

	"github.com/u00io/udirect/tcpconn"
)

type Stat struct {
	TotalReceived int64
	TotalSent     int64
}

var srvStat Stat

var mtx sync.Mutex

var data1 = make([]byte, 100*1024*1024)
var data2 = make([]byte, 100*1024*1024)

func client() {
	c := tcpconn.NewClient()
	c.OnConnected = func(client *tcpconn.Client) {
	}
	c.OnReceived = func(client *tcpconn.Client, data []byte) {
	}
	c.OnDisconnected = func(client *tcpconn.Client) {
	}
	c.Start("127.0.0.1", 8452)
	for {
		// time.Sleep(1 * time.Millisecond)
		err := c.Send([]byte(data1))
		if err != nil {
			fmt.Println("Error sending data:", err)
		}
	}
}

func server() {
	srv := tcpconn.NewServer()
	srv.OnConnected = func(client *tcpconn.Client) {
	}
	srv.OnReceived = func(client *tcpconn.Client, data []byte) {
		mtx.Lock()
		srvStat.TotalReceived += int64(len(data))
		srvStat.TotalSent += int64(len(data))
		mtx.Unlock()
		err := client.Send([]byte(data))
		if err != nil {
			fmt.Println("Error sending data:", err)
		}
	}
	srv.OnDisconnected = func(client *tcpconn.Client) {
	}
	srv.Start(8452)
	for {
		time.Sleep(10 * time.Millisecond)
	}
}

func Run() {
	go server()
	go client()
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
