package mediaid

import (
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"
)

var (
	entropyOnce sync.Once
	entropy     *ulid.MonotonicEntropy
)

func newEntropy() *ulid.MonotonicEntropy {
	entropyOnce.Do(func() {
		source := rand.NewSource(time.Now().UnixNano())
		entropy = ulid.Monotonic(rand.New(source), 0)
	})
	return entropy
}

// New returns a jan_* ULID string.
func New() string {
	id := ulid.MustNew(ulid.Timestamp(time.Now()), newEntropy())
	return "jan_" + strings.ToLower(id.String())
}

// IsValid reports whether the string is a jan_* ULID.
func IsValid(value string) bool {
	if !strings.HasPrefix(value, "jan_") {
		return false
	}
	_, err := Parse(value)
	return err == nil
}

// Parse strips the jan_ prefix and returns the ULID.
func Parse(value string) (ulid.ULID, error) {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "jan_")
	value = strings.TrimPrefix(value, "JAN_")
	return ulid.Parse(value)
}
