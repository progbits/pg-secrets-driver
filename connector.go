package connector

import (
	"context"
	"database/sql/driver"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"sync"
)

// CredentialsProvider implementations return a valid Postgres data source
// name.
type CredentialsProvider interface {
	GetDataSourceName() (string, error)
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

	dataSourceName, err := c.provider.GetDataSourceName()
	if err != nil {
		return nil, err
	}

	conn, err := c.driver.Open(dataSourceName)
	if err != nil {
		if e, ok := err.(*pq.Error); ok {
			if e.Code == "28P01" {
				// Invalid password.
				dataSourceName, err := c.provider.GetDataSourceName()
				if err != nil {
					return nil, err
				}
				return c.driver.Open(dataSourceName)
			}
		}
		return nil, err
	}
	return conn, err
}

// Driver returns the underlying driver of the connector.
func (c *PgSecretsConnector) Driver() driver.Driver {
	return c.driver
}
