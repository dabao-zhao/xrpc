package xrpc

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"net/http"
)

var (
	_ Request  = &defaultRequest{}
	_ Response = &defaultResponse{}
)

type Request interface {
	GetMethod() string
	GetParams() []byte
	GetId() string
}

type Response interface {
	Error() error
	GetErrCode() int
	GetReply() []byte
	SetReqId(id string)
}

type defaultRequest struct {
	Method string
	Args   []byte
	Id     string
}

func (d *defaultRequest) GetMethod() string { return d.Method }
func (d *defaultRequest) GetParams() []byte { return d.Args }
func (d *defaultRequest) GetId() string     { return d.Id }

type defaultResponse struct {
	Reply   []byte
	Err     string
	ErrCode int
	Id      string
}

func (d *defaultResponse) Error() error {
	if d.Err == "" {
		return nil
	}
	return errors.New(d.Err)
}

func (d *defaultResponse) GetReply() []byte   { return d.Reply }
func (d *defaultResponse) GetErrCode() int    { return d.ErrCode }
func (d *defaultResponse) SetReqId(id string) { d.Id = id }

var (
	_ Codec = &gobCodec{}
)

func init() {
	gob.Register([]*defaultRequest{})
	gob.Register([]*defaultResponse{})

	gob.Register(&defaultRequest{})
	gob.Register(&defaultResponse{})
}

type Codec interface {
	ServerCodec
	ClientCodec
}

type ServerCodec interface {
	ReadRequest(data []byte) ([]Request, error)
	ReadRequestBody(reqBody []byte, data interface{}) error
	NewResponse(data interface{}) Response
	ErrResponse(errCode int, err error) Response
	EncodeResponses(v interface{}) ([]byte, error)
	Send(w http.ResponseWriter, statusCode int, b []byte) error
}

type ClientCodec interface {
	NewRequest(method string, argv interface{}) Request
	EncodeRequests(v interface{}) ([]byte, error)
	ReadResponse(data []byte) ([]Response, error)
	ReadResponseBody(respBody []byte, data interface{}) error
}

func NewGobCodec() Codec {
	codec := &gobCodec{}
	return codec
}

type gobCodec struct {
}

func (g *gobCodec) Encode(argv interface{}) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(buf)

	if err := enc.Encode(argv); err != nil {
		return nil, fmt.Errorf("g.enc.Encode(argv) got err: %v", err)
	}

	return buf.Bytes(), nil
}

func (g *gobCodec) Decode(data []byte, out interface{}) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)

	if err := dec.Decode(out); err != nil {
		return fmt.Errorf("[Decode] got err: %v", err)
	}

	return nil
}

func (g *gobCodec) NewResponse(data interface{}) Response {
	reply, err := g.Encode(data)
	if err != nil {
		log.Printf("[NewResponse] could not encode reply=%v, err=%v", data, err)
		return nil
	}
	resp := &defaultResponse{
		Reply:   reply,
		Err:     "",
		ErrCode: Success,
	}

	return resp
}

func (g *gobCodec) ErrResponse(errCode int, err error) Response {
	resp := &defaultResponse{
		Err:     err.Error(),
		ErrCode: errCode,
	}
	return resp
}

func (g *gobCodec) NewRequest(method string, data interface{}) Request {
	args, err := g.Encode(data)
	if err != nil {
		log.Printf("could not encode argv, err=%v", err)
		return nil
	}

	req := &defaultRequest{
		Method: method,
		Args:   args,
	}

	return req
}

func (g *gobCodec) ReadRequest(data []byte) ([]Request, error) {
	reqs := make([]Request, 0)
	if err := g.Decode(data, &reqs); err != nil {
		log.Printf("[ReadRequest] could not g.Decode(data, reqs), err=%v", err)
		return nil, err
	}

	return reqs, nil
}

func (g *gobCodec) ReadResponse(data []byte) ([]Response, error) {
	resps := make([]Response, 0)
	if err := g.Decode(data, &resps); err != nil {
		return nil, fmt.Errorf("could not decode response: %v", err)
	}
	return resps, nil
}

func (g *gobCodec) ReadResponseBody(respBody []byte, data interface{}) error {
	return g.Decode(respBody, data)
}

func (g *gobCodec) ReadRequestBody(reqBody []byte, data interface{}) error {
	return g.Decode(reqBody, data)
}

func (g *gobCodec) EncodeRequests(v interface{}) ([]byte, error) {
	return g.Encode(v)
}

func (g *gobCodec) EncodeResponses(v interface{}) ([]byte, error) {
	return g.Encode(v)
}

func (g *gobCodec) Send(w http.ResponseWriter, statusCode int, b []byte) error {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(statusCode)
	_, err := w.Write(b)
	return err
}
