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
		[]string{"process"},
	)

	processCpuTimeCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "process_cpu_time_total",
			Help: "CPU time consumed by processes",
		},
		[]string{"process"},
	)

	processMemoryResidentGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "process_memory_resident",
			Help: "Resident memory of processes in bytes",
		},
		[]string{"process"},
	)

	hostnameChecksumGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "hostname_checksum",
			Help: "Checksum of hostname",
		},
	)
	
	hostnameGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "hostname",
			Help: "Hostname of the machine",
		},
		[]string{"hostname"},
	)

	uptimeSecondsCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "uptime_seconds",
			Help: "Uptime of the machine in seconds",
		},
	)

	countLoginUsersGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "login_users_count",
			Help: "Number of login users",
		},
	)

	puppetCatalogLastCompileTimestampGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "puppet_catalog_last_compile_timestamp",
			Help: "Timestamp of the last puppet catalog compile",
		},
	)

	puppetCatalogLastCompileStatusGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "puppet_catalog_last_compile_status",
			Help: "Status of the last puppet catalog compile",
		},
	)


	previousHostnameLabel string

	fileHashMutex sync.Mutex
	networkTargetMutex sync.Mutex
	processCountMutex sync.Mutex
	processCpuTimeMutex sync.Mutex
	processMemoryResidentMutex sync.Mutex
	processRunningStatusMutex sync.Mutex
	hostnameChecksumMutex sync.Mutex
	hostnameMutex sync.Mutex
	uptimeSecondsMutex sync.Mutex
	countLoginUsersMutex sync.Mutex
	puppetCatalogLastCompileTimestampMutex sync.Mutex
	puppetCatalogLastCompileStatusMutex sync.Mutex
)

func RegistrMetrics() {
    prometheus.DefaultRegisterer.Unregister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
    prometheus.DefaultRegisterer.Unregister(collectors.NewGoCollector())

	prometheus.MustRegister(hashGauge)
	prometheus.MustRegister(networkTargetGauge)
	prometheus.MustRegister(processCountGauge)
	prometheus.MustRegister(processCpuTimeCounter)
	prometheus.MustRegister(processMemoryResidentGauge)
	prometheus.MustRegister(processRunningStatusGauge)
	prometheus.MustRegister(hostnameChecksumGauge)
	prometheus.MustRegister(hostnameGauge)
	prometheus.MustRegister(uptimeSecondsCounter)
	prometheus.MustRegister(countLoginUsersGauge)
	prometheus.MustRegister(puppetCatalogLastCompileTimestampGauge)
	prometheus.MustRegister(puppetCatalogLastCompileStatusGauge)
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

func UpdateProcessMemoryResidentMetrics(process string, memory uint64) {
	processMemoryResidentMutex.Lock()
	processMemoryResidentGauge.WithLabelValues(process).Set(float64(memory))
	processMemoryResidentMutex.Unlock()
}

func UpdateProcessRunningStatusMetrics(process string, status int) {
	processRunningStatusMutex.Lock()
	processRunningStatusGauge.WithLabelValues(process).Set(float64(status))
	processRunningStatusMutex.Unlock()
}

func UpdateHostnameChecksumMetrics(checksum float64) {
	hostnameChecksumMutex.Lock()
	hostnameChecksumGauge.Set(checksum)
	hostnameChecksumMutex.Unlock()
}

func UpdateHostnameMetrics(hostname string) {
	hostnameMutex.Lock()
	if previousHostnameLabel != "" && previousHostnameLabel != hostname {
		hostnameGauge.DeleteLabelValues(previousHostnameLabel)
	}
	hostnameGauge.WithLabelValues(hostname).Set(1)
	previousHostnameLabel = hostname
	hostnameMutex.Unlock()
}

func UpdateUptimeSecondsMetrics(uptime float64) {
	uptimeSecondsMutex.Lock()
	uptimeSecondsCounter.Add(uptime)
	uptimeSecondsMutex.Unlock()
}

func UpdateLoginUsersCountMetrics(count int) {
	countLoginUsersMutex.Lock()
	countLoginUsersGauge.Set(float64(count))
	countLoginUsersMutex.Unlock()
}

func UpdatePuppetCatalogLastCompileTimestampMetrics(timestamp int64) {
	puppetCatalogLastCompileTimestampMutex.Lock()
	puppetCatalogLastCompileTimestampGauge.Set(float64(timestamp))
	puppetCatalogLastCompileTimestampMutex.Unlock()
}

func UpdatePuppetCatalogLastCompileStatusMetrics(status bool) {
	statusValue := 0
	if status {
		statusValue = 1
	}
	puppetCatalogLastCompileStatusMutex.Lock()
	puppetCatalogLastCompileStatusGauge.Set(float64(statusValue))
	puppetCatalogLastCompileStatusMutex.Unlock()
}