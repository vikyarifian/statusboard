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

