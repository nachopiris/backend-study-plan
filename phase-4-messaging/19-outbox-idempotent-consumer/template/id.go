package outbox

import (
	"crypto/rand"
	"encoding/hex"
)

// newID returns a short random identifier prefixed with prefix (e.g.
// "order_a3f1...", "msg_9c02..."). It is GIVEN: you do not need to implement
// ID generation yourself, this practice is about the outbox and idempotency
// mechanisms, not about ID schemes.
func newID(prefix string) string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err) // crypto/rand failing means the OS entropy source is broken
	}
	return prefix + "_" + hex.EncodeToString(b[:])
}
