package wait

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/gravitational/robotest/lib/defaults"
	"github.com/gravitational/trace"
	log "github.com/sirupsen/logrus"
)

// Abort causes Retry function to stop with error
func Abort(err error) AbortRetry {
	return AbortRetry{Err: err}
}

// Continue causes Retry function to continue trying and logging message
func Continue(format string, args ...interface{}) ContinueRetry {
	message := fmt.Sprintf(format, args...)
	return ContinueRetry{Message: message}
}

// AbortRetry if returned from Retry, will lead to retries to be stopped,
// but the Retry function will return internal Error
type AbortRetry struct {
	Err error
}

func (r AbortRetry) Error() string {
	return fmt.Sprintf("Abort(%v)", r.Err)
}

// ContinueRetry if returned from Retry, will be lead to retry next time
type ContinueRetry struct {
	Message string
}

func (r ContinueRetry) Error() string {
	return fmt.Sprintf("ContinueRetry(%v)", r.Message)
}

// Retry attempts to execute fn with default delay retrying it for a default number of attempts.
// fn can return AbortRetry to abort or ContinueRetry to continue the execution.
func Retry(ctx context.Context, fn func() error) error {
	r := Retryer{
		Delay:    defaults.RetryDelay,
		Attempts: defaults.RetryAttempts,
	}
	return r.Do(ctx, fn)
}

// Do retries the given function fn for the configured number of attempts until it succeeds
// or all attempts have been exhausted
func (r Retryer) Do(ctx context.Context, fn func() error) (err error) {
	if r.FieldLogger == nil {
		r.FieldLogger = log.NewEntry(log.StandardLogger())
	}

	if ctx.Err() != nil {
		return trace.Wrap(ctx.Err())
	}

	for i := 1; i <= r.Attempts; i += 1 {
		err = fn()
		if err == nil {
			r.Debug("succeded")
			return nil
		}

		le := r.FieldLogger
		if deadline, ok := ctx.Deadline(); ok {
			le = le.WithField("timeout-in", fmt.Sprintf("%v", time.Until(deadline)))
		}
		switch origErr := err.(type) {
		case AbortRetry:
			le.WithError(err).Error("aborted")
			return origErr.Err
		case ContinueRetry:
			le.Debugf("%v retry in %v", origErr.Message, r.Delay)
		default:
			le.Debugf("unsuccessful attempt %v: %v, retry in %v", i, trace.UserMessage(err), r.Delay)
		}

		select {
		case <-time.After(backoff(r.Delay, i)):
		case <-ctx.Done():
			r.Error("context timed out")
			return err
		}
	}
	r.Errorf("all attempts failed:\n%v", trace.DebugReport(err))
	return err
}

// Retryer is a process that can retry a function
type Retryer struct {
	// Delay specifies the interval between retry attempts
	Delay time.Duration
	// Attempts specifies the number of attempts to execute before failing.
	// Should be >= 1, zero value is not useful
	Attempts int
	// FieldLogger specifies the log sink
	log.FieldLogger
}

func backoff(baseDelay time.Duration, errCount int) time.Duration {
	delay := baseDelay * time.Duration(math.Pow(2, float64(errCount)-1))
	if delay > defaults.RetryMaxDelay {
		return defaults.RetryMaxDelay
	} else {
		return delay
	}
}
