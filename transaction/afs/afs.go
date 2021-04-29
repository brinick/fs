package afs

import (
	"context"

	"github.com/brinick/fs/transaction"
	"github.com/brinick/logging"
)

// NewTransaction will create a transaction object and call
// its open() method. The transaction Close() method should
// be deferred immediately after calling this, assuming
// no error was returned.
func NewTransaction(opts *Opts, log logging.Logger) *Transaction {
	t := Transaction{
		attempts: opts.MaxTransactionAttempts,
	}

	t.Transaction.Starter = &t
	t.Transaction.Stopper = &t
	return &t
}

// Opts configures the transaction
type Opts struct {
	// User with the necessary rights to install
	SudoUser string `json:"sudo_user"`

	// How many times we try to open our own AFS transaction
	MaxTransactionAttempts int `json:"max_transaction_open_attempts"`
}

// Transaction represents an AFS transaction
type Transaction struct {
	transaction.Transaction
	attempts int
}

// Kill will halt the ongoing transaction forcefully
// exiting without publishing
func (t *Transaction) Kill(ctx context.Context) error {
	return nil
}
