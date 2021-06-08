package geecache

import (
	"fmt"
	"geecache/geecache/consistenthash"
	pb "geecache/geecache/geecachepb"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"google.golang.org/protobuf/proto"
)

const defaultBasePath = "/_geecache/"
const defaultReplicas = 50

type HTTPPool struct{
	self string 
	basePath string 
	mu sync.Mutex
	peers *consistenthash.Map
	httpGetter map[string]*HTTPGetter
}

type HTTPGetter struct {
	baseURL string
}

// func (h *HTTPGetter)Get(group string, key string)  ([]byte, error) {

func (h *HTTPGetter)Get(in *pb.Request, out *pb.Response) error {
	u := fmt.Sprintf("%v%v/%v", h.baseURL, url.QueryEscape(in.GetGroup()), url.QueryEscape(in.GetKey()))
	res, err := http.Get(u)
	if err != nil {
		return  err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("server return %v", res.StatusCode)
	}

		bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("reading response body error %v", err)
	}


	if err = proto.Unmarshal(bytes,out); err != nil {
		return fmt.Errorf("decoding response body:%v", err)
	}

	return nil

	// defer res.Body.Close()
	// if res.StatusCode != http.StatusOK {
	// 	return nil, fmt.Errorf("Server return: %v", res.Status)
	// }

	// bytes, err := ioutil.ReadAll(res.Body)
	// if err != nil {
	// 	return nil ,fmt.Errorf("reading response body error %v", err)
	// }

	// return bytes, nil
}

var _ PeerGetter = (*HTTPGetter)(nil)

func NewHttpPool(self string) *HTTPPool{
	return &HTTPPool{
		self: self,
		basePath: defaultBasePath,
	}
}



func (h *HTTPPool) Log(format string,v ...interface{}){
	log.Printf("[Server %s] %s", h.self, fmt.Sprintf(format, v...))
}

func (h *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("url path: %s, basePath %s",r.URL.Path,h.basePath)
	if !strings.HasPrefix(r.URL.Path, h.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}

	h.Log("%s %s", r.Method, r.URL.Path)
	parts := strings.SplitN(r.URL.Path[len(h.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	groupName := parts[0]
	key := parts[1]
	group := GetGroup(groupName)
	if nil == group {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}

	view, err := group.Get(key)
	if nil != err {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return 
	}

	body, err := proto.Marshal(&pb.Response{Value:view.ByteSlice()})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return 
	}


	w.Header().Set("Content-Type","application/octet-stream")
	w.Write(body)
}


func (p *HTTPPool) Set(peers ...string){
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers = consistenthash.New(defaultReplicas, nil)
	p.peers.Add(peers...)
	p.httpGetter = make(map[string]*HTTPGetter, len(peers))
	for _,peer := range peers {
		p.httpGetter[peer] = &HTTPGetter{
			baseURL: peer + p.basePath,
		}
	}
}

func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("Pick peer %s", peer)
		return p.httpGetter[peer], true
	}

	return nil, false
}


// 利用强制类型转换，确保HTTPPool 实现了接口PeerPicker
// 如果没有实现，在编译期间报错
var _ PeerPicker = (*HTTPPool)(nil) 
