Lazy DSN database driver for Go
===============================

[![GoDoc](https://godoc.org/github.com/gkristic/lazydsn?status.svg)](https://godoc.org/github.com/gkristic/lazydsn)
[![Go Report Card](https://goreportcard.com/badge/github.com/gkristic/lazydsn)](https://goreportcard.com/report/github.com/gkristic/lazydsn)

This package implements a database agnostic, `database/sql` style driver, with
delayed DSN evaluation. It's a wrapper for a real driver but, instead of using
the DSN as provided by Go's package, it relies on a provider to resolve the
DSN with every new connection attempt. This allows for use cases where, despite
connecting to the same (or functionally equivalent) database, applications are
forced to cope with ever rotating credentials, where the password or even also
the user change with time. Credentials rotation is typical in highly secured
environments and needs to be performed online, while applications are running.
This fully supports services such as AWS Secrets Manager.

A "simple" approach to cope with rotating credentials is detecting connection
errors and reopening the database. However, this can get tricky pretty fast.
Go's `database/sql` package keeps a pool of connections, recycling them as
needed on failures, timeout, etc. This means that connections can be attempted
at anytime, prompting the need to add instrumentation everywhere. This, in
turn, complicates the code, is error prone and more difficult to maintain.

This package proposes a different approach, moving instrumentation to the
driver side instead. By doing this, every new connection that is required in
the pool will be opened with updated credentials. If you use regular rotation,
you only need to configure your connections to have a bounded lifetime. See
[`sql.SetConnMaxLifetime`][conn-lifetime]. Your application may look like this:

```go
import (
	"database/sql"
	"time"

	"github.com/gkristic/lazydsn"
	"github.com/go-sql-driver/mysql"
)

const alias = "lazydsn:mysql"

func main() {
	lazydsn.Register(alias, &mysql.MySQLDriver{},
		lazydsn.DSNProviderFunc(func (dsn string) (string, error) {
			// Compute a new dsn; e.g., by using AWS Secrets Manager
			return dsn, nil
		}),
	)

	db := sql.Open(alias, "arn:...")
	db.SetConnMaxLifetime(time.Hour)
	// Keep working with db as usual
}
```

[conn-lifetime]: https://golang.org/pkg/database/sql/#DB.SetConnMaxLifetime
