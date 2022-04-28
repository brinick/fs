package transaction

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Transactioner defines the interface for file system transactions
type Transactioner interface {
	opener
	closer
	starter
	stopper
	aborter
}

type opener interface {
	Open(context.Context) error
}

type closer interface {
	Close(context.Context) error
}

type starter interface {
	Start(context.Context) error
	Attempts() int
}

type stopper interface {
	Stop(context.Context) error
}
type aborter interface {
	Kill(context.Context) error
}

type OpenError struct {
	Err error
}

func (t OpenError) Error() string {
	return fmt.Sprintf("Transaction Open Error: %v", t.Err)
}

type CloseError struct {
	Err error
}

func (t CloseError) Error() string {
	return fmt.Sprintf("Transaction Close Error: %v", t.Err)
}

type AbortError struct {
	Err error
}

func (t AbortError) Error() string {
	return fmt.Sprintf("Transaction Abort Error: %v", t.Err)
}

// Transaction is the base struct for transactions with specific
// transaction handlers should embed
type Transaction struct {
	ongoing bool
	Starter starter
	Stopper stopper
	Aborter aborter
}

// Open is the handler for opening a transaction
func (t *Transaction) Open(ctx context.Context) error {
	if t.ongoing {
		return nil
	}

	var (
		err      error
		attempts = t.Attempts()
	)

	for attempts > 0 {
		err := t.Starter.Start(ctx)

		// We break and return if no error returned (transaction opened ok),
		// or the error is a context cancel/deadline related one. Any other
		// error implies trying again to open the transaction.
		if err == nil ||
			errors.Is(err, context.Canceled) ||
			errors.Is(err, context.DeadlineExceeded) {
			// set ongoing true only if no error was returned
			t.ongoing = (err == nil)
			break
		}

		attempts--

		// TODO: communicate the attempts?
		// Wait 10 seconds (interruptible) between transaction attempts
		select {
		case <-time.After(time.Second * time.Duration(10)):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return err
}

// Close will cleanly shut down the transaction
func (t *Transaction) Close(ctx context.Context) error {
	if !t.ongoing {
		return nil
	}

	t.ongoing = false
	return t.Stopper.Stop(ctx)
}

// Abort will kill the ongoing transaction
func (t *Transaction) Abort(ctx context.Context) error {
	if !t.ongoing {
		return nil
	}
	return t.Aborter.Kill(ctx)
}

// Start should be implemented by embedding transactions.
// It is called by Open.
func (t *Transaction) Start(ctx context.Context) error {
	return nil
}

// Stop should be implemented by embedding transactions.
// It is called by Close.
func (t *Transaction) Stop(ctx context.Context) error {
	return nil
}

// Attempts gets the default number of attempts to open a transaction
func (t *Transaction) Attempts() int {
	// TODO: make configurable
	return 3
}
