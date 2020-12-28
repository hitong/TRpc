package tRpc

import (
	"encoding/json"
	"errors"
	"net"
	"reflect"
	"sync"
	"time"

	"github.com/globalsign/mgo/bson"
	log "github.com/sirupsen/logrus"
)

type TRpcMgr struct {
	register map[string]interface{}
	mutex    sync.Mutex
	connMap  sync.Map
}

type TRpc struct {
	Handle   net.Conn
	in       *json.Decoder
	out      *json.Encoder
	Ch       chan *TRpcMessage
	close    chan bool
	BeClosed bool
	RetMap   sync.Map
	register map[string]interface{}
}

type TRpcMessage struct {
	ID       string
	Receiver string
	Service  string
	Args     []interface{}
	Future   []interface{}
	Error    error
}

const DefaultServerAddr = "192.168.92.167:12345"

func DefaultKeyFunc(keySrc string) interface{} {
	return keySrc
}

func NewTRpcMgr() *TRpcMgr {
	return &TRpcMgr{
		register: make(map[string]interface{}),
		mutex:    sync.Mutex{},
		connMap:  sync.Map{},
	}
}

func (t *TRpcMgr) Register(key string, val interface{}) {
	if t.register == nil {
		t.mutex.Lock()
		defer t.mutex.Unlock()
		if t.register == nil {
			t.register = make(map[string]interface{})
		}
	}
	t.register[key] = val
}

func (t *TRpcMgr) GetTRpc(key string) *TRpc {
	if v, ok := t.connMap.Load(key); ok {
		return v.(*TRpc)
	}
	return nil
}

func (t *TRpcMgr) NewTRpc(conn net.Conn) *TRpc {
	tRpc := &TRpc{
		Handle:   conn,
		in:       json.NewDecoder(conn),
		out:      json.NewEncoder(conn),
		Ch:       make(chan *TRpcMessage),
		close:    make(chan bool),
		BeClosed: false,
		RetMap:   sync.Map{},
		register: t.register,
	}
	t.connMap.Store(conn.RemoteAddr().String(), tRpc)
	return tRpc
}

func (t *TRpcMgr) Range(F func(key, value interface{}) bool) {
	t.connMap.Range(F)
}

func (t *TRpcMgr) DeleteKey(key string) {
	t.connMap.Delete(key)
}

func (tRpc *TRpc) Server() {
	go tRpc.sendLoop()
	go tRpc.receiveLoop()
}

func (tRpc *TRpc) sendLoop() {
	defer func() {
		log.Println("send loop exit")
	}()
	for {
		select {
		case data := <-tRpc.Ch:
			tRpc.out.Encode(data)
			//log.Println("send data ", data)
		case <-tRpc.close:
			return
		}
	}
}

type Caller struct {
}

var c Caller

func (c *Caller) HelloWorld(data map[string]interface{}) (map[string]interface{}, string) {
	log.Println("hello world")
	log.Println("data ", data)
	return data, "dddddddddddddd"
}

func (c *Caller) LogDebug() string {
	return "saaaaaaaaaa"
}

func (tRpc *TRpc) receiveLoop() {
	defer func() {
		log.Println("receive loop exit")
	}()
	for {
		receiveBuffer := &TRpcMessage{}
		if err := tRpc.in.Decode(receiveBuffer); err != nil {
			if tRpc.BeClosed == false {
				tRpc.Close()
			}
			return
		}
		if receiveBuffer.Service == "" {
			if v, ok := tRpc.RetMap.LoadAndDelete(receiveBuffer.ID); ok {
				ch := v.(chan interface{})
				//println(receiveBuffer.Future)
				ch <- receiveBuffer.Future
				close(ch)
			}
		} else {
			tRpc.CallFunc(receiveBuffer)
			receiveBuffer.Service = ""
			tRpc.Ch <- receiveBuffer
		}
	}
}

func (tRpc *TRpc) Close() {
	tRpc.BeClosed = true
	tRpc.close <- true
	tRpc.Handle.Close()
}

func (tRpc *TRpc) CallFunc(msg *TRpcMessage) {
	if tRpc.register[msg.Receiver] == nil {
		msg.Error = errors.New("Not registered Server " + msg.Receiver)
		return
	}
	v := reflect.ValueOf(tRpc.register[msg.Receiver]).MethodByName(msg.Service)
	if v.Kind() != reflect.Func {
		msg.Error = errors.New("Not Found Service " + msg.Service)
		return
	}
	vals := make([]reflect.Value, 0, len(msg.Args))
	for _, arg := range msg.Args {
		vals = append(vals, reflect.ValueOf(arg))
	}
	values := v.Call(vals)
	for _, v := range values {
		msg.Future = append(msg.Future, v.Interface())
	}
}

func (tRpc *TRpc) Send(receiver string, serviceName string, args ...interface{}) <-chan interface{} {
	if tRpc == nil || tRpc.BeClosed {
		log.Println("Connect not exist or has been closed")
		return nil
	}
	msg := TRpcMessage{
		ID:       bson.NewObjectId().Hex(),
		Service:  serviceName,
		Receiver: receiver,
		Args:     args,
		Future:   nil,
		Error:    nil,
	}
	retChan := make(chan interface{}, 1)
	tRpc.RetMap.Store(msg.ID, retChan)
	tRpc.Ch <- &msg

	time.AfterFunc(10*time.Second, func() {
		ii, _ := tRpc.RetMap.LoadAndDelete(msg.ID)
		if ii == nil {
			return
		}
		defer func() {
			log.Println("rpc time out")
			if r := recover(); r != nil {
			}
		}()

		close(retChan)
	})
	return retChan
}
