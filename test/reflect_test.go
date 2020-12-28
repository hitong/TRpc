package test

import (
	"fmt"
	NetRpc "github.com/hitong/TRpc"
	"reflect"
	"testing"
)

type Test struct {
}

func (*Test) Hello() {
	fmt.Println("Hello reflect")
}

func Test_ReflectFunc(t *testing.T) {
	msg := NetRpc.TRpcMessage{
		ID:       "",
		Receiver: "",
		Service:  "Hello",
		Args:     nil,
		Future:   nil,
		Error:    nil,
	}
	v := reflect.ValueOf(&Test{}).MethodByName(msg.Service)
	if v.Kind() != reflect.Func {
		t.Fatal("not func")
	}
	vals := make([]reflect.Value, 0, len(msg.Args))
	for _, arg := range msg.Args {
		vals = append(vals, reflect.ValueOf(arg))
	}
	values := v.Call(vals) //方法调用并返回值
	for i := range values {
		fmt.Println(values[i])
	}
}
