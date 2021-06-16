package main

import (
	"fmt"
	"geerpc/geerpc/client"
	geerpc "geerpc/geerpc/server"
	"log"
	"net"
	"sync"
	"time"
)


func startserver(addr chan string) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal("network error: ", err)
	}

	log.Println("start rpc server on addr: ", l.Addr())
	addr <- l.Addr().String()

	geerpc.Accept(l)
}


func main() {
	log.SetFlags(0)
	addr := make(chan string)

	go startserver(addr)
	client, _ := client.Dial("tcp", <-addr)
	defer func() {
		_ = client.Close()
	}()

	time.Sleep(time.Second)
	var wg sync.WaitGroup
	for i:=0; i < 5;i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			args := fmt.Sprintf("geerpc req %d", i)
			var reply string
			if err := client.Call("foo.sum",args, &reply); err != nil {
				log.Fatal("call foo.sum failed ", err)
			}
			log.Println("reply ", reply)
		}(i)
	}

	wg.Wait()
}