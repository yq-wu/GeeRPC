package xclient

import (
	"errors"
	"math"
	"math/rand"
	"sync"
	"time"
)

type selectMode int

const (
	RandomSelect selectMode = iota
	RoundRobinSelect
)

type discovery interface {
	Refresh() error
	Update([]string) error
	Get(selectMode) (string, error)
	GetAll() ([]string, error)
}

type MultiServersDiscovery struct {
	r       *rand.Rand
	mu      sync.RWMutex
	servers []string
	index   int
}

func NewMultiServersDiscovery(server []string) *MultiServersDiscovery {
	m := &MultiServersDiscovery{
		r:       rand.New(rand.NewSource(time.Now().UnixNano())), // 根据时间生成随机数种子，再生成随机数
		servers: server,
	}
	m.index = m.r.Intn(math.MaxInt32 - 1) // 轮询时候避免每次都从0开始
	return m
}

func (m *MultiServersDiscovery) Refresh() error {
	// todo
	return nil
}

func (m *MultiServersDiscovery) Update(servers []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.servers = servers
	return nil
}

func (m *MultiServersDiscovery) Get(mode selectMode) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := len(m.servers)
	if n == 0 {
		return "", errors.New("rpc discovery: no available servers")
	}
	switch mode {
	case RandomSelect:
		return m.servers[m.r.Intn(n)], nil
	case RoundRobinSelect:
		index := m.index % n
		m.index = index + 1
		return m.servers[index], nil
	default:
		return "", errors.New("rpc discovery: not supported select mode")
	}
}

func (m *MultiServersDiscovery) GetAll() ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	servers := make([]string, len(m.servers), cap(m.servers))
	copy(servers, m.servers)
	return servers, nil
}
