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
)
