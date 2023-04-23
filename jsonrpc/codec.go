package jsonrpc

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"math/rand"
	"net/http"
	"reflect"
	"time"

	"github.com/dabao-zhao/xrpc"
)

var (
	_ xrpc.Request  = &jsonRequest{}
	_ xrpc.Response = &jsonResponse{}
	_ xrpc.Codec    = &jsonCodec{}
)

const (
	version    = "2.0"
	baseStr    = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	baseStrLen = 62
	lenReqID   = 8
)

type jsonRequest struct {
	ID      string      `json:"id"`
	Method  string      `json:"method"`
	Args    interface{} `json:"params"`
	Version string      `json:"jsonrpc"`
}

func (j *jsonRequest) GetId() string     { return j.ID }
func (j *jsonRequest) GetMethod() string { return j.Method }
func (j *jsonRequest) GetParams() []byte {
	b, err := json.Marshal(j.Args)
	if err != nil {
		panic(err)
	}
	return b
}

type jsonResponse struct {
	ID      string      `json:"id"`
	Err     *xrpc.Error `json:"error,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Version string      `json:"jsonrpc"`
}

func (j *jsonResponse) SetReqId(id string) { j.ID = id }
func (j *jsonResponse) Error() error       { return j.Err }
func (j *jsonResponse) GetReply() []byte {
	b, err := json.Marshal(j.Result)
	if err != nil {
		panic(err)
	}
	return b
}
func (j *jsonResponse) GetResult() interface{} {
	return j.Result
}
func (j *jsonResponse) GetErrCode() int {
	if j.Err == nil {
		return xrpc.Success
	}
	return j.Err.ErrCode
}

type jsonCodec struct {
}

func NewJSONCodec() xrpc.Codec {
	return &jsonCodec{}
}

func (j *jsonCodec) encode(argv interface{}) ([]byte, error) {
	return json.Marshal(argv)
}

func (j *jsonCodec) decode(data []byte, out interface{}) error {
	dec := json.NewDecoder(bytes.NewBuffer(data))
	dec.DisallowUnknownFields()
	return dec.Decode(out)
}

func (j *jsonCodec) NewResponse(reply interface{}) xrpc.Response {
	resp := &jsonResponse{
		Version: version,
		ID:      "",
		Result:  reply,
	}

	return resp
}

func (j *jsonCodec) ErrResponse(errCode int, err error) xrpc.Response {
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}

	return &jsonResponse{
		Err: &xrpc.Error{
			ErrCode: errCode,
			ErrMsg:  errMsg,
		},
		Version: version,
	}
}

func (j *jsonCodec) NewRequest(method string, argv interface{}) xrpc.Request {
	req := &jsonRequest{
		ID:      randId(),
		Method:  method,
		Args:    argv,
		Version: version,
	}

	return req
}

func (j *jsonCodec) ReadResponse(data []byte) (resps []xrpc.Response, err error) {
	jsonResps := make([]*jsonResponse, 0)
	if err = j.decode(data, &jsonResps); err != nil {
		resp := new(jsonResponse)
		if err = j.decode(data, resp); err != nil {
			return nil, err
		}
		resps = append(resps, resp)
		return resps, nil
	}

	for _, jsonResp := range jsonResps {
		resps = append(resps, jsonResp)
	}

	return resps, nil
}

func (j *jsonCodec) ReadRequest(data []byte) (reqs []xrpc.Request, err error) {
	jsonReqs := make([]*jsonRequest, 0)
	if err = j.decode(data, &jsonReqs); err != nil {
		req := new(jsonRequest)
		if err = j.decode(data, req); err != nil {
			return nil, err
		}
		reqs = append(reqs, req)
		return reqs, nil
	}

	for _, jsonReq := range jsonReqs {
		reqs = append(reqs, jsonReq)
	}

	return reqs, nil
}

func (j *jsonCodec) ReadRequestBody(data []byte, out interface{}) error {
	var v interface{}
	err := json.Unmarshal(data, &v)
	if err != nil {
		return err
	}
	typeOfV := reflect.TypeOf(v)
	typeOfO := reflect.TypeOf(out)
	if typeOfV.Kind() == reflect.Ptr {
		typeOfV = typeOfV.Elem()
	}
	// 理论上 out 一直是指针
	if typeOfO.Kind() == reflect.Ptr {
		typeOfO = typeOfO.Elem()
	}
	if typeOfV.Kind() == reflect.Slice && typeOfO.Kind() != reflect.Slice {
		args := v.([]interface{})
		data, err = json.Marshal(args[0])
		if err != nil {
			return err
		}
	}

	return j.decode(data, out)
}

func (j *jsonCodec) ReadResponseBody(data []byte, out interface{}) error {
	return j.decode(data, out)
}

func (j *jsonCodec) EncodeRequests(v interface{}) ([]byte, error) {
	return j.encode(v)
}

func (j *jsonCodec) EncodeResponses(v interface{}) ([]byte, error) {
	return j.encode(v)
}

func (g *jsonCodec) Send(w http.ResponseWriter, statusCode int, b []byte) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, err := w.Write(b)
	return err
}

func randId() string {
	bs := []byte(baseStr)
	result := make([]byte, 0, lenReqID)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < lenReqID; i++ {
		result = append(result, bs[r.Intn(baseStrLen)])
	}
	m := md5.New()
	m.Write(result)
	return hex.EncodeToString(m.Sum(nil))
}
