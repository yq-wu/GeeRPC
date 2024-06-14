package registry

import (
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	defaultPath    = "/_geerpc_/registry"
	defaultTimeout = time.Minute * 5
)

type GeeRegistry struct {
	mu      sync.Mutex
	timeout time.Duration
	servers map[string]*ServerIterm
}

type ServerIterm struct {
	addr  string
	start time.Time
}

func New(timeout time.Duration) *GeeRegistry {
	return &GeeRegistry{
		timeout: timeout,
		servers: make(map[string]*ServerIterm),
	}
}

var DefaultGeeRegister = New(defaultTimeout)

func (g *GeeRegistry) putServer(addr string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if server, ok := g.servers[addr]; ok {
		server.start = time.Now()
	} else {
		g.servers[addr] = &ServerIterm{
			addr:  addr,
			start: time.Now(),
		}
	}
}
func (g *GeeRegistry) aliveServers() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	var aliveServers []string
	for addr, server := range g.servers {
		if g.timeout == 0 || server.start.Add(g.timeout).After(time.Now()) {
			aliveServers = append(aliveServers, addr)
		} else {
			delete(g.servers, addr)
		}
	}
	sort.Strings(aliveServers)
	return aliveServers
}

// Runs at /_geerpc_/registry
func (r *GeeRegistry) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET":
		// keep it simple, server is in req.Header
		w.Header().Set("X-Geerpc-Servers", strings.Join(r.aliveServers(), ","))
	case "POST":
		// keep it simple, server is in req.Header
		addr := req.Header.Get("X-Geerpc-Server")
		if addr == "" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		r.putServer(addr)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// HandleHTTP registers an HTTP handler for GeeRegistry messages on registryPath
func (r *GeeRegistry) HandleHTTP(registryPath string) {
	http.Handle(registryPath, r)
	log.Println("rpc registry path:", registryPath)
}

func HandleHTTP() {
	DefaultGeeRegister.HandleHTTP(defaultPath)
}

func Heartbeat(registry, addr string, duration time.Duration) {
	if duration == 0 {
		duration = defaultTimeout - time.Duration(1)*time.Minute
	}
	var err error
	err = sendHeartbeat(registry, addr)
	go func() {
		t := time.NewTicker(duration)
		for err == nil {
			<-t.C
			err = sendHeartbeat(registry, addr)
		}
	}()
}

func sendHeartbeat(registry, addr string) error {
	log.Println(addr, "send heart beat to registry", registry)
	httpClient := &http.Client{}
	req, _ := http.NewRequest("POST", registry, nil)
	req.Header.Set("X-Geerpc-Server", addr)
	if _, err := httpClient.Do(req); err != nil {
		log.Println("rpc server: heart beat err:", err)
		return err
	}
	return nil
}
