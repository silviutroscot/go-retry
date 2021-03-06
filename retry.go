//   Copyright 2020 Vimeo
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package retry

import (
	"context"
	"fmt"
	"time"
)

// Retryable manages the operations of a retryable operation.
type Retryable struct {
	// Backoff parameters to use for retry
	B Backoff

	// ShouldRetry is a filter function to indicate whether to continue
	// iterating based on the error.
	// An implementation that uniformly returns true is used if nil
	ShouldRetry func(error) bool

	// Maximum retry attempts
	MaxSteps int32
}

// NewRetryable returns a newly constructed Retryable instance
func NewRetryable(MaxSteps int32) *Retryable {
	return &Retryable{
		B:           DefaultBackoff(),
		ShouldRetry: nil,
		MaxSteps:    MaxSteps,
	}
}

// Retry calls the function `f` at most `MaxSteps` times using the exponential
// backoff parameters defined in `B`, or until the context expires.
func (r *Retryable) Retry(ctx context.Context, f func(context.Context) error) error {
	b := r.B.Clone()
	b.Reset()
	filter := r.ShouldRetry
	if filter == nil {
		filter = func(err error) bool {
			return true
		}
	}
	errors := make([]error, 0, 0)
	for n := int32(0); n < r.MaxSteps; n++ {
		err := f(ctx)
		if err == nil {
			return nil
		}
		if !filter(err) {
			return err
		}
		errors = append(errors, err)
		select {
		case <-time.After(b.Next()):
			continue
		case <-ctx.Done():
			return fmt.Errorf(
				"context expired while retrying: %s. retried %d times",
				ctx.Err(), n)
		}
	}
	return fmt.Errorf("aborting retry. errors: %+v", errors)
}

// Retry calls the function `f` at most `steps` times using the exponential
// backoff parameters defined in `b`, or until the context expires.
func Retry(ctx context.Context, b Backoff, steps int, f func(context.Context) error) error {
	// Make sure b is clean.
	b.Reset()
	r := Retryable{B: b, MaxSteps: int32(steps)}
	return r.Retry(ctx, f)
}
