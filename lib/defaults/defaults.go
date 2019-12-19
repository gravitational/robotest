package defaults

import "time"

const (
	// AgentLogPath defines the location of the install agent log on the remote
	// node
	AgentLogPath = "gravity-system.log"

	// AgentShrinkLogPath defines the location of the shrink agent log on the remote
	// node
	AgentShrinkLogPath = "gravity-system.log"

	// ReportPath defines path to report file generated by `gravity report` command
	ReportPath = "/var/lib/gravity/crashreport.tar.gz"

	// RetryDelay defines the interval between retry attempts
	RetryDelay = 5 * time.Second
	// RetryMaxDelay defines maximum interval between retry attempts
	RetryMaxDelay = time.Minute
	// RetryAttempts defines the maximum number of retry attempts
	RetryAttempts = 100

	// SSHConnectTimeout defines the timeout for establishing an SSH connection
	SSHConnectTimeout = 30 * time.Second

	// MinDiskSpeed is minimum write performance
	MinDiskSpeed = uint64(1e7)

	// NodeRole is the default role when installing/joining a node
	NodeRole = "node"

	// GravityDir is the default location of all gravity data on a node
	GravityDir = "/var/lib/gravity"

	// EtcdRetryTimeout specifies the total timeout for retrying etcd commands
	// in case of transient errors
	EtcdRetryTimeout = 5 * time.Minute

	// TerraformRetryDelay
	TerraformRetryDelay = 5 * time.Minute

	// ClusterStatusTimeout specifies the maximum amount of time to wait for cluster status
	ClusterStatusTimeout = 5 * time.Minute

	// TerraformRetries is the maximum number of attempts to reprovision the
	// infrastructure upon encountering an error from 'terraform apply'
	TerraformRetries = 2

	// BQDataset is the BigQuery dataset where run data is stored
	BQDataset = "robotest"

	// BQTable is the BigQuery table where progress data is stored
	BQTable = "progress"

	// MaxRetriesPerTest specifies the maximum number of attempts a failing
	// test is retried (including the first failure)
	MaxRetriesPerTest = 3
	// MaxPreemptedRetriesPerTest specifies the maximum number of node preemptions
	// to tolerate before aborting the test
	MaxPreemptedRetriesPerTest = 10

	// TmpDir is temporary file folder
	TmpDir = "/tmp"
)
