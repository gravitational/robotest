package gravity

import (
	"time"
)

// default timeout to wait for cloud-init to complete
var cloudInitTimeout = time.Minute * 30

// default timeout to wait for clocks to synchronize between nodes
var clockSyncTimeout = time.Minute * 15

// default timeout to wait for I/O stabilize on VMs
var diskWaitTimeout = time.Minute * 10

const (
	retrySSH    = time.Second * 10
	deadlineSSH = time.Minute * 5 // abort if we can't get it within this reasonable period

	// minimum required disk speed (10MB/s)
	minDiskSpeed = uint64(1e7)
)

var DefaultTimeouts = OpTimeouts{
	Install:          time.Minute * 15, // install threshold per node
	Upgrade:          time.Minute * 30, // upgrade threshold per node
	Uninstall:        time.Minute * 5,  // uninstall threshold per node
	Status:           time.Minute * 30, // sufficient for failover procedures
	Leave:            time.Minute * 15, // threshold to leave cluster
	CollectLogs:      time.Minute * 7,  // to collect logs from node
	WaitForInstaller: time.Minute * 10, // wait for build to complete in parallel
}
