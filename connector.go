package connector

import (
	"context"
	"database/sql/driver"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"sync"
)

const errInvalidPassword = "28P01"

// CredentialsProvider implementations are a source of Postgres data
// source names, where a data source name is a valid Postgres connection
// string (https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING).
//
// The GetDataSourceName method is called each time a new database connection
// is required, e.g. each time the PgSecretsConnector.Connect method is called.
// If the PgSecretsConnector.Connect method fails with `invalid_password`, the
// GetDataSourceName method is called a further n times, where n is determined
// by the result of the Retries method. This presents an opportunity to attempt
// to connect with different sets of credentials, e.g. rotated_current and
// rotated_previous.
type CredentialsProvider interface {
	GetDataSourceName() (string, error)
	Retries() int
}

// PgSecretsConnector is the main library type and implements the
// driver.Connector interface. This means it can be passed to the sql.OpenDB
// method as a source of database connections. The connection string used to
// open a new database connection is sourced from the configured
// CredentialsProvider implementation.
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
