package main

import (
	"geerpc/geerpc/client"
	"geerpc/geerpc/server"
	geerpc "geerpc/geerpc/server"
	"log"
	"net"
	"sync"
	"time"
)

type Foo int

type Args struct {
	Nums1, Nums2 int
}

func (f Foo) Sum(args Args, reply *int) error {
	*reply = args.Nums1 + args.Nums2
	return nil
}

func startServer_serice(addr chan string) {
	var foo Foo
	if err := server.Register(&foo); err != nil {
		log.Fatal("regiter error: ", err)
	}
	l, err := net.Listen("tcp",":0")
	if err != nil {
		log.Fatal("network error ", err)
	}

	log.Println("rpc server start on ", l.Addr())
	addr <- l.Addr().String()
	geerpc.Accept(l)
}

func main() {
	log.SetFlags(0)
	addr := make(chan string)
	go startServer_serice(addr)
	client, _ := client.Dial("tcp", <-addr)
	defer func(){
		_ = client.Close()
	}()

	time.Sleep(time.Second)
	var wg sync.WaitGroup
	for i:=0; i < 5;i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			args := &Args{
				Nums1:i,
				Nums2: i * i,
			}
			var reply int
			if err := client.Call("Foo.Sum", args, &reply); err != nil {
					log.Fatal("call Foo.Sum failed ", err.Error())
				}

				log.Printf("%d + %d = %d\n", args.Nums1,args.Nums2, reply)
			}(i)
		}
		wg.Wait()
}