package defaults

import "time"

const (
	// RetryDelay defines the interval between retry attempts
	RetryDelay = 5 * time.Second
	// RetryAttempts defines the maximum number of retry attempts
	RetryAttempts = 100

	// FindTimeout defines the timeout to use for lookup operations
	FindTimeout = 20 * time.Second

	// AgentTimeout defines the amount of time to wait for agents to connect
	AgentServerTimeout = 5 * time.Minute

	// InstallTimeout defines the amount of time to wait for installation to complete
	InstallTimeout = 20 * time.Minute

	// PollInterval defines the frequency of polling attempts
	PollInterval = 10 * time.Second

	PauseTimeout = 100 * time.Millisecond

	ShortTimeout = 1 * time.Second

	AjaxCallTimeout   = 20 * time.Second
	ServerLoadTimeout = 20 * time.Second
	ElementTimeout    = 20 * time.Second
	OperationTimeout  = 5 * time.Minute

	ProfileLabel = "worker node"
	InstanceType = "m3.large"
)
