package main

import (
	"fmt"

	"github.com/dabao-zhao/xrpc"
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
	c := xrpc.NewClientWithCodec(xrpc.NewGobCodec(), "127.0.0.1:9999")

	var sum int
	_ = c.Call("Int.Sum", &Args{A: 1, B: 2}, &sum)
	fmt.Println(sum)

	var reply MultiReply
	_ = c.Call("Int.Multi", &MultiArgs{A: &Args{1, 2}, B: &Args{3, 4}}, &reply)
	fmt.Println(reply.A, reply.B)
}
