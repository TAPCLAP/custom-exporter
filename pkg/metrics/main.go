package metrics

import (
	"strconv"
	"sync"

	"github.com/orangeAppsRu/custom-exporter/pkg/filehash"
	"github.com/orangeAppsRu/custom-exporter/pkg/network"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

var (
	hashGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "file_hash",
			Help: "SHA256 hash of files",
		},
		[]string{"file"},
	)

	networkTargetGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "network_target",
			Help: "Network port availability",
		},
		[]string{"host", "port", "protocol"},
	)

	processCountGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "processes_count",
			Help: "Number of running processes",
		},
		[]string{"type"},
	)

	processRunningStatusGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "process_running_status",
			Help: "Status of process (running or not)",
		},
		[]string{"type"},
	)

	processCpuTimeCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "process_cpu_time_total",
			Help: "CPU time consumed by processes",
		},
		[]string{"process"},
	)


	fileHashMutex sync.Mutex
	networkTargetMutex sync.Mutex
	processCountMutex sync.Mutex
	processCpuTimeMutex sync.Mutex
	processRunningStatusMutex sync.Mutex
)

func RegistrMetrics() {
    prometheus.DefaultRegisterer.Unregister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
    prometheus.DefaultRegisterer.Unregister(collectors.NewGoCollector())

	prometheus.MustRegister(hashGauge)
	prometheus.MustRegister(networkTargetGauge)
	prometheus.MustRegister(processCountGauge)
	prometheus.MustRegister(processCpuTimeCounter)
	prometheus.MustRegister(processRunningStatusGauge)
}

func UpdateFileHashMetrics(filesWithHash []filehash.FileHash) {
	for _, fileInfo := range filesWithHash {
		fileHashMutex.Lock()
		hashGauge.WithLabelValues(fileInfo.File).Set(fileInfo.Hash)
		fileHashMutex.Unlock()
	}
}

func UpdateNetworkTargetsMetrics(targets []network.ResultTarget) {
	for _, t := range targets {
		value := 0
		if t.IsOpen {
			value = 1
		}
		networkTargetMutex.Lock()
		networkTargetGauge.With(prometheus.Labels{
			"host": t.Host,
			"port": strconv.Itoa(int(t.Port)),
			"protocol": t.Protocol,
		}).Set(float64(value))
		networkTargetMutex.Unlock()
	}
}

func UpdateProcessCountMetrics(typeProcess string, count int) {
	processCountMutex.Lock()
	processCountGauge.WithLabelValues(typeProcess).Set(float64(count))
	processCountMutex.Unlock()
}

func UpdateProcessCPUTimeMetrics(process string, time float64) {
	processCpuTimeMutex.Lock()
	processCpuTimeCounter.WithLabelValues(process).Add(time)
	processCpuTimeMutex.Unlock()
}

func UpdateProcessRunningStatusMetrics(process string, status int) {
	processRunningStatusMutex.Lock()
	processRunningStatusGauge.WithLabelValues(process).Set(float64(status))
	processRunningStatusMutex.Unlock()
}
