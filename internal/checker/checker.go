package checker

import (
	"net"
	"net/http"
	"strings"
	"time"
)

type CheckResult struct {
	Status    string
	LatencyMs int64
}

func PerformCheck(target string, checkType string, timeout time.Duration) CheckResult {
	if timeout <= 0 {
		timeout = 2 * time.Second
	}

	start := time.Now()

	if strings.ToLower(checkType) == "http" {
		url := target
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			url = "http://" + url
		}
		client := http.Client{Timeout: timeout}
		resp, err := client.Get(url)
		elapsed := time.Since(start).Milliseconds()
		if err != nil {
			return CheckResult{Status: "OFFLINE", LatencyMs: 0}
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			return CheckResult{Status: "ONLINE", LatencyMs: elapsed}
		}
		return CheckResult{Status: "DEGRADED", LatencyMs: elapsed}
	}

	conn, err := net.DialTimeout("tcp", target, timeout)
	elapsed := time.Since(start).Milliseconds()
	if err != nil {
		return CheckResult{Status: "OFFLINE", LatencyMs: 0}
	}
	conn.Close()

	return CheckResult{Status: "ONLINE", LatencyMs: elapsed}
}
