package connector

import (
	"context"
	"database/sql/driver"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"sync"
)

const errInvalidPassword = "28P01"

// CredentialsProvider implementations return a valid Postgres data source
// name.
type CredentialsProvider interface {
	GetDataSourceName() (string, error)
	Retries() int
}

// PgSecretsConnector implements the driver.Connector interface.
type PgSecretsConnector struct {
	mutex    sync.Mutex
	driver   driver.Driver
	provider CredentialsProvider
}

func NewPgSecretsConnector(provider CredentialsProvider) *PgSecretsConnector {
	return &PgSecretsConnector{
		driver:   pq.Driver{},
		provider: provider,
	}
}

func (c *PgSecretsConnector) Connect(ctx context.Context) (driver.Conn, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	var authErr error = nil
	for i := 0; i < c.provider.Retries(); i++ {
		dataSourceName, err := c.provider.GetDataSourceName()
		if err != nil {
			return nil, err
		}

		conn, err := c.driver.Open(dataSourceName)
		if err == nil {
			// Connected successfully.
			return conn, err
		}

		pqErr, ok := err.(*pq.Error)
		if !ok {
			// Not a Postgres error, don't retry.
			return nil, err
		}

		if pqErr.Code == errInvalidPassword {
			// Authentication error, maybe try again.
			authErr = err
			continue
		}

		// Not an authentication error, don't retry.
		return nil, err
	}
	return nil, authErr
}

// Driver returns the underlying driver of the connector.
func (c *PgSecretsConnector) Driver() driver.Driver {
	return c.driver
}
