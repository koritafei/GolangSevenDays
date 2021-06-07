package main

import (
	"flag"
	"fmt"
	"geecache/geecache"
	"log"
	"net/http"
)

var db = map[string]string{
	"Tom":"630",
	"Jack":"587",
	"Sam":"567",
}

func createGroups()*geecache.Group{
	return geecache.NewGroup("scores",2 << 10, geecache.GetFunc( func(key string)([]byte, error){
		log.Println("[SlowDB] search key", key)
		if v,ok := db[key]; ok {
			return []byte(v), nil
		}
		return nil, fmt.Errorf("%s not exists", key)
	}))
}

func startCacheServer(addr string, addrs []string, gee *geecache.Group){
	peers := geecache.NewHttpPool(addr)
	peers.Set(addrs...)
	gee.RegisterPeers(peers)
	log.Printf("geecache is run at %s\n", addr)
	log.Fatal(http.ListenAndServe(addr[7:], peers))
}

func startAPIServer(apiaddr string, gee *geecache.Group) {
	http.Handle("/api", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		view, err := gee.Get(key)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return 
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(view.ByteSlice())
	}))

	log.Println("fontend server is run at %s", apiaddr)
	log.Fatal(http.ListenAndServe(apiaddr[7:], nil))
}


func main() {
	fmt.Printf("Hello, Cache\n")
	var port int
	var api bool

	flag.IntVar(&port, "port", 8001, "Geecache server port")
	flag.BoolVar(&api, "api", false, "Start a api server ?")
	flag.Parse()

	apiAddr := "http://localhost:9999"
	addrMap := map[int]string{
		8001: "http://localhost:8001",
		8002: "http://localhost:8002",
		8003: "http://localhost:8003",
	}

	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}

	gee := createGroups()
	if api {
		go startAPIServer(apiAddr, gee)
	}
	startCacheServer(addrMap[port], []string(addrs), gee)
}
