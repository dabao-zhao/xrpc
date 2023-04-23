package xrpc

import (
	"log"
	"reflect"
	"unicode"
	"unicode/utf8"
)

type methodType struct {
	method    reflect.Method
	ArgType   reflect.Type
	ReplyType reflect.Type
}

type service struct {
	name   string
	val    reflect.Value
	typ    reflect.Type
	method map[string]*methodType
}

func (s *service) call(mType *methodType, arg, reply reflect.Value) error {
	function := mType.method.Func
	returnValues := function.Call([]reflect.Value{s.val, arg, reply})
	if i := returnValues[0].Interface(); i != nil {
		return i.(error)
	}
	return nil
}

func isExported(name string) bool {
	r, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(r)
}

func isExportedOrBuiltinType(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return isExported(t.Name()) || t.PkgPath() == ""
}

func suitableMethods(typ reflect.Type) map[string]*methodType {
	methods := make(map[string]*methodType)
	for m := 0; m < typ.NumMethod(); m++ {
		method := typ.Method(m)
		if mt := suitableMethod(method); mt != nil {
			methods[method.Name] = mt
		}
	}
	return methods
}

func suitableMethodWithName(typ reflect.Type, methodName string) *methodType {
	if method, ex := typ.MethodByName(methodName); ex {
		return suitableMethod(method)
	}
	log.Println("rpc.RegisterName: has no such method")
	return nil
}

func suitableMethod(method reflect.Method) *methodType {
	mType := method.Type
	mName := method.Name

	// Method must be exported.
	if method.PkgPath != "" {
		return nil
	}
	// Method needs three ins: receiver, *args, *reply.
	if mType.NumIn() != 3 {
		log.Printf("rpc.Register: method %q has %d input parameters; needs exactly three\n", mName, mType.NumIn())
		return nil
	}
	// First arg need not be a pointer.
	argType := mType.In(1)
	if !isExportedOrBuiltinType(argType) {
		log.Printf("rpc.Register: argument type of method %q is not exported: %q\n", mName, argType)
		return nil
	}
	// Second arg must be a pointer.
	replyType := mType.In(2)
	if replyType.Kind() != reflect.Ptr {
		log.Printf("rpc.Register: reply type of method %q is not a pointer: %q\n", mName, replyType)
		return nil
	}
	// Reply type must be exported.
	if !isExportedOrBuiltinType(replyType) {
		log.Printf("rpc.Register: reply type of method %q is not exported: %q\n", mName, replyType)
		return nil
	}
	// Method needs one out.
	if mType.NumOut() != 1 {
		log.Printf("rpc.Register: method %q has %d output parameters; needs exactly one\n", mName, mType.NumOut())
		return nil
	}
	// The return type of the method must be error.
	if returnType := mType.Out(0); returnType != reflect.TypeOf((*error)(nil)).Elem() {
		log.Printf("rpc.Register: return type of method %q is %q, must be error\n", mName, returnType)
		return nil
	}
	return &methodType{method: method, ArgType: argType, ReplyType: replyType}
}
