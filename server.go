package xrpc

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"reflect"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/dabao-zhao/xrpc/proto"
)

type Server struct {
	m     sync.Map    // map[string]*service
	codec ServerCodec // codec to read request and writeResponse
}

func NewServerWithCodec(codec ServerCodec) *Server {
	if codec == nil {
		codec = NewGobCodec()
	}
	return &Server{codec: codec}
}

func (s *Server) Register(data interface{}) error {
	srv := new(service)
	srv.typ = reflect.TypeOf(data)
	srv.val = reflect.ValueOf(data)
	sName := reflect.Indirect(srv.val).Type().Name()

	if sName == "" {
		return errors.New("rpc.Register: no service name for type " + srv.typ.String())
	}

	if !isExported(sName) {
		return errors.New("rpc.Register: type " + sName + " is not exported")
	}
	srv.name = sName
	srv.method = suitableMethods(srv.typ)

	if _, dup := s.m.LoadOrStore(sName, srv); dup {
		return errors.New("rpc: service already defined: " + sName)
	}
	return nil
}

func (s *Server) RegisterName(data interface{}, methodName string) error {
	srv := new(service)
	srv.typ = reflect.TypeOf(data)
	srv.val = reflect.ValueOf(data)
	sName := reflect.Indirect(srv.val).Type().Name()

	mt := suitableMethodWithName(srv.typ, methodName)

	i, ex := s.m.Load(sName)
	if ex {
		loadedSrv := i.(*service)
		loadedSrv.method[mt.method.Name] = mt
		s.m.Store(sName, loadedSrv)
	} else {
		if sName == "" {
			return errors.New("rpc.Register: no service name for type " + srv.typ.String())
		}

		if !isExported(sName) {
			return errors.New("rpc.Register: type " + sName + " is not exported")
		}
		srv.name = sName
		srv.method = make(map[string]*methodType)
		srv.method[mt.method.Name] = mt
		s.m.Store(sName, srv)
	}
	return nil
}

func (s *Server) call(reqs []Request) (replies []Response) {
	replies = make([]Response, len(reqs))
	wg := sync.WaitGroup{}
	wg.Add(len(reqs))
	for idx, req := range reqs {
		go func(req Request, idx int) {
			defer wg.Done()
			replies[idx] = s.handleRequest(req)
		}(req, idx)
	}
	wg.Wait()
	return
}

func (s *Server) serveConn(conn net.Conn) {
	rr := bufio.NewReader(conn)
	wr := bufio.NewWriter(conn)

	var (
		pRec  = proto.New()
		pSend = proto.New()
		resps = make([]Response, 0)
	)

	for {
		if err := pRec.ReadTCP(rr); err != nil {
			log.Printf("ReadTCP error: %v", err)
			break
		}
		reqs, err := s.codec.ReadRequest(pRec.Body)
		if err != nil {
			resps = append(resps, s.codec.ErrResponse(ParseErr, err))
			if pSend.Body, err = s.codec.EncodeResponses(resps); err != nil {
				log.Printf("could not encode responses, err=%v", err)
				continue
			}
			_ = pSend.WriteTCP(wr)
			_ = wr.Flush()
			continue
		}
		resps = s.call(reqs)
		if pSend.Body, err = s.codec.EncodeResponses(resps); err != nil {
			log.Printf("could not encode responses, err=%v", err)
			continue
		}
		_ = pSend.WriteTCP(wr)
		_ = wr.Flush()
	}
}

func (s *Server) ServeTCP(addr string) {
	log.Printf("RPC server over TCP is listening: %s", addr)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("listener.Accept(), err=%v", err)
			continue
		}

		go s.serveConn(conn)
	}
}

func (s *Server) ListenAndServe(addr string) {
	log.Printf("RPC server over HTTP is listening: %s", addr)
	if err := http.ListenAndServe(
		addr,
		http.TimeoutHandler(s, 5*time.Second, "timeout"),
	); err != nil {
		panic(err)
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer func() {
		if err, ok := recover().(error); ok && err != nil {
			log.Printf("[ServeHTTP] recover %v with stack: \n", err)
			debug.PrintStack()
		}
	}()

	var (
		data []byte
		b    []byte
		err  error
	)

	if req.Method != http.MethodPost {
		err := errors.New("method not allowed: " + req.Method)
		resp := s.codec.ErrResponse(MethodNotFound, err)
		b, _ := s.codec.EncodeResponses(resp)
		_ = s.codec.Send(w, http.StatusOK, b)
		return
	}

	if data, err = io.ReadAll(req.Body); err != nil {
		resp := s.codec.ErrResponse(InvalidParamErr, err)
		b, _ := s.codec.EncodeResponses(resp)
		_ = s.codec.Send(w, http.StatusOK, b)
		return
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(req.Body)

	rpcReqs, err := s.codec.ReadRequest(data)
	if err != nil {
		resp := s.codec.ErrResponse(ParseErr, err)
		b, _ := s.codec.EncodeResponses(resp)
		_ = s.codec.Send(w, http.StatusOK, b)
		return
	}

	resps := s.call(rpcReqs)
	if len(resps) == 1 {
		b, _ = s.codec.EncodeResponses(resps[0])
	} else {
		b, _ = s.codec.EncodeResponses(resps)
	}
	_ = s.codec.Send(w, http.StatusOK, b)
	return
}

func (s *Server) handleRequest(req Request) Response {
	var (
		reply Response
	)
	serviceName, methodName, err := parseFromRPCMethod(req.GetMethod())
	if err != nil {
		reply = s.codec.ErrResponse(InvalidRequest, err)
		reply.SetReqId(req.GetId())
		return reply
	}

	svcI, ok := s.m.Load(serviceName)
	if !ok {
		reply = s.codec.ErrResponse(MethodNotFound, errors.New("rpc: can't find service "+serviceName))
		reply.SetReqId(req.GetId())
		return reply
	}

	svc := svcI.(*service)
	mType := svc.method[methodName]
	if mType == nil {
		reply = s.codec.ErrResponse(MethodNotFound, errors.New("rpc: can't find method "+req.GetMethod()))
		reply.SetReqId(req.GetId())
		return reply
	}

	var (
		argV       reflect.Value
		argIsValue = false
	)
	if mType.ArgType.Kind() == reflect.Ptr {
		argV = reflect.New(mType.ArgType.Elem())
	} else {
		argV = reflect.New(mType.ArgType)
		argIsValue = true
	}
	if argIsValue {
		argV = argV.Elem() // argV guaranteed to be a pointer now.
	}

	if err := s.codec.ReadRequestBody(req.GetParams(), argV.Interface()); err != nil {
		reply = s.codec.ErrResponse(InternalErr, errors.New("rpc: could not read request body "+req.GetMethod()))
		reply.SetReqId(req.GetId())
		return reply
	}

	var replyV reflect.Value
	replyV = reflect.New(mType.ReplyType.Elem())
	switch mType.ReplyType.Elem().Kind() {
	case reflect.Map:
		replyV.Elem().Set(reflect.MakeMap(mType.ReplyType.Elem()))
	case reflect.Slice:
		replyV.Elem().Set(reflect.MakeSlice(mType.ReplyType.Elem(), 0, 0))
	}

	if err := svc.call(mType, argV, replyV); err != nil {
		reply = s.codec.ErrResponse(InternalErr, err)
		reply.SetReqId(req.GetId())
	} else {
		reply = s.codec.NewResponse(replyV.Interface())
		reply.SetReqId(req.GetId())
	}

	return reply
}

func parseFromRPCMethod(reqMethod string) (serviceName, methodName string, err error) {
	if strings.Count(reqMethod, ".") != 1 {
		return "", "", fmt.Errorf("rpc: service/method request ill-formed: %s", reqMethod)
	}
	dot := strings.LastIndex(reqMethod, ".")
	if dot < 0 {
		return "", "", fmt.Errorf("rpc: service/method request ill-formed: %s", reqMethod)
	}

	serviceName = reqMethod[:dot]
	methodName = reqMethod[dot+1:]

	return
}
