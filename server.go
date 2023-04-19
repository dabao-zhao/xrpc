package xrpc

import (
	"bufio"
	"encoding/json"
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
		errMsg := "rpc.Register: no service name for type " + srv.typ.String()
		log.Printf(errMsg)
		return errors.New(errMsg)
	}

	if !isExported(sName) {
		errMsg := "rpc.Register: type " + sName + " is not exported"
		log.Printf(errMsg)
		return errors.New(errMsg)
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
			errMsg := "rpc.Register: no service name for type " + srv.typ.String()
			log.Printf(errMsg)
			return errors.New(errMsg)
		}

		if !isExported(sName) {
			errMsg := "rpc.Register: type " + sName + " is not exported"
			log.Printf(errMsg)
			return errors.New(errMsg)
		}
		srv.name = sName
		srv.method = make(map[string]*methodType)
		srv.method[mt.method.Name] = mt
		s.m.Store(sName, srv)
	}
	return nil
}

func (s *Server) call(reqs []Request) (replies []Response) {
	defer func() { log.Printf("server called end") }()
	replies = make([]Response, len(reqs))
	for idx, req := range reqs {
		var (
			reply Response
		)

		serviceName, methodName, err := parseFromRPCMethod(req.GetMethod())
		if err != nil {
			log.Printf("parseFromRPCMethod err=%v", err)
			reply = s.codec.ErrResponse(InvalidRequest, err)
			replies[idx] = reply
			continue
		}

		// method existed or not
		svcI, ok := s.m.Load(serviceName)
		if !ok {
			err := errors.New("rpc: can't find service " + serviceName)
			reply = s.codec.ErrResponse(MethodNotFound, err)
			replies[idx] = reply
			continue
		}

		svc := svcI.(*service)
		mType := svc.method[methodName]
		if mType == nil {
			err := errors.New("rpc: can't find method " + req.GetMethod())
			reply = s.codec.ErrResponse(MethodNotFound, err)
			reply.SetReqId(req.GetId())
			replies[idx] = reply
			continue
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
			log.Printf("could not readRequestBody err=%v", err)
			err := errors.New("rpc: could not read request body " + req.GetMethod())
			reply = s.codec.ErrResponse(InternalErr, err)
			replies[idx] = reply
			continue
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
			// normal response
			reply = s.codec.NewResponse(replyV.Interface())
			reply.SetReqId(req.GetId())
		}

		replies[idx] = reply
	}

	return
}

func (s *Server) serveConn(conn net.Conn) {
	rr := bufio.NewReader(conn)
	wr := bufio.NewWriter(conn)

	var (
		pRecv = proto.New()
		pSend = proto.New()
		resps = make([]Response, 0)
	)

	for {
		if err := pRecv.ReadTCP(rr); err != nil {
			log.Printf("ReadTCP error: %v", err)
			break
		}

		log.Printf("recv a new request: %v", string(pRecv.Body))
		reqs, err := s.codec.ReadRequest(pRecv.Body)
		if err != nil {
			log.Printf("could not parse request: %v", err)
			resps = append(resps, s.codec.ErrResponse(ParseErr, err))
			goto wr
		}
		resps = s.call(reqs)
		log.Printf("s.call(req) req: %v result: %v", reqs, resps)
	wr:
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
		err  error
	)
	switch req.Method {
	case http.MethodPost:
		if data, err = io.ReadAll(req.Body); err != nil {
			resp := s.codec.ErrResponse(InvalidParamErr, err)
			b, err := s.codec.EncodeResponses(resp)
			log.Printf("s.codec.EncodeResponses err=%v", err)
			_ = String(w, http.StatusOK, b)
			return
		}
		defer req.Body.Close()
	default:
		err := errors.New("method not allowed: " + req.Method)
		resp := s.codec.ErrResponse(MethodNotFound, err)
		b, err := s.codec.EncodeResponses(resp)
		log.Printf("s.codec.EncodeResponses err=%v", err)
		_ = String(w, http.StatusOK, b)
		return
	}

	log.Printf("[HTTP] got request data: %v", string(data))
	rpcReqs, err := s.codec.ReadRequest(data)
	if err != nil {
		resp := s.codec.ErrResponse(ParseErr, err)
		b, err := s.codec.EncodeResponses(resp)
		log.Printf("s.codec.EncodeResponses err=%v", err)
		_ = String(w, http.StatusOK, b)
		return
	}

	resps := s.call(rpcReqs)
	log.Printf("s.call(rpcReq) result: %v", resps)
	b, err := s.codec.EncodeResponses(resps)
	log.Printf("s.codec.EncodeResponses err=%v", err)
	_ = String(w, http.StatusOK, b)
	return
}

func JSON(w http.ResponseWriter, statusCode int, v interface{}) error {
	w.WriteHeader(statusCode)
	w.Header().Set("Content-Type", "application/json")
	b, err := json.Marshal(v)
	if err != nil {
		log.Printf("could not marshal v=%v, err=%v", v, err)
		return err
	}

	_, err = io.WriteString(w, string(b))
	return err
}

func String(w http.ResponseWriter, statusCode int, b []byte) error {
	w.WriteHeader(statusCode)
	w.Header().Set("Content-Type", "text/plain")
	_, err := io.WriteString(w, string(b))
	return err
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

	return serviceName, methodName, nil
}
