package uapi

import (
	"strconv"
	"sync"
)

type ConnectionsPool struct {
	mtx     sync.Mutex
	clients map[string]*Client
}

var pool *ConnectionsPool

func init() {
	pool = &ConnectionsPool{
		clients: make(map[string]*Client),
	}
}

func GetClient(addr string, port int) *Client {
	pool.mtx.Lock()
	defer pool.mtx.Unlock()
	key := addr + ":" + strconv.Itoa(port)
	if client, exists := pool.clients[key]; exists {
		return client
	}
	client := NewClient(addr, port)
	client.Start()
	pool.clients[key] = client
	return client
}
