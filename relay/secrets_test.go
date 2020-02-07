package relay

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func ExampleNewSecrets() {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	secrets := NewSecrets(rng)
	fmt.Println(secrets.String())
}

func TestSecretCollision(t *testing.T) {
	const n = 1000
	seen := make([]string, 0, n)

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	secrets := NewSecrets(rng)

	for i := 0; i < n; i++ {
		secret := secrets.String()
		for _, prev := range seen {
			if prev == secret {
				t.Errorf("duplicate secret: %v", prev)
			}
		}
		seen = append(seen, secret)
	}
}

func TestSecretConsistency(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	secrets := NewSecrets(rng)
	seq := []string{"fast-blue-began", "fire-type-here", "better-chance-glad"}

	for _, want := range seq {
		got := secrets.String()
		if got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}
