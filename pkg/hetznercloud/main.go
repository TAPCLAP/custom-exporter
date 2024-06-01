package hetznercloud

import (
	"fmt"
	"net"
	// "strings"
	"sync"
	"os"
	"context"	

	"github.com/hetznercloud/hcloud-go/v2/hcloud"

)

var (
	updateServerList sync.Mutex
)



type Server struct {
	ID     int64	`json:"id"`
	Name   string	`json:"name"`
	Type   string	`json:"type"`
	Zone   string   `json:"zone"`
	Region string	`json:"region"`
	IP     net.IP   `json:"ip"`
}

type ClientConfig struct {
	Token string
}

type Client struct {
	name string
	client *hcloud.Client
}

type HetznerClouds struct {
	servers []Server
	clients []Client
}


func NewHetznerClouds(clientClouds []ClientConfig) (*HetznerClouds) {
	h := HetznerClouds{}
	for _, c := range clientClouds {
		h.clients = append(h.clients, Client{
			client: hcloud.NewClient(hcloud.WithToken(c.Token)),
		})
	}

	return &h
}

func (h *HetznerClouds) getServers() {
	for _, c := range h.clients {
		servers, _, err := c.client.Server.List(context.Background(), hcloud.ServerListOpts{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading hcloud servers: %v\n", err)
			continue
		}
		updateServerList.Lock()
		for _, s := range servers {
			h.servers = append(h.servers, Server{
				ID:     s.ID,
				Name:   s.Name,
				Type:   s.ServerType.Name,
				Zone:   s.Datacenter.Name,
				Region: s.Datacenter.Location.Name,
				IP:     net.ParseIP(s.PublicNet.IPv4.IP.String()),
			})
		}
		updateServerList.Unlock()
	}
}

func (h *HetznerClouds) GetServers() []Server {
	h.getServers()
	return h.servers
}