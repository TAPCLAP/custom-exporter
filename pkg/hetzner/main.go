package hetzner

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"os"

	hrobot "github.com/nl2go/hrobot-go"
)

var (
	updateServerList sync.Mutex
)


type Hetzner struct {
	hrobotServers []HrobotServer
	hrobot 	      hrobot.RobotClient
}


type HrobotServer struct {
	ID     int64	`json:"id"`
	Name   string	`json:"name"`
	Type   string	`json:"type"`
	Zone   string   `json:"zone"`
	Region string	`json:"region"`
	IP     net.IP   `json:"ip"`
}


func NewHetzner(user string, pass string) (*Hetzner) {
	hetzner := Hetzner{}
	hetzner.hrobot = hrobot.NewBasicAuthClient(user, pass)
	return &hetzner
}

func (h *Hetzner) readHrobotServers() error {
	servers, err := h.hrobot.ServerGetList()
	if err != nil {
		return err
	}
	var hservers []HrobotServer
	for _, s := range servers {
		region := strings.ToLower(strings.Split(s.Dc, "-")[0])
		server := HrobotServer{
			ID:     int64(s.ServerNumber),
			Name:   s.ServerName,
			Type:   s.Product,
			Zone:   strings.ToLower(s.Dc),
			Region: region,
			IP:     net.ParseIP(s.ServerIP),
		}
		hservers = append(hservers, server)
	}
	updateServerList.Lock()
	h.hrobotServers = hservers
	updateServerList.Unlock()
	return nil
}

func (h *Hetzner) GetServers() []HrobotServer {
	err := h.readHrobotServers()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading hrobot servers: %v\n", err)
	}
	return h.hrobotServers
}