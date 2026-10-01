package telemetry

import "errors"

// errNotImplemented is returned by stubs in this package that need to signal "not implemented"
// through an error return value instead of a zero value or a no-op (see each function's doc
// comment for which applies).
var errNotImplemented = errors.New("not implemented")
