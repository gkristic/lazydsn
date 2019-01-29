package lazydsn

import (
	"context"
)

// A DSNProvider allows converting a DSN, as provided to this driver via
// database/sql, into a DSN suitable for using with the inner driver.
type DSNProvider interface {
	FetchDSN(string) (string, error)
}

// A FullDSNProvider acts like a DSNProvider, but provides an additional
// function to fetch the DSN using a context, thus allowing the chance to
// cancel the operation.
type FullDSNProvider interface {
	DSNProvider
	FetchDSNWithContext(context.Context, string) (string, error)
}

// fullProvider implements a FullDSNProvider. It's used as a wrapper, to
// augment types that do not provide context cancellation.
type fullProvider struct {
	DSNProvider
}

// FetchDSNWithContext for this type simply ignores the context, because the
// original type was determined to have no support for it.
func (p fullProvider) FetchDSNWithContext(_ context.Context, dsn string) (string, error) {
	return p.FetchDSN(dsn)
}

// DSNProviderFunc provides a convenient type so that applications don't have
// to declare specific types and methods with the only purpose of having a
// DSNProvider. This makes it possible to use an inline function literal
// instead.
type DSNProviderFunc func(string) (string, error)

// FetchDSN exercises the original function to resolve a DSN.
func (f DSNProviderFunc) FetchDSN(dsn string) (string, error) {
	return f(dsn)
}

// DSNProviderWCFunc provides a convenient type so that applications don't have
// to declare specific types and methods with the only purpose of having a
// DSNProvider. This makes it possible to use an inline function literal
// instead.
type DSNProviderWCFunc func(context.Context, string) (string, error)

// FetchDSN exercises the function providing an empty context.
func (f DSNProviderWCFunc) FetchDSN(dsn string) (string, error) {
	return f(context.Background(), dsn)
}

// FetchDSNWithContext exercises the original function, providing the context
// passed in to this function by the driver.
func (f DSNProviderWCFunc) FetchDSNWithContext(ctx context.Context, dsn string) (string, error) {
	return f(ctx, dsn)
}
