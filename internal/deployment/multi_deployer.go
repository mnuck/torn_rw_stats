package deployment

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

// dataDeployer is the method set MultiDeployer fans out to. It mirrors
// ports.Deployer without importing the ports package.
type dataDeployer interface {
	DeployData(data io.Reader, size int64, filename string) error
}

// MultiDeployer fans each deployment out to several deployers, e.g. the SSH
// server and R2 while migrating between them.
type MultiDeployer struct {
	deployers []dataDeployer
}

// NewMultiDeployer creates a deployer that writes to every given deployer
func NewMultiDeployer(deployers ...dataDeployer) *MultiDeployer {
	return &MultiDeployer{deployers: deployers}
}

// DeployData deploys the same data to every deployer. A failure in one does
// not stop the others; all failures are returned together.
func (m *MultiDeployer) DeployData(data io.Reader, size int64, filename string) error {
	// Each deployer consumes its reader, so buffer the data once and give
	// every deployer a fresh reader over it
	buf, err := io.ReadAll(data)
	if err != nil {
		return fmt.Errorf("failed to read data: %w", err)
	}

	var errs []error
	for _, d := range m.deployers {
		if err := d.DeployData(bytes.NewReader(buf), size, filename); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
