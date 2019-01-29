/*
Package lazydsn implements a database agnostic driver with delayed DSN
evaluation. This allows for use cases where, despite connecting to the same (or
functionally equivalent) database, applications are forced to cope with ever
rotating credentials, where the password or even also the user change with
time. Credentials rotation is typical in highly secured environments and needs
to be performed online, while applications are running. This fully supports
services such as AWS Secrets Manager.

Warning: Please make sure that you do NOT fundamentally change access when you
rotate credentials. Changing the password is fine. Switching to a different
user works, as long as both users have exactly equivalent access privileges
(grants and whatever applies to your database). The database/sql package
assumes a constant DSN after the database is opened. If access rights change,
then different connections in the pool may behave differently, which will most
likely lead to a debug nightmare, to say the least. Of course nothing prevents
you from changing other pieces in the DSN, like the database/schema that you
connect to, but that's more often than not a very bad idea. (To start with, all
of your queries would have to be fully qualified. So, don't do it unless you
know exactly what you're doing.)

Like any driver, you don't use this package directly. The only difference with
respecto to other drivers is that this is not automatically available simply by
importing the package. The reason is that this driver needs something else to
work with: a DSN provider. That's the place where you define the actual DSN
that you want to use, starting from the DSN that the database/sql package
provides (that matches what you give in sql.Open). You can use any type that
implements the DSNProvider interface. For extra convenience, a DSNProviderFunc
allows you to define a single function inline, instead of having to declare a
type and methods. Using it, your application may look like this:

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

Once you open the database with this driver and set the connection lifetime,
everything looks just as usual from the application's perspective. However,
behind the scenes, connections have a predefined expiration, and they are
renewed using the latest credentials available. Credentials rotation is thus
fully suported, but completely transparent.

If the type that you provide also implements FullDSNProvider, then a
cancellation context will be provided when available. Again, for convenience,
you can use a DSNProviderWCFunc to give your context-enabled function inline.
*/
package lazydsn
