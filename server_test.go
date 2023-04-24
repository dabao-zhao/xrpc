package xrpc

import (
	"reflect"
	"testing"
)

type Args struct {
	A int
	B int
}

type Int struct{}

func (i *Int) Sum(args *Args, reply *int) error {
	*reply = args.A + args.B
	return nil
}

func TestServer_call(t *testing.T) {
	codec := NewGobCodec().(*gobCodec)
	s := NewServerWithCodec(codec)
	_ = s.Register(new(Int))

	argv, _ := codec.Encode(&Args{A: 222, B: 333})

	r := new(int)
	*r = 555
	reply, _ := codec.Encode(r)

	type args struct {
		req Request
	}
	tests := []struct {
		name string
		args args
		want []Response
	}{
		{
			name: "case 0",
			args: args{
				req: &defaultRequest{Method: "Int.Sum", Args: argv},
			},
			want: []Response{
				&defaultResponse{Err: "", Reply: reply},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := s.call([]Request{tt.args.req}); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Server.call() = %v, want %v", got, tt.want)
			}
		})
	}
}
