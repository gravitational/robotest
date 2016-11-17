package defaults

import "time"

const (
	// RetryDelay defines the interval between retry attempts
	RetryDelay = 5 * time.Second
	// RetryAttempts defines the maximum number of retry attempts
	RetryAttempts = 100

	// FindTimeout defines the timeout to use for lookup operations
	FindTimeout = 20 * time.Second
)
