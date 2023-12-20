package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

type Server interface {
	Address() string
	IsAlive() bool
	Serve(rw http.ResponseWriter, r http.Request)
}

type simpleserver struct {
	addr  string
	proxy *httputil.ReverseProxy
}

func newsimpleserver(addr string) *simpleserver {
	serverUrl, err := url.Parse(addr)
	handleErr(err)

	return &simpleserver{
		addr:  addr,
		proxy: httputil.NewSingleHostReverseProxy(serverUrl),
	}
}

type Loadbalancer struct {
	port            string
	roundRobincount int
	servers         []Server
}

func NewLoadBalancer(port string, servers []Server) *Loadbalancer {
	return &Loadbalancer{
		port:            port,
		roundRobincount: 0,
		servers:         servers,
	}
}

func handleErr(err error) {
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
}

func (s *simpleserver) Address() string { return s.addr }

func (s *simpleserver) IsAlive() bool { return true }

func (s *simpleserver) Serve(rw http.ResponseWriter, req http.Request) {
	s.proxy.ServeHTTP(rw, &req)
}

func (lb *Loadbalancer) getNextAvailabeServer() Server {
	server := lb.servers[lb.roundRobincount%len(lb.servers)]
	for !server.IsAlive() {
		lb.roundRobincount++
		server = lb.servers[lb.roundRobincount%len(lb.servers)]
	}
	lb.roundRobincount++
	return server
}

func (lb *Loadbalancer) serveProxy(rw http.ResponseWriter, req *http.Request) {
	targetserver := lb.getNextAvailabeServer()
	fmt.Printf("forwarding request to address %q\n", targetserver.Address())
	targetserver.Serve(rw, *req)
}

func main() {
	servers := []Server{
		newsimpleserver("https://www.facebook.com"),
		newsimpleserver("https://www.bing.com"),
		newsimpleserver("https://www.duckduckgo.com"),
	}
	lb := NewLoadBalancer("8000", servers)
	handleRedirect := func(rw http.ResponseWriter, req *http.Request) {
		lb.serveProxy(rw, req)
	}
	http.HandleFunc("/", handleRedirect)
	fmt.Printf("Serving request at 'localhost:%s'\n", lb.port)
	http.ListenAndServe(":"+lb.port, nil)
}
