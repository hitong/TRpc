package client

import (
	"context"
	"log"
	"net"
	"time"

	"github.com/hitong/tRpc"
)

var tRpc *netRpc.TRpc

func DialAndServer(mgr *netRpc.TRpcMgr, addr string) (*netRpc.TRpc, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	ret := mgr.NewTRpc(conn)
	ret.Server()
	return ret, err
}

func DialAndServerWithReconnection(ctx context.Context, mgr *netRpc.TRpcMgr, addr string, t time.Duration) <-chan *netRpc.TRpc {
	var tRpc *netRpc.TRpc
	for {
		if tRpc, _ = DialAndServer(mgr, addr); tRpc != nil {
			break
		}
	}
	ch := make(chan *netRpc.TRpc, 1)
	ch <- tRpc
	tick := time.Tick(t)
	go func() {
		for {
			select {
			case <-tick:
				if tRpc == nil || tRpc.BeClosed {
					log.Println("reconnect")
					if tRpc, _ = DialAndServer(mgr, addr); tRpc != nil {
						ch <- tRpc
					}
				}
			case <-ctx.Done():
				log.Println("exit reconnect")
				close(ch)
				return
			}
		}
	}()
	return ch
}

func Rpc(receiver string, serviceName string, args ...interface{}) <-chan interface{} {
	return tRpc.Send(receiver, serviceName, args...)
}
