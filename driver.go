package lazydsn

import (
	"context"
	"database/sql"
	"database/sql/driver"
)

// Driver is not a database driver by itself, but rather a wrapper on top of
// any driver that can be used with Go's database/sql. It adds the capability
// to resolve DSNs at the moment that each individual connection in the pool is
// opened. Contrast with the typical database/sql behavior, where every new
// connection in the pool is created under the same DSN, set at the time the
// database was sql.Open'ed.
type Driver struct {
	driver.Driver
	dsnp FullDSNProvider
}

// New creates a new driver with the given inner driver d and DSN provider.
// Note that d can be a driver for any database; a Driver is database agnostic.
// This does NOT register the driver with database/sql. See Register. This
// function is provided so that other packages are able to create a properly
// initialized driver, in case they want to extend it (just like we're doing
// here with other drivers!)
func New(d driver.Driver, dsnp DSNProvider) *Driver {
	fdsnp, ok := dsnp.(FullDSNProvider)

	if !ok {
		fdsnp = fullProvider{
			DSNProvider: dsnp,
		}
	}

	return &Driver{
		Driver: d,
		dsnp:   fdsnp,
	}
}

// Register creates and registers the driver under the provided alias, with the
// given inner driver and DSN provider. Applications don't typically register
// drivers directly, relying on implicit registration via package import and
// drivers' init functions. However, this driver requires parameters (the DSN
// provider), that applications have to provide for this driver to be
// meaningful at all. It's a good practice to register this as close to the
// most basic packages in your application as possible, to separate business
// code from the intricacies of dealing with database drivers.
func Register(alias string, d driver.Driver, dsnp DSNProvider) {
	sql.Register(alias, New(d, dsnp))
}

// Open opens a database connection and returns the latter as a driver.Conn
// type. This is part of the driver.Driver interface. The DSN provided to this
// driver doesn't need to follow the format imposed by the underlying driver.
// In fact, it can be completely different; no restrictions are enforced by
// this package. The translation from the DSN provided by database/sql to this
// function and the one needed by the inner driver is entirely done by the
// DSNProvider assigned to this driver.
func (d *Driver) Open(dsn string) (driver.Conn, error) {
	innerDSN, err := d.dsnp.FetchDSN(dsn)

	if err != nil {
		return nil, err
	}

	return d.Driver.Open(innerDSN)
}

// dsnConnector is a basic connector for an inner driver that does not
// implement the driver.DriverContext interface, meaning that its Open method
// must be called every time that a new connection is required.
type dsnConnector struct {
	masterDSN string
	driver    *Driver
}

// Connect opens a new connection by calling the Open method in this driver.
func (c *dsnConnector) Connect(_ context.Context) (driver.Conn, error) {
	return c.driver.Open(c.masterDSN)
}

// Driver returns the driver for the connector.
func (c *dsnConnector) Driver() driver.Driver {
	return c.driver
}

// dsnConnector implements the driver.Connector interface.
var _ driver.Connector = &dsnConnector{}

// nativeConnector is a connector for inner drivers that implement the
// driver.DriverContext interface. We keep both a master DSN (as given to
// Driver) and the last known inner DSN, as returned from the DSN provider.
// That helps us renew the inner driver's connector only when the inner DSN
// changes.
type nativeConnector struct {
	masterDSN string
	innerDSN  string
	connector driver.Connector
	driver    *Driver
}

// Connect opens a new connection by using the inner driver's connector type.
// The inner DSN is always fetched and check against the one that the connector
// was created for. A new connector is created every time a change is detected.
func (c *nativeConnector) Connect(ctx context.Context) (driver.Conn, error) {
	innerDSN, err := c.driver.dsnp.FetchDSNWithContext(ctx, c.masterDSN)

	if err != nil {
		return nil, err
	}

	if innerDSN != c.innerDSN {
		// Configuration changed; we need a new connector.
		conn, err := c.driver.Driver.(driver.DriverContext).OpenConnector(innerDSN)

		if err != nil {
			return nil, err
		}

		c.connector = conn
		c.innerDSN = innerDSN
	}

	return c.connector.Connect(ctx)
}

// Driver returns the driver for the connector.
func (c *nativeConnector) Driver() driver.Driver {
	return c.driver
}

// nativeConnector implements the driver.Connector interface.
var _ driver.Connector = &nativeConnector{}

// OpenConnector returns a driver.Connector that can be used to open
// connections to the database without having the inner driver parsing the DSN
// repeatedly. That's, of course, as long as the inner driver implements the
// driver.DriverContext interface. If not, the resulting connector will simply
// be wrapping the Open method.
func (d *Driver) OpenConnector(dsn string) (driver.Connector, error) {
	if driverCtx, ok := d.Driver.(driver.DriverContext); ok {
		innerDSN, err := d.dsnp.FetchDSN(dsn)

		if err != nil {
			return nil, err
		}

		connector, err := driverCtx.OpenConnector(innerDSN)

		if err != nil {
			return nil, err
		}

		return &nativeConnector{
			masterDSN: dsn,
			innerDSN:  innerDSN,
			connector: connector,
			driver:    d,
		}, nil
	}

	return &dsnConnector{
		masterDSN: dsn,
		driver:    d,
	}, nil
}
