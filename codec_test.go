package xrpc

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJsonRequest(t *testing.T) {
	req := defaultRequest{
		Id:     "1",
		Method: "Int.Sum",
		Args:   []byte("arg"),
	}
	assert.Equal(t, req.Id, req.GetId())
	assert.Equal(t, req.Method, req.GetMethod())
	assert.Equal(t, req.Args, req.GetParams())
}

func TestJsonResponse(t *testing.T) {
	resp := defaultResponse{
		Reply:   []byte("reply"),
		Err:     "err",
		ErrCode: 1,
		Id:      "",
	}

	assert.Equal(t, errors.New(resp.Err), resp.Error())
	assert.Equal(t, resp.Reply, resp.GetReply())
	assert.Equal(t, nil, resp.GetResult())
	assert.Equal(t, resp.ErrCode, resp.GetErrCode())

	resp.SetReqId("2")
	assert.Equal(t, "2", resp.Id)
}

func TestNewGobCodec(t *testing.T) {
	codec := NewGobCodec()

	assert.Equal(t, &gobCodec{}, codec)
}

func TestGobCodec_NewResponse(t *testing.T) {
	codec := NewGobCodec()

	result := "data"
	resp := codec.NewResponse(result)

	g := gobCodec{}
	b, _ := g.Encode(result)
	assert.Equal(t, b, resp.GetReply())
	assert.Equal(t, nil, resp.GetResult())
	assert.Nil(t, resp.Error())
}

func TestGobCodec_NewRequest(t *testing.T) {
	codec := NewGobCodec()

	method := "Int.Sum"
	arg := "arg"
	req := codec.NewRequest(method, arg)

	assert.Equal(t, method, req.GetMethod())

	g := gobCodec{}
	b, _ := g.Encode(arg)
	assert.Equal(t, b, req.GetParams())
}

func TestGobCodec_ReadResponse(t *testing.T) {
	codec := NewGobCodec()

	resp := codec.NewResponse("data")
	g := gobCodec{}
	b, _ := g.Encode([]Response{resp})
	resps, err := codec.ReadResponse(b)
	assert.Nil(t, err)
	assert.Equal(t, []Response{resp}, resps)

	b, _ = g.Encode("")
	_, err = codec.ReadResponse(b)
	assert.NotNil(t, err)

	b, _ = g.Encode([]Response{resp})
	resps, err = codec.ReadResponse(b)
	assert.Nil(t, err)
	assert.Equal(t, []Response{resp}, resps)
}

func TestGobCodec_ReadRequest(t *testing.T) {
	codec := NewGobCodec()
	g := gobCodec{}

	method := "Int.Sum"
	arg := "arg"
	req := codec.NewRequest(method, arg)
	b, _ := g.Encode([]Request{req})

	requests, err := codec.ReadRequest(b)
	assert.Nil(t, err)
	assert.Equal(t, []Request{req}, requests)

	b, _ = g.Encode("")
	_, err = codec.ReadRequest(b)
	assert.NotNil(t, err)

	b, _ = g.Encode([]Request{req})

	requests, err = codec.ReadRequest(b)
	assert.Nil(t, err)
	assert.Equal(t, []Request{req}, requests)

}

func TestGobCodec_ReadRequestBody(t *testing.T) {
	codec := NewGobCodec()

	type Args struct {
		A int `json:"a"`
		B int `json:"b"`
	}

	out := new(Args)
	method := "Int.Sum"
	arg := &Args{
		A: 1,
		B: 2,
	}
	req := codec.NewRequest(method, arg)
	err := codec.ReadRequestBody(req.GetParams(), out)
	assert.Nil(t, err)
	assert.Equal(t, arg, out)
}

func TestGobCodec_ReadResponseBody(t *testing.T) {
	codec := NewGobCodec()

	data := "1"
	resp := codec.NewResponse(data)

	out := new(string)
	err := codec.ReadResponseBody(resp.GetReply(), out)
	assert.Nil(t, err)
	assert.Equal(t, &data, out)
}

func TestGobCodec_EncodeRequests(t *testing.T) {
	codec := NewGobCodec()
	data := "1"
	g := gobCodec{}
	b, _ := g.Encode(data)
	res, err := codec.EncodeRequests(data)
	assert.Nil(t, err)
	assert.Equal(t, b, res)
}

func TestGobCodec_EncodeResponses(t *testing.T) {
	codec := NewGobCodec()
	data := "1"
	g := gobCodec{}
	b, _ := g.Encode(data)
	res, err := codec.EncodeResponses(data)
	assert.Nil(t, err)
	assert.Equal(t, b, res)
}

func TestGobCodec_ErrResponse(t *testing.T) {
	codec := NewGobCodec()

	errCode := 1
	err := errors.New("1")
	resp := codec.ErrResponse(errCode, err)

	assert.Equal(t, err, resp.Error())
	assert.Equal(t, errCode, resp.GetErrCode())
}

func TestGobCodec_Send(t *testing.T) {

}
