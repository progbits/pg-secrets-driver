package connector

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"net"
	"net/url"
	"strings"
	"testing"
)

type PostgresContainer struct {
	testcontainers.Container
	Host string
	Port string
}

func NewPostgresContainer(ctx context.Context) (*PostgresContainer, error) {
	req := testcontainers.ContainerRequest{
		Image:        "postgres:13.7",
		ExposedPorts: []string{"5432/tcp"},
		Env:          map[string]string{"POSTGRES_PASSWORD": "pa$$w0rd"},
		WaitingFor:   wait.ForListeningPort("5432/tcp"),
	}
	container, err := testcontainers.GenericContainer(
		ctx,
		testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		})
	if err != nil {
		return nil, err
	}

	mappedPort, err := container.MappedPort(ctx, "5432")
	if err != nil {
		return nil, err
	}

	hostIP, err := container.Host(ctx)
	if err != nil {
		return nil, err
	}

	postgresContainer := PostgresContainer{
		Container: container,
		Host:      hostIP,
		Port:      mappedPort.Port()}
	return &postgresContainer, nil
}

func (c *PostgresContainer) Shutdown() {
	_ = c.Terminate(context.Background())
}

type TestCredentialsProvider struct {
	host      string
	port      string
	passwords []string
	count     int
}

func (p *TestCredentialsProvider) GetDataSourceName() (string, error) {
	withPassword := func(password string) string {
		rawUrl := fmt.Sprintf("postgresql://%s:%s/postgres", p.host, p.port)
		dsn, _ := url.Parse(rawUrl)
		query := dsn.Query()
		query.Add("user", "postgres")
		query.Add("password", password)
		query.Add("sslmode", "disable")
		dsn.RawQuery = query.Encode()
		return dsn.String()
	}

	password := p.passwords[p.count]
	p.count++

	return withPassword(password), nil
}

func (p *TestCredentialsProvider) Retries() int {
	return len(p.passwords)
}

func TestConnector(t *testing.T) {
	container, err := NewPostgresContainer(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer container.Shutdown()

	type TestCase struct {
		provider    CredentialsProvider
		expectedErr error
	}
	testCases := []TestCase{
		{
			provider: &TestCredentialsProvider{
				host:      container.Host,
				port:      container.Port,
				passwords: []string{"pa$$w0rd"},
			},
			expectedErr: nil,
		},
		{
			provider: &TestCredentialsProvider{
				host:      container.Host,
				port:      container.Port,
				passwords: []string{"wrong-password"},
			},
			expectedErr: &pq.Error{
				Code:    "28P01",
				Message: "password authentication failed for user \"postgres\"",
			},
		},
		{
			provider: &TestCredentialsProvider{
				host:      container.Host,
				port:      container.Port,
				passwords: []string{"wrong-password-0", "wrong-password-1"},
			},
			expectedErr: &pq.Error{
				Code:    "28P01",
				Message: "password authentication failed for user \"postgres\"",
			},
		},
		{
			provider: &TestCredentialsProvider{
				host:      "badhost",
				port:      container.Port,
				passwords: []string{"pa$$word"},
			},
			expectedErr: &net.OpError{
				Op:  "dial",
				Net: "tcp",
				Err: &net.DNSError{
					Err:        "Temporary failure in name resolution",
					Name:       "badhost",
					IsNotFound: true,
				},
			},
		},
		{
			provider: &TestCredentialsProvider{
				host:      container.Host,
				port:      container.Port,
				passwords: []string{"wrong-password-0", "wrong-password-1", "pa$$w0rd"},
			},
			expectedErr: nil,
		},
	}

	for _, c := range testCases {
		connector := NewPgSecretsConnector(c.provider)
		db := sql.OpenDB(connector)
		_, err = db.Exec("SELECT 1")
		if err != c.expectedErr {
			if !strings.Contains(err.Error(), c.expectedErr.Error()) {
				t.Fatal("unexpected error:", err)
			}
		}
	}
}
