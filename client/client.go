package client

import (
	"context"
	"github.com/hitong/tRpc"
	"log"
	"net"
	"time"
)

var gRpc *tRpc.TRpc

func DialAndServer(mgr *tRpc.TRpcMgr, addr string) (*tRpc.TRpc, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	ret := mgr.NewTRpc(conn)
	ret.Server()
	return ret, err
}

func DialAndServerWithReconnection(ctx context.Context, mgr *tRpc.TRpcMgr, addr string, t time.Duration) <-chan *tRpc.TRpc {
	var gRpc *tRpc.TRpc
	for {
		if gRpc, _ = DialAndServer(mgr, addr); gRpc != nil {
			break
		}
	}
	ch := make(chan *tRpc.TRpc, 1)
	ch <- gRpc
	tick := time.Tick(t)
	go func() {
		for {
			select {
			case <-tick:
				if gRpc == nil || gRpc.BeClosed {
					log.Println("reconnect")
					if gRpc, _ = DialAndServer(mgr, addr); gRpc != nil {
						ch <- gRpc
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
	return gRpc.Send(receiver, serviceName, args...)
}
