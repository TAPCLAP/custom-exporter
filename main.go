package main

import (
	"flag"
	"fmt"
	"os"
	"time"
	"net/http"
	// "encoding/json"

	"github.com/orangeAppsRu/custom-exporter/pkg/filehash"
	"github.com/orangeAppsRu/custom-exporter/pkg/config"
	"github.com/orangeAppsRu/custom-exporter/pkg/metrics"
	"github.com/orangeAppsRu/custom-exporter/pkg/network"	
	"github.com/orangeAppsRu/custom-exporter/pkg/proc"	


	"github.com/prometheus/client_golang/prometheus/promhttp"

)

func main() {
	configFilePath := flag.String("config", "", "path to config file")
	flag.Parse()

	if *configFilePath == "" {
		*configFilePath = os.Getenv("CONFIG")
	}

	if *configFilePath == "" {
		fmt.Println("Config file path is not provided. Use --config flag or set CONFIG environment variable.")
		return
	}

	cfg, err := config.ReadConfig(*configFilePath)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	host := os.Getenv("HOST")
	if host == "" {
		host = "127.0.0.1"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8200"
	}
	listenAddr := fmt.Sprintf("%s:%s", host, port)

	metrics.RegistrMetrics()

	if cfg.FileHashCollector.Enabled  {
		go func() {
			for {

				filesWithHash := []filehash.FileHash{}
				for _, filePath := range cfg.FileHashCollector.Files {
					if _, err := os.Stat(filePath); err == nil {
						number, err := filehash.Calculate(filePath)
						if err != nil {
							fmt.Printf("Error calculating hash for %s: %v\n", filePath, err)
							continue
						}
			
						filesWithHash = append(filesWithHash, filehash.FileHash{
							File: filePath,
							Hash: number,
						})
					}
				}
				metrics.UpdateFileHashMetrics(filesWithHash)
				time.Sleep(60 * time.Second)
			}
		}()
	}

	if cfg.PortCollector.Enabled  {
		go func() {
			for {
				rTargets := network.CheckTargets(cfg.PortCollector.Targets)
				metrics.UpdateNetworkTargetsMetrics(rTargets)
				time.Sleep(60 * time.Second)
			}
		}()
	}

	if cfg.ProcessCollector.Enabled {
		go func() {
			for {
				if np, err := proc.CountProcesses(); err != nil {
					fmt.Printf("Error counting processes: %v\n", err)
				} else {
					metrics.UpdateProcessCountMetrics("all", np)
				}

				if npt, err := proc.CountProcessTypes(); err != nil {
					fmt.Printf("Error counting processes: %v\n", err)
				} else {
					for typeProcess, count := range npt {
						metrics.UpdateProcessCountMetrics(typeProcess, count)
					}
				}

				if processResources, err := proc.AggregateCPUTimeAndMemoryUsageByRegex(cfg.ProcessCollector.Processes); err != nil {
					fmt.Printf("Error aggregating process resources: %v\n", err)
				} else {
					for process, usage := range processResources {
						metrics.UpdateProcessCPUTimeMetrics(process, usage.CPUTime)
					}
				}

				if processRunningStatus, err := proc.FindProcessesByRegex(cfg.ProcessCollector.Processes); err != nil { 
					fmt.Printf("Error finding processes: %v\n", err)
				} else {
					for process, count := range processRunningStatus {
						metrics.UpdateProcessRunningStatusMetrics(process, count)
					}
				}
				
				time.Sleep(60 * time.Second)
			}
		}()
	}
	
	
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	fmt.Println("Prometheus metrics server started at", listenAddr)
	if err := http.ListenAndServe(listenAddr, nil); err != nil {
		fmt.Printf("Error starting server: %s\n", err)
	}


}

