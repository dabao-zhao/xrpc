package main

import (
	"net/http"

	"github.com/dabao-zhao/xrpc"
	"github.com/dabao-zhao/xrpc/jsonrpc"
)

type Int int

type Args struct {
	A int `json:"a"`
	B int `json:"b"`
}

func (i *Int) Sum(args *Args, reply *int) error {
	*reply = args.A + args.B
	return nil
}

func (i *Int) Sum2(args *[]int, reply *int) error {
	for _, arg := range *args {
		*reply += arg
	}
	return nil
}

type MultiArgs struct {
	A *Args `json:"aa"`
	B *Args `json:"bb"`
}

type MultiReply struct {
	A int `json:"aa"`
	B int `json:"bb"`
}

func (i *Int) Multi(args *MultiArgs, reply *MultiReply) error {
	reply.A = args.A.A * args.A.B
	reply.B = args.B.A * args.B.B
	return nil
}

func main() {
	s := xrpc.NewServerWithCodec(jsonrpc.NewJSONCodec())
	mineInt := new(Int)
	_ = s.Register(mineInt)
	go s.ServeTCP("127.0.0.1:9999")

	// 开启http
	_ = http.ListenAndServe(":9998", s)
}
