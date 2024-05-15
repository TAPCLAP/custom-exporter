package proc

import (
	"regexp"	
	"github.com/prometheus/procfs"
	
)

var (
	processTypesMap = map[string]string{
		"R": "running",
		"I": "idle",
		"S": "sleeping",
		"D": "uninterruptible_disk_sleep",
		"Z": "zombie",
		"T": "stopped",
		"t": "tracing_stop",
		"X": "dead",
		"x": "dead",
		"K": "wakekill",
		"W": "waking",
		"P": "parked",
	}
)

type ProcessFilter struct {
	Process string `yaml:"process"`
	Regex   string `yaml:"regex"`
}

type ProcessUsage struct {
	CPUTime        float64
	ResidentMemory uint64
}

func CountProcesses() (int, error) {
	procs, err := procfs.AllProcs()
	if err != nil {
		return 0, err
	}
	return len(procs), nil
}

func CountProcessTypes() (map[string]int, error) {
	procs, err := procfs.AllProcs()
	if err != nil {
		return nil, err
	}

	processTypes := make(map[string]int)
	for _, proc := range procs {
		stat, err := proc.Stat()
		if err != nil {
			return nil, err
		}
		processTypes[processTypesMap[stat.State]]++
	}

	return processTypes, nil
}


func findProcessByName(regex string) ([]int, error) {
	procs, err := procfs.AllProcs()
	if err != nil {
		return nil, err
	}
	var matchingProcs []int
	for _, proc := range procs {
		stat, err := proc.Stat()
		if err != nil {
			return nil, err
		}
		match, err := regexp.MatchString(regex, stat.Comm)
		if err != nil {
			return nil, err
		}
		if match {
			matchingProcs = append(matchingProcs, proc.PID)
		}
	}
	return matchingProcs, nil
}

func aggregateCPUTime(pids []int) (float64, error) {
	totalCPUTime := 0.0
	for _, pid := range pids {
		proc, err := procfs.NewProc(pid)
		if err != nil {
			return 0, err
		}
		stat, err := proc.Stat()
		if err != nil {
			return 0, err
		}
		totalCPUTime += float64(stat.CPUTime())
	}
	return totalCPUTime, nil
}

func aggregateResidentMemoryUsage(pids []int) (uint64, error) {
	totalMemory := uint64(0)
	for _, pid := range pids {
		proc, err := procfs.NewProc(pid)
		if err != nil {
			return 0, err
		}
		stat, err := proc.Stat()
		if err != nil {
			return 0, err
		}
		totalMemory += uint64(stat.ResidentMemory())
	}
	return totalMemory, nil
}

func AggregateCPUTimeAndMemoryUsageByRegex(processes []ProcessFilter) (map[string]ProcessUsage, error) {
	result := make(map[string]ProcessUsage)

	for _, proc := range processes {
		matchingProcs, err := findProcessByName(proc.Regex)
		if err != nil {
			return nil, err
		}

		pids := []int{}
		for _, pid := range matchingProcs {
			pids = append(pids, pid)
		}

		totalCPUTime, err := aggregateCPUTime(pids)
		if err != nil {
			return nil, err
		}

		totalMemory, err := aggregateResidentMemoryUsage(pids)
		if err != nil {
			return nil, err
		}

		result[proc.Process] = struct {
			CPUTime         float64
			ResidentMemory  uint64
		}{
			CPUTime:        totalCPUTime,
			ResidentMemory: totalMemory,
		}
	}

	return result, nil
}

func FindProcessesByRegex(processes []ProcessFilter) (map[string]int, error) {
	result := make(map[string]int)
	for _, proc := range processes {
		matchingProcs, err := findProcessByName(proc.Regex)
		if err != nil {
			return nil, err
		}
		if len(matchingProcs) > 0 {
			result[proc.Process] = 1
		} else {
			result[proc.Process] = 0
		}
	}
	return result, nil
}