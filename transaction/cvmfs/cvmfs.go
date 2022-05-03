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

	// RootDir is the directory on which the transaction is opened
	// If empty string, open on repository root.
	RootDir string `json:"root_dir"`

	// Machine with rights to contact the CVMFS gateway node
	ReleaseManager string `json:"release_manager"`

	// How many times we try to open the CVMFS transaction before aborting
	MaxOpenAttempts int `json:"max_open_attempts"`

	// How many times we try to publish the CVMFS transaction before aborting
	MaxPublishAttempts int `json:"max_publish_attempts"`

	// Seconds to wait between each attempt to publish
	PublishAttemptsWait int `json:"publish_attempts_wait"`
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
		Repo:                opts.NightlyRepo,
		Binary:              opts.Binary,
		Node:                opts.ReleaseManager,
		Root:                opts.RootDir,
		openAttempts:        opts.MaxOpenAttempts,
		publishAttempts:     opts.MaxPublishAttempts,
		publishAttemptsWait: opts.PublishAttemptsWait,
		catalogDirs:         nestedCatalogDirs,
	}

	t.Transaction.Starter = &t
	t.Transaction.Stopper = &t
	return &t
}

// Transaction represents a CVMFS transaction
type Transaction struct {
	transaction.Transaction
	Binary              string
	Repo                string
	Node                string
	Root                string
	log                 logging.Logger
	openAttempts        int
	publishAttempts     int
	publishAttemptsWait int
	catalogDirs         []string
}

// OpenAttempts provides the number of tries allowed for opening the transaction
func (t *Transaction) OpenAttempts() int {
	return t.openAttempts
}

// PublishAttempts provides the number of tries allowed for publishing the transaction
func (t *Transaction) PublishAttempts() int {
	return t.publishAttempts
}

// PublishAttemptsWait provides the seconds to wait between publish attempts
func (t *Transaction) PublishAttemptsWait() int {
	return t.publishAttemptsWait
}

// Start will open a new transaction. If one is already ongoing on
// this node, it will return an error
func (t *Transaction) Start(ctx context.Context) error {
	return transaction.OpenError{Err: t.execCmd(ctx, "transaction")}
}

// Stop will exit the transaction after publishing
func (t *Transaction) Stop(ctx context.Context) error {
	// TODO: should we abort publish if we cannot create catalogs? Probably not.
	createNestedCatalogs(t.catalogDirs...)
	return transaction.CloseError{Err: t.execCmd(ctx, "publish")}
}

// Kill will halt the ongoing transaction forcefully
// exiting without publishing
func (t *Transaction) Kill(ctx context.Context) error {
	return transaction.AbortError{Err: t.execCmd(ctx, "abort -f")}
}

func (t *Transaction) execCmd(ctx context.Context, cmd string) error {
	path, err := t.relPath()
	if err != nil {
		return err
	}
	fullCmd := fmt.Sprintf("%s %s %s", t.Binary, cmd, path)
	res := shell.Run(fullCmd, shell.Context(ctx))
	t.log.InfoL(res.Stdout().Lines())
	t.log.ErrorL(res.Stderr().Lines())
	return res.Err()
}

// relPath returns the path below the repo root
func (t *Transaction) relPath() (string, error) {
	if t.Root == "" {
		return t.Repo, nil
	}
	prefix := fmt.Sprintf("/cvmfs/%s/", t.Repo)
	path, err := filepath.Rel(prefix, t.Root)
	if err != nil {
		err = fmt.Errorf("Unable to calculate root path: %v", err)
	}

	return path, err
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
