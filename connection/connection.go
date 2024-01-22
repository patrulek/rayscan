package connection

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/jsonrpc"
	"github.com/patrulek/rayscan/config"
)

type Connection struct {
	ConnectionInfo config.RPCNode
	RPCClient      *rpc.Client
	CooldownUntil  time.Time
}

type RPCPool struct {
	Connections []*Connection
	CurrentIdx  int
	mu          sync.RWMutex
}

func (r *RPCPool) Size() int {
	return len(r.Connections)
}

func (r *RPCPool) BaseConnection() Connection {
	for _, c := range r.Connections {
		if c.ConnectionInfo.RPCEndpoint == rpc.MainNetBeta_RPC {
			return *c
		}
	}

	panic("No base connection found!")
}

func (r *RPCPool) NamedConnection(name string) Connection {
	for _, c := range r.Connections {
		if c.ConnectionInfo.Name == name {
			return *c
		}
	}

	panic(fmt.Sprintf("No connection with name %s found!", name))
}

func (r *RPCPool) Client() *rpc.Client {
	r.mu.Lock()
	defer r.mu.Unlock()

	oldIdx := r.CurrentIdx

	for i := 0; r.Connections[oldIdx].CooldownUntil.After(time.Now()); i++ {
		oldIdx = (oldIdx + 1) % len(r.Connections)
		if i == len(r.Connections) {
			fmt.Printf("All connections are on cooldown, waiting for cooldown to end... Consider adding new RPC providers\n")
			time.Sleep(50 * time.Millisecond)
			i = 0
		}
	}

	r.CurrentIdx = (oldIdx + 1) % len(r.Connections)
	return r.Connections[oldIdx].RPCClient
}

func (r *RPCPool) Close() {
	for _, c := range r.Connections {
		c.RPCClient.Close()
	}

	r.Connections = nil
}

type ConnectionInfo struct {
	Name    string
	RPCNode string
	WSNode  string
}

func NewRPCClientPool(nodes map[string]config.RPCNode) (*RPCPool, error) {
	var rpcPool RPCPool

	initialLen := len(nodes)
	fmt.Printf("Checking connection list...\n")

	for k, v := range nodes {
		rpcClient := rpc.New(v.RPCEndpoint)

		ctx, cancel := context.WithTimeout(context.Background(), 2000*time.Millisecond)
		health, err := rpcClient.GetHealth(ctx)
		cancel()

		if err != nil || health != "ok" {
			reason := err.Error()
			code := -1
			var asErr *jsonrpc.RPCError
			if errors.As(err, &asErr) {
				reason = err.(*jsonrpc.RPCError).Message
				code = err.(*jsonrpc.RPCError).Code
			}

			fmt.Printf("Remove unhealthy connection: %s (reason: %s, code: %d)\n", v.Name, reason, code)
			rpcClient.Close()
			continue
		}

		v.Name = k
		fmt.Printf("Connection %s is healthy\n", v.Name)

		rpcPool.Connections = append(rpcPool.Connections, &Connection{
			ConnectionInfo: v,
			RPCClient:      rpcClient,
		})
	}

	if len(rpcPool.Connections) == 0 {
		return nil, fmt.Errorf("No healthy connections found!")
	}

	healthyConnectionNames := make([]string, len(rpcPool.Connections))
	for i, c := range rpcPool.Connections {
		healthyConnectionNames[i] = c.ConnectionInfo.Name
	}

	fmt.Printf("Connection list checked! %d/%d connections are ok [%s]\n", len(rpcPool.Connections), initialLen, strings.Join(healthyConnectionNames, ", "))
	return &rpcPool, nil
}
