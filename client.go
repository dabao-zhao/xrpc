package xrpc

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"reflect"
	"time"

	"github.com/dabao-zhao/xrpc/proto"
)

func NewClientWithCodec(codec ClientCodec, tcpAddr string) *Client {
	if codec == nil {
		codec = NewGobCodec()
	}

	return &Client{
		tcpAddr: tcpAddr,
		codec:   codec,
	}
}

type Client struct {
	tcpAddr string

	codec ClientCodec

	tcpConn net.Conn
}

func (c *Client) Call(method string, args, reply interface{}) error {
	req := c.codec.NewRequest(method, args)
	resps := make([]Response, 0)
	if err := c.callTcp([]Request{req}, &resps); err != nil {
		return err
	}

	resp := resps[0]
	if err := c.codec.ReadResponseBody(resp.GetReply(), reply); err != nil {
		return err
	}
	return nil
}

func (c *Client) CallBatch(reqs []Request, reply interface{}) error {
	t := reflect.TypeOf(c.codec)
	if t.Kind() == reflect.Ptr && t.Elem().Name() != "jsonCodec" {
		return errors.New("only jsonrpc support CallBatch")
	}
	r := reflect.TypeOf(reply)
	if r.Kind() != reflect.Ptr || (r.Elem().Kind() != reflect.Slice && r.Elem().Kind() != reflect.Array) {
		return errors.New("multi reply should be array or slice pointer")
	}

	resps := make([]Response, len(reqs))
	if err := c.callTcp(reqs, &resps); err != nil {
		return err
	}
	var results []interface{}
	for _, resp := range resps {
		results = append(results, resp.GetResult())
	}

	respBody, err := c.codec.EncodeResponses(results)
	if err != nil {
		return err
	}
	if err := c.codec.ReadResponseBody(respBody, reply); err != nil {
		return err
	}

	return nil
}

func (c *Client) callTcp(reqs []Request, resps *[]Response) (err error) {
	if err = c.valid(); err != nil {
		return err
	}

	var (
		wr    = bufio.NewWriter(c.tcpConn)
		rr    = bufio.NewReader(c.tcpConn)
		pSend = proto.New()
		pRec  = proto.New()
	)

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	select {
	case <-timeoutCtx.Done():
		return errors.New("timeout")
	default:
		if pSend.Body, err = c.codec.EncodeRequests(&reqs); err != nil {
			return err
		}

		if err := pSend.WriteTCP(wr); err != nil {
			return err
		}
		_ = wr.Flush()

		if err := pRec.ReadTCP(rr); err != nil {
			return err
		}

		*resps, err = c.codec.ReadResponse(pRec.Body)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) Close() {
	if c.tcpConn == nil {
		return
	}
	if err := c.tcpConn.Close(); err != nil {
		log.Printf("could not close c.tcpConn, err=%v", err)
	}
}

func (c *Client) valid() error {
	if c.codec == nil {
		return errors.New("client has an empty codec")
	}

	if c.tcpConn == nil {
		conn, err := net.Dial("tcp", c.tcpAddr)
		if err != nil {
			return fmt.Errorf("net.Dial tcp get err: %v", err)
		}
		c.tcpConn = conn
	}
	return nil
}
