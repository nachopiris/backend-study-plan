package isolation

import (
	"context"
	"errors"
)

// ErrLastDoctorOnCall is returned by GoOffCall when taking doctorName off
// call would leave zero doctors on call.
var ErrLastDoctorOnCall = errors.New("isolation: cannot go off call, no other doctor is on call")

// GoOffCall takes doctorName off call (on_call = false), unless doing so would
// leave nobody else on call, in which case it returns ErrLastDoctorOnCall and
// makes no change to the database.
//
// This is a write-skew scenario, not a lost-update one: two different rows
// are involved (each doctor updates only their own row), so a naive "SELECT
// count(*) ... then UPDATE my row" can pass the check for two doctors at the
// same time — each one sees the other still on call — and both updates
// commit, leaving zero doctors on call even though neither transaction did
// anything "wrong" in isolation.
//
// Implement this with a SERIALIZABLE transaction (retried up to maxRetries
// times on SQLSTATE 40001, same as IncrementSerializable): inside that single
// transaction, count how many OTHER doctors (doctor_name != doctorName) are
// currently on_call = true; if that count is zero, return ErrLastDoctorOnCall
// without updating anything; otherwise set doctorName's on_call to false and
// commit. The check and the write must be part of the same transaction for
// SERIALIZABLE to be able to detect the conflict between two concurrent
// callers.
func (s *Store) GoOffCall(ctx context.Context, doctorName string, maxRetries int) error {
	return errors.New("not implemented")
}
