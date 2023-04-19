package main

import (
	"github.com/dabao-zhao/xrpc"
	"github.com/dabao-zhao/xrpc/jsonrpc"
)

type Args struct {
	A int `json:"a"`
	B int `json:"b"`
}

type MultiArgs struct {
	A *Args `json:"aa"`
	B *Args `json:"bb"`
}

type MultiReply struct {
	A int `json:"aa"`
	B int `json:"bb"`
}

func main() {
	c := xrpc.NewClientWithCodec(jsonrpc.NewJSONCodec(), "127.0.0.1:9999")

	var sum int
	c.Call("Int.Sum", &Args{A: 1, B: 2}, &sum)
	println(sum)

	var reply MultiReply
	c.Call("Int.Multi", &MultiArgs{A: &Args{1, 2}, B: &Args{3, 4}}, &reply)
	println(reply.A, reply.B)
}
