package system

import (
	"fmt"
	"hash/crc32"
	"os"
	"strconv"
	"strings"
	// "encoding/binary"
	// "encoding/json"
	"os/exec"
)


func HostnameChecksum() (float64, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return 0, fmt.Errorf("failed to get hostname: %w", err)
	}

	tablePolynomial := crc32.MakeTable(crc32.IEEE)
	hash := crc32.Checksum([]byte(hostname), tablePolynomial)

	return float64(hash), nil
}


func UptimeInSeconds() (float64, error) {
	uptimeFile := "/proc/uptime"
	content, err := os.ReadFile(uptimeFile)
	if err != nil {
		return 0, fmt.Errorf("failed to read uptime file: %w", err)
	}

	fields := strings.Fields(string(content))
	if len(fields) < 1 {
		return 0, fmt.Errorf("invalid format in uptime file")
	}

	uptimeSeconds, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse uptime value: %w", err)
	}

	return uptimeSeconds, nil
}

func CountLoginUsers() (int, error) {
	output, err := exec.Command("who", "-q").Output()
	if err != nil {
		return 0, fmt.Errorf("failed to execute 'who -q': %w", err)
	}
	usersLine := strings.TrimSpace(string(output))
	usersField := strings.Split(usersLine, "=")
	if len(usersField) != 2 {
		return 0, fmt.Errorf("invalid format in 'who -q' output")
	}
	numUsers, err := strconv.Atoi(usersField[1])
	if err != nil {
		return 0, fmt.Errorf("failed to parse number of users: %w", err)
	}
	return numUsers, nil
}

