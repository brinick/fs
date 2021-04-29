package cvmfs

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/brinick/fs"
	"github.com/brinick/fs/transaction"
	"github.com/brinick/logging"
	"github.com/brinick/shell"
)

// Opts configures the CVMFS transaction
type Opts struct {
	// User with the necessary rights to install
	SudoUser string `json:"sudo_user"`

	// Path to the CVMFS server binary
	Binary string `json:"cvmfs_server_binary"`

	// Name of the nightly repo
	NightlyRepo string `json:"nightly_repo"`

	// Machine with rights to contact the CVMFS gateway node
	ReleaseManager string `json:"release_manager"`

	// How many times we try to open our own CVMFS transaction
	MaxTransactionAttempts int `json:"max_transaction_open_attempts"`
}

func shellWithContext(ctx context.Context, cmd string, args ...string) error {
	c := exec.CommandContext(ctx, cmd, args...)
	return c.Run()
}

var (
	// ErrTooManyAttempts is the error returned once the maximum number
	// of allowed open transaction attempts is reached
	ErrTooManyAttempts = fmt.Errorf("Too many attempts made to open transaction")
)

// NewTransaction will create a transaction object and call
// its open() method. The transaction Close() method should
// be deferred immediately after calling this, assuming
// no error was returned.
func NewTransaction(opts *Opts, log logging.Logger, nestedCatalogDirs ...string) *Transaction {
	t := Transaction{
		Repo:        opts.NightlyRepo,
		Binary:      opts.Binary,
		Node:        opts.ReleaseManager,
		attempts:    opts.MaxTransactionAttempts,
		catalogDirs: nestedCatalogDirs,
	}

	t.Transaction.Starter = &t
	t.Transaction.Stopper = &t
	return &t
}

// Transaction represents a CVMFS transaction
type Transaction struct {
	transaction.Transaction
	Binary      string
	Repo        string
	Node        string
	log         logging.Logger
	attempts    int
	catalogDirs []string
}

// Attempts provides the number of tries allowed for opening the transaction
func (t *Transaction) Attempts() int {
	return t.attempts
}

// Start will open a new transaction. If one is already ongoing on
// this node, it will return an error
func (t *Transaction) Start(ctx context.Context) error {
	cmd := fmt.Sprintf("%s transaction %s", t.Binary, t.Repo)
	res := shell.Run(cmd, shell.Context(ctx))
	t.log.InfoL(res.Stdout().Lines())
	t.log.ErrorL(res.Stderr().Lines())
	return res.Err()
}

// Stop will exit the transaction after publishing
func (t *Transaction) Stop(ctx context.Context) error {
	// TODO: should we abort publish if we cannot create catalogs? Probably not.
	createNestedCatalogs(t.catalogDirs...)
	cmd := fmt.Sprintf("%s publish %s", t.Binary, t.Repo)
	res := shell.Run(cmd, shell.Context(ctx))
	t.log.InfoL(res.Stdout().Lines())
	t.log.ErrorL(res.Stderr().Lines())
	return res.Err()
}

// Kill will halt the ongoing transaction forcefully
// exiting without publishing
func (t *Transaction) Kill(ctx context.Context) error {
	cmd := fmt.Sprintf("%s abort -f %s", t.Binary, t.Repo)
	res := shell.Run(cmd, shell.Context(ctx))
	t.log.InfoL(res.Stdout().Lines())
	t.log.ErrorL(res.Stderr().Lines())
	return res.Err()
}

func createNestedCatalogs(dirs ...string) error {
	for _, dir := range dirs {
		catalog := fs.NewFile(filepath.Join(dir, ".cvmfscatalog"))
		if err := catalog.Touch(true); err != nil {
			return err
		}
	}

	return nil
}
