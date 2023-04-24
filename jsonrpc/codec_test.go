package jsonrpc

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/dabao-zhao/xrpc"
	"github.com/stretchr/testify/assert"
)

func TestJsonRequest(t *testing.T) {
	req := jsonRequest{
		Id:     "1",
		Method: "Int.Sum",
		Args: struct {
			A int
			B int
		}{
			1, 2,
		},
		Version: "2.0",
	}
	assert.Equal(t, req.Id, req.GetId())
	assert.Equal(t, req.Method, req.GetMethod())

	b, _ := json.Marshal(req.Args)
	assert.Equal(t, b, req.GetParams())
}

func TestJsonResponse(t *testing.T) {
	resp := jsonResponse{
		Id: "1",
		Err: &xrpc.Error{
			ErrCode: 1,
			ErrMsg:  "1",
		},
		Result: struct {
			A int
			B int
		}{
			1, 2,
		},
		Version: "2.0",
	}
	assert.Equal(t, resp.Err, resp.Error())
	b, _ := json.Marshal(resp.Result)
	assert.Equal(t, b, resp.GetReply())
	assert.Equal(t, resp.Result, resp.GetResult())
	assert.Equal(t, resp.Err.ErrCode, resp.GetErrCode())

	resp.Err = nil
	assert.Equal(t, xrpc.Success, resp.GetErrCode())

	resp.SetReqId("2")
	assert.Equal(t, "2", resp.Id)
}

func TestNewJSONCodec(t *testing.T) {
	codec := NewJSONCodec()

	assert.Equal(t, &jsonCodec{}, codec)
}

func TestJsonCodec_NewResponse(t *testing.T) {
	codec := NewJSONCodec()

	result := "data"
	resp := codec.NewResponse(result)

	b, _ := json.Marshal(result)
	assert.Equal(t, b, resp.GetReply())
	assert.Equal(t, result, resp.GetResult())
	assert.Nil(t, resp.Error())
}

func TestJsonCodec_NewRequest(t *testing.T) {
	codec := NewJSONCodec()

	method := "Int.Sum"
	arg := "arg"
	req := codec.NewRequest(method, arg)

	assert.Equal(t, method, req.GetMethod())

	b, _ := json.Marshal(arg)
	assert.Equal(t, b, req.GetParams())

	assert.NotEqual(t, "", req.GetId())
}

func TestJsonCodec_ReadResponse(t *testing.T) {
	codec := NewJSONCodec()

	resp := codec.NewResponse("data")
	b, _ := json.Marshal(resp)

	resps, err := codec.ReadResponse(b)
	assert.Nil(t, err)
	assert.Equal(t, []xrpc.Response{resp}, resps)

	b, _ = json.Marshal("")
	_, err = codec.ReadResponse(b)
	assert.NotNil(t, err)

	b, _ = json.Marshal([]xrpc.Response{resp})
	resps, err = codec.ReadResponse(b)
	assert.Nil(t, err)
	assert.Equal(t, []xrpc.Response{resp}, resps)
}

func TestJsonCodec_ReadRequest(t *testing.T) {
	codec := NewJSONCodec()

	method := "Int.Sum"
	arg := "arg"
	req := codec.NewRequest(method, arg)
	b, _ := json.Marshal(req)

	requests, err := codec.ReadRequest(b)
	assert.Nil(t, err)
	assert.Equal(t, []xrpc.Request{req}, requests)

	b, _ = json.Marshal("")
	_, err = codec.ReadRequest(b)
	assert.NotNil(t, err)

	b, _ = json.Marshal([]xrpc.Request{req})

	requests, err = codec.ReadRequest(b)
	assert.Nil(t, err)
	assert.Equal(t, []xrpc.Request{req}, requests)

}

func TestJsonCodec_ReadRequestBody(t *testing.T) {
	codec := NewJSONCodec()

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

	arg2 := []Args{
		{
			A: 1,
			B: 2,
		},
	}
	req = codec.NewRequest(method, arg2)
	err = codec.ReadRequestBody(req.GetParams(), out)
	assert.Nil(t, err)
	assert.Equal(t, &arg2[0], out)
}

func TestJsonCodec_ReadResponseBody(t *testing.T) {
	codec := NewJSONCodec()

	data := "1"
	resp := codec.NewResponse(data)

	out := new(string)
	err := codec.ReadResponseBody(resp.GetReply(), out)
	assert.Nil(t, err)
	assert.Equal(t, &data, out)
}

func TestJsonCodec_EncodeRequests(t *testing.T) {
	codec := NewJSONCodec()
	data := "1"

	b, _ := json.Marshal(data)
	res, err := codec.EncodeRequests(data)
	assert.Nil(t, err)
	assert.Equal(t, b, res)
}

func TestJsonCodec_EncodeResponses(t *testing.T) {
	codec := NewJSONCodec()
	data := "1"

	b, _ := json.Marshal(data)
	res, err := codec.EncodeResponses(data)
	assert.Nil(t, err)
	assert.Equal(t, b, res)
}

func TestJsonCodec_ErrResponse(t *testing.T) {
	codec := NewJSONCodec()

	errCode := 1
	err := errors.New("1")
	resp := codec.ErrResponse(errCode, err)

	assert.Equal(t, &xrpc.Error{
		ErrCode: errCode,
		ErrMsg:  err.Error(),
	}, resp.Error())
	assert.Equal(t, errCode, resp.GetErrCode())
}

func TestJsonCodec_Send(t *testing.T) {

}
