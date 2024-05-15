package network

import (
	"fmt"
	"net"
	"os"
	"sync"
	"time"

    // "github.com/sirupsen/logrus"
)


type Target struct {
    Host string `yaml:"host"`
    Port uint16 `yaml:"port"`
    Protocol string `yaml:"protocol"`
}

type ResultTarget struct {
    Host string 
    Port uint16
    Protocol string
    IsOpen bool
}

func checkTCPTarget(t Target, results chan <- ResultTarget, wg *sync.WaitGroup) {
    defer wg.Done()

    address := fmt.Sprintf("%s:%d", t.Host, t.Port)
    conn, err := net.DialTimeout("tcp", address, time.Second * 5)
    if err != nil {
        fmt.Fprintf(os.Stderr, "checkTCPTarget: try connect to host: %s, error: %s\n", address, err)
        results <- ResultTarget{
            Host: t.Host, 
            Port: t.Port,
            Protocol: t.Protocol,
            IsOpen: false,
        }
        return
    }
    defer conn.Close()
    results <- ResultTarget{
        Host: t.Host, 
        Port: t.Port,
        Protocol: t.Protocol,
        IsOpen: true,
    }
    
}

func checkUDPTarget(t Target, results chan <- ResultTarget, wg *sync.WaitGroup) {
    defer wg.Done()

    address := fmt.Sprintf("%s:%d", t.Host, t.Port)
    result := ResultTarget{
        Host: t.Host, 
        Port: t.Port,
        Protocol: t.Protocol,
        IsOpen: false,
    }
    conn, err := net.DialTimeout("udp", address, time.Second * 5)
    if err != nil {
        fmt.Fprintf(os.Stderr, "checkUDPTarget: try connect to host: %s, error: %s\n", address, err)
        results <- result
        return
    }
    defer conn.Close()

    conn.SetReadDeadline(time.Now().Add(2 * time.Second))

	_, err = conn.Write([]byte("hello"))    
	if err != nil {
		fmt.Fprintf(os.Stderr, "checkUDPTarget: try send packet to host: %s, error: %s\n", address, err)
        results <- result
        return
	}

	buffer := make([]byte, 1024)
	_, err = conn.Read(buffer)
	if err != nil {
		fmt.Fprintf(os.Stderr, "checkUDPTarget: try response from host: %s, error: %s\n", address, err)
        results <- result
		return
	}

    result.IsOpen = true
    results <- result
}

func CheckTargets(targets []Target) []ResultTarget {
    var wg sync.WaitGroup
    results := make(chan ResultTarget)

    for _, t := range targets {
        if t.Protocol == "TCP" {
            wg.Add(1)
            go checkTCPTarget(t, results, &wg)
        }
        if t.Protocol == "UDP" {
            wg.Add(1)
            go checkUDPTarget(t, results, &wg)
        }
    }
    go func() {
        wg.Wait()
        close(results)
    }()
    
    var resultTargets []ResultTarget
	for r := range results {
        resultTargets = append(resultTargets, r)
	}
    return resultTargets
}