package orchestrator

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"sync"
	"time"
)

type Server struct {
	Name     string  `json:"name"`
	IP       string  `json:"ip"`
	Port     int     `json:"port"`
	Active   bool    `json:"active"`
	CPU      float64 `json:"cpu,omitempty"`
	LastSeen int64   `json:"last_seen"`
}

type Orchestrator struct {
	mu          sync.RWMutex
	servers    map[string]*Server
	primaryIP  string
	haproxyCfg string
}

var (
	global *Orchestrator
	once  sync.Once
)

func Init(primaryIP, haproxyConfig string) *Orchestrator {
	once.Do(func() {
		global = &Orchestrator{
			servers:    make(map[string]*Server),
			primaryIP: primaryIP,
			haproxyCfg: haproxyConfig,
		}
	})
	return global
}

func Get() *Orchestrator {
	if global == nil {
		return Init("127.0.0.1", "/etc/haproxy/haproxy.cfg")
	}
	return global
}

func (o *Orchestrator) AddServer(name, ip string, port int) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	
	if _, ok := o.servers[name]; ok {
		return fmt.Errorf("server %s already exists", name)
	}
	
	server := &Server{
		Name:     name,
		IP:       ip,
		Port:     port,
		Active:   true,
		LastSeen: time.Now().Unix(),
	}
	o.servers[name] = server
	
	if err := o.updateHAProxy(name, ip, port, "add"); err != nil {
		log.Printf("Error updating HAProxy: %v", err)
	}
	
	log.Printf("Server added: %s (%s:%d)", name, ip, port)
	return nil
}

func (o *Orchestrator) RemoveServer(name string) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	
	server, ok := o.servers[name]
	if !ok {
		return fmt.Errorf("server %s not found", name)
	}
	
	if err := o.updateHAProxy(name, server.IP, server.Port, "remove"); err != nil {
		log.Printf("Error updating HAProxy: %v", err)
	}
	
	delete(o.servers, name)
	log.Printf("Server removed: %s", name)
	return nil
}

func (o *Orchestrator) GetServers() []Server {
	o.mu.RLock()
	defer o.mu.RUnlock()
	
	result := make([]Server, 0, len(o.servers))
	for _, s := range o.servers {
		result = append(result, *s)
	}
	return result
}

func (o *Orchestrator) GetActiveCount() int {
	o.mu.RLock()
	defer o.mu.RUnlock()
	
	count := 0
	for _, s := range o.servers {
		if s.Active {
			count++
		}
	}
	return count
}

func (o *Orchestrator) UpdateServerStatus(name string, active bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	
	if s, ok := o.servers[name]; ok {
		s.Active = active
		s.LastSeen = time.Now().Unix()
	}
}

func (o *Orchestrator) updateHAProxy(name, ip string, port int, action string) error {
	scriptPath := fmt.Sprintf("./scripts/haproxy_%s.sh", action)
	cmd := exec.Command("sh", scriptPath, name, ip, fmt.Sprintf("%d", port))
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Script output: %s", string(output))
	}
	return err
}

func (o *Orchestrator) ToJSON() string {
	o.mu.RLock()
	defer o.mu.RUnlock()
	
	data := struct {
		Primary string   `json:"primary"`
		Servers []Server `json:"servers"`
	}{
		Primary: o.primaryIP,
		Servers: o.GetServers(),
	}
	
	b, _ := json.Marshal(data)
	return string(b)
}