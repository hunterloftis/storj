package relay

import "math/rand"

// Secrets is a generator that provides unique secret strings in the form "first-second-third."
type Secrets struct {
	rng *rand.Rand
}

// NewSecrets returns a new secret generator based on the provided random-number generator.
//
// If `rng` is deterministic, the secret strings will be as well, so be sure to
// seed your generator with something pseudo-random like the current time in order to
// generate unique secrets.
func NewSecrets(rng *rand.Rand) Secrets {
	return Secrets{rng: rng}
}

// String returns the next random secret from the generator.
func (s Secrets) String() string {
	n := len(words)
	a, b, c := s.rng.Intn(n), s.rng.Intn(n), s.rng.Intn(n)
	return words[a] + "-" + words[b] + "-" + words[c]
}
