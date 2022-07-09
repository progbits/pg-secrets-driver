# pg-secrets-driver

A Postgres database driver that supports credential rotation.

## Getting Started

The following example demonstrates a simple `CredentialsProvider`
implementation, configured with a fixed list of passwords. The implementation
cycles though the fixed list of passwords, trying each in turn, until it is able
to establish a connection successfully.

In actual usage, it is likely that a `CredentialsProvider` implementation would
source the dynamic components of the data source name (e.g. the password) from
either the environment of the process or a service such as  AWS SecretsManager
or HashiCorp Vault.

```go
package main

import (
	"database/sql"
	conn "github.com/progbits/pg-secrets-driver"
	"log"
	"net/url"
)

type TestCredentialsProvider struct {
	count     int
	passwords []string
}

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

func (p *TestCredentialsProvider) Retries() int {
	return len(p.passwords)
}

func main() {
	passwords := []string{"foo", "bar", "baz", "password", "pa$$w0rd", "p@ssw0rd"}
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
```

The example code above can be found in the
[example](https://github.com/progbits/pg-secrets-driver/tree/main/example)
directory.

To run the example, start a new Postgres instance with the superuser
password `pa$$w0rd`, the 5th value in the list held by `TestCredentialsProvider`:

```shell
docker run -e 'POSTGRES_PASSWORD=pa$$w0rd' -d -p 5432:5432 postgres:14.3
```

The example can then be run using `make`:

```shell
make run-example
```
