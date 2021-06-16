# `RPC`
## 服务端与消息编码
```go
package codec

import (
	"io"
)

type Header struct {
	ServiceMethod string // format service method
	Seq uint64 // sequence number chosen by client
	Error string
}

type Codec interface {
	io.Closer
	ReadHeader(*Header) error
	ReadBody(interface{}) error

	Write(*Header,interface{}) error
}

type NewCodecFunc func(io.ReadWriteCloser) Codec

type Type string

const (
	GobType Type = "application/gob"
	JsonType Type = "application/json"
)

var NewCodecFuncMap map[Type]NewCodecFunc

func init(){
	NewCodecFuncMap = make(map[Type]NewCodecFunc)
	NewCodecFuncMap[GobType] = NewGobCodec
}
```
编码：
```go
package codec

import (
	"bufio"
	"encoding/gob"
	"io"
	"log"
)

type GobCodec struct {
	conn io.ReadWriteCloser
	buf *bufio.Writer
	dec *gob.Decoder
	enc *gob.Encoder
}

var _ Codec = (*GobCodec)(nil)

func NewGobCodec(conn io.ReadWriteCloser) Codec {
	buf := bufio.NewWriter(conn)

	return &GobCodec{
		conn: conn,
		buf: buf,
		dec: gob.NewDecoder(conn),
		enc: gob.NewEncoder(buf),
	}
}

func (c *GobCodec)ReadHeader(h *Header) error {
	return c.dec.Decode(h)
}

func (c *GobCodec)ReadBody(body interface{}) error {
	return c.dec.Decode(body)
}

func (c *GobCodec)Write(h *Header, body interface{}) (err error) {
	defer func() {
		_ = c.buf.Flush()
		if err != nil {
			_ = c.Close()
		}
	}()

	if err := c.enc.Encode(h); err != nil {
		log.Printf("rpc codec : gob error encoding header: %v", err)
		return err
	}

	if err = c.enc.Encode(body);err != nil {
		log.Printf("rpc codec : gob error encoding body : %v", err)
		return err
	}

	return nil
}

func (c *GobCodec)Close() error {
	return c.conn.Close()
}
```
### `client`
```go
package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"geerpc/geerpc/codec"
	"io"
	"log"
	"net"
	"sync"
)

type Call struct {
	Seq uint64
	ServiceMothed string
	Args interface{}
	Reply interface{}
	Error error
	Done chan *Call
}

func (call *Call) done() {
	call.Done <- call
}

type Client struct {
	cc codec.Codec
	opt *codec.Option
	sending 		 sync.Mutex
	header codec.Header
	mu sync.Mutex
	seq uint64
	pending map[uint64]*Call
	closing bool
	shutdown bool
}

var _ io.Closer = (*Client)(nil)

var ErrShutdown = errors.New("connection is shut down")

func (client *Client) Close() error {
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.shutdown {
		return ErrShutdown
	}

	client.closing = true
	return client.cc.Close()
}

func (client *Client)IsAvaiable() bool {
	client.mu.Lock()
	defer client.mu.Unlock()

	return !client.shutdown && !client.closing
}

func (client *Client) registerCall(call *Call) (uint64, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.closing || client.shutdown {
		return 0, ErrShutdown
	}

	call.Seq = client.seq
	client.pending[call.Seq] = call
	client.seq++

	return call.Seq, nil
}


func (client *Client) removeCall(seq uint64) *Call {
	client.mu.Lock()
	defer client.mu.Unlock()
	call := client.pending[seq]
	delete(client.pending, seq)

	return call
}

func (client *Client) terminateCalls(err error) {
	client.sending.Lock()
	defer client.sending.Unlock()
	client.mu.Lock()
	defer client.mu.Unlock()
	client.shutdown = false
	for _, call := range client.pending {
		call.Error = err
		call.done()
	}
}

func (client *Client)receive(){
	var err error

	for err == nil {
		var h codec.Header

		if err = client.cc.ReadHeader(&h); err != nil {
			break
		}

		call := client.removeCall(h.Seq)

		switch {
			case call == nil :
				err = client.cc.ReadBody(nil)
			case h.Error != "":
				call.Error = fmt.Errorf(h.Error)
				err = client.cc.ReadBody(nil)
				call.done()
			default:
				err = client.cc.ReadBody(call.Reply)
				if err != nil {
					call.Error = errors.New("read body error " + err.Error())
				}
				call.done()
		}

	}

	client.terminateCalls(err)
}

func NewClient(conn net.Conn, opt *codec.Option) (*Client, error){
	f := codec.NewCodecFuncMap[opt.CodeType]
	if f == nil {
		err := fmt.Errorf("invalid codec type %s", opt.CodeType)
		log.Println("rpc client:codec error ", err)
		return nil, err
	}

	if err := json.NewEncoder(conn).Encode(opt); err != nil {
		log.Println("rpc client: option error ", err)
		_ = conn.Close()
		return nil, err
	}

	return newClientCodec(f(conn), opt), nil
}

func newClientCodec(cc codec.Codec, opt *codec.Option) *Client {
	client := &Client{
		seq : 1,
		cc: cc,
		opt: opt,
		pending: make(map[uint64]*Call),
	}

	return client
}

func parseOption(opts ...*codec.Option) (*codec.Option, error){
	if len(opts) == 0 || opts[0] == nil {
		return codec.DefaultOption, nil
	}

	if len(opts) != 1{
		return nil, errors.New("number of options is more than 1")
	}

	opt := opts[0]
	opt.MagicNumber = codec.DefaultOption.MagicNumber

	if opt.CodeType == "" {
		opt.CodeType = codec.DefaultOption.CodeType
	}

	return opt, nil
}

func Dial(network, address string, opts ...*codec.Option)(client *Client, err error) {
	opt, err := parseOption(opts...)
	if err != nil {
		return nil, err
	}

	conn,err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}

	defer func() {
		if client == nil {
			_ = conn.Close()
		}
	}()

	return NewClient(conn,opt)
}

func (client *Client) send(call *Call) {
	client.sending.Lock()
	defer client.sending.Unlock()
	seq, err := client.registerCall(call)
	if err != nil {
		call.Error = err
		call.done()
		return 
	}

	client.header.ServiceMethod = call.ServiceMothed
	client.header.Seq = call.Seq
	client.header.Error = ""

	if err := client.cc.Write(&client.header, call.Args); err != nil {
		call = client.removeCall(seq)
		if err != nil {
			call.Error = err
			call.done()
		}
	}
}

func (client *Client)Go(serviceMethod string, args, reply interface{}, done chan *Call) *Call {
	if done == nil {
		done = make(chan *Call, 10)
	} else if cap(done) == 0 {
		log.Println("rpc client: done channel is unbuffered")
	} 

	call := &Call{
		ServiceMothed: serviceMethod,
		Args: args,
		Reply: reply,
		Done: done,
	}

	client.send(call)

	return call
}

func (client *Client)Call(serviceMethod string, args, reply interface{}) error {	
	call := <- client.Go(serviceMethod, args, reply, make(chan *Call,1)).Done

	return call.Error
}
```


