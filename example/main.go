package main

import (
	"database/sql"
	"fmt"
	conn "github.com/progbits/pg-secrets-driver"
	"log"
	"net/url"
)

// TestCredentialsProvider is a credentials' provider that returns data source
// names templated from a fixed list of potential passwords.
type TestCredentialsProvider struct {
	count     int
	passwords []string
}

// GetDataSourceName returns a data source name configured with the nth
// password in the list, where n is the number of times GetDataSourceName has
// been called. E.g. the first call to GetDataSourceName returns a data source
// name templated with the 0th password, the second call to GetDataSourceName
// returns a data source name templated with the 1st password, and so on.
func (p *TestCredentialsProvider) GetDataSourceName() (string, error) {
	withPassword := func(password string) string {
		dsn, _ := url.Parse("postgresql://localhost/postgres")
		query := dsn.Query()
		query.Add("user", "postgres")
		query.Add("password", password)
		query.Add("sslmode", "disable")
		dsn.RawQuery = query.Encode()
		return dsn.String()
	}

	password := p.passwords[p.count]
	p.count++

	fmt.Printf("Returning data source name with password: %s\n", password)

	return withPassword(password), nil
}

// Retries returns the number of potential passwords.
func (p *TestCredentialsProvider) Retries() int {
	return len(p.passwords)
}

func main() {
	passwords := []string{"foo", "bar", "baz", "password", "pa$$w0rd", "wrong-password"}
	provider := &TestCredentialsProvider{
		passwords: passwords,
	}
	connector := conn.NewPgSecretsConnector(provider)

	db := sql.OpenDB(connector)
	_, err := db.Exec("SELECT 1")
	if err != nil {
		log.Print(err)
	}
}
