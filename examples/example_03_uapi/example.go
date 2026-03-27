package example03uapi

import (
	"fmt"
	"sync"
	"time"

	"github.com/u00io/udirect/forms"
	"github.com/u00io/udirect/uapi"
)

var dataToSend = make([]byte, 1*1024)

type Stat struct {
	ClientsSent     int64
	ServerProcessed int64
}

var mtx sync.Mutex
var srvStat Stat

func runClient(addr string) {
	client := uapi.NewClient(addr, 13245)
	client.Start()

	form := forms.NewForm()

	for {
		fmt.Println("Call ping")
		resp, err := client.Call("ping", form)
		if err == nil {
			respStr := resp.GetFieldString("message")
			fmt.Println("			Response:", respStr)
			mtx.Lock()
			srvStat.ClientsSent++
			mtx.Unlock()
		}
		if err != nil {
			fmt.Println("Error calling ping:", err)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

type Processor struct {
}

var counter int

func (p *Processor) Process(form *forms.Form) (*forms.Form, error) {
	resp := forms.NewForm()
	resp.SetFieldString("message", "pong")
	fmt.Println("			Received ping, sending pong")
	counter++
	if counter%10 == 0 {
		return nil, fmt.Errorf("simulated error")
	}
	return resp, nil
}

func runServer() {
	srv := uapi.NewServer()
	srv.SetProcessor(&Processor{})
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

		_ = serverSpeed
		_ = clientSpeed

		//fmt.Println("SERVER", serverSpeed, "CLIENTS", clientSpeed)
		mtx.Unlock()
	}
}
