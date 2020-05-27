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
	deadlineSSH = time.Minute * 5 // abort if we can't get it within this reasonable period
	// retrySSH defines the frequency of SSH connect attempts
	retrySSH = 5 * time.Second

	autoscaleRetries = 20               // total number of attempts when checking autoscale changes
	autoscaleWait    = time.Second * 15 // amount of time to wait between attempts to autoscale the cluster

	// minimum required disk speed (10MB/s)
	minDiskSpeed = uint64(1e7)
)

var DefaultTimeouts = OpTimeouts{
	Install:          time.Minute * 15, // install threshold per node
	Upgrade:          time.Minute * 30, // upgrade threshold per node
	Uninstall:        time.Minute * 5,  // uninstall threshold per node
	UninstallApp:     time.Minute * 5,  // application uninstall threshold
	NodeStatus:       time.Minute * 1,  // limit for status to return on a single node
	ClusterStatus:    time.Minute * 5,  // limit for status to queisce across the cluster
	Leave:            time.Minute * 15, // threshold to leave cluster
	CollectLogs:      time.Minute * 7,  // to collect logs from node
	WaitForInstaller: time.Minute * 30, // wait for build to complete in parallel
	AutoScaling:      time.Minute * 10, // wait for autoscaling operation
	TimeSync:         time.Minute * 5,  // wait for ntp to converge
	ResolveInPlanet:  time.Minute * 1,  // resolve a hostname inside planet with dig
	GetPods:          time.Minute * 1,  // use kubectl to query pods on the API master
	ExecScript:       time.Minute * 5,  // user provided script, this should be specified by the user
}
