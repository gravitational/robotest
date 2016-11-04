package utils

import (
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gravitational/trace"
)

const (
	// exponentialRetryInitialDelay is the interval between the first and second retry attempts
	exponentialRetryInitialDelay = 5 * time.Second
	// exponentialRetryMaxDelay is the maximum delay between retry attempts
	exponentialRetryMaxDelay = 30 * time.Second
)

// Abort causes Retry function to stop with error
func Abort(err error) *AbortRetry {
	return &AbortRetry{Err: err}
}

// Continue causes Retry function to continue trying and logging message
func Continue(message string) *ContinueRetry {
	return &ContinueRetry{Message: message}
}

// AbortRetry if returned from Retry, will lead to retries to be stopped,
// but the Retry function will return internal Error
type AbortRetry struct {
	Err error
}

func (a *AbortRetry) Error() string {
	return fmt.Sprintf("Abort(%v)", a.Err)
}

// ContinueRetry if returned from Retry, will be lead to retry next time
type ContinueRetry struct {
	Message string
}

func (s *ContinueRetry) Error() string {
	return fmt.Sprintf("ContinueRetry(%v)", s.Message)
}

// Retry attempts to execute fn up to maxAttempts sleepig for period between attempts.
// fn can return an instance of Abort to abort or Continue to continue the execution.
func Retry(period time.Duration, maxAttempts int, fn func() error) error {
	var err error
	for i := 1; i <= maxAttempts; i += 1 {
		err = fn()
		if err == nil {
			return nil
		}
		switch origErr := err.(type) {
		case *AbortRetry:
			return origErr.Err
		case *ContinueRetry:
			log.Infof("%v retry in %v", origErr.Message, period)
		default:
			log.Infof("unsuccessful attempt %v: %v, retry in %v", i, trace.UserMessage(err), period)
		}
		time.Sleep(period)
	}
	log.Errorf("all attempts failed:\n%v", trace.DebugReport(err))
	return err
}
