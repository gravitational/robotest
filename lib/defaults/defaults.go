package defaults

import "time"

const (
	// AgentLogPath defines the location of the install agent log on the remote
	// node
	AgentLogPath = "/var/log/gravity.agent.log"

	// AgentShrinkLogPath defines the location of the shrink agent log on the remote
	// node
	AgentShrinkLogPath = "/var/log/gravity.agent.shrink.log"

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

	// TerraformRetries
	TerraformRetries = 2

	// BQDataset is bigquery dataset where run data is stored
	BQDataset = "robotest"

	// BQTable is bigquery table where run data is stored
	BQTable = "progress"

	// MaxRetriesPerTest
	MaxRetriesPerTest = 2
)
