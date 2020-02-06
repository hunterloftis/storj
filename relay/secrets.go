package relay

import "math/rand"

type Secrets struct {
	rng *rand.Rand
}

func NewSecrets(rng *rand.Rand) Secrets {
	return Secrets{rng: rng}
}

func (s Secrets) String() string {
	n := len(words)
	a, b, c := s.rng.Intn(n), s.rng.Intn(n), s.rng.Intn(n)
	return words[a] + "-" + words[b] + "-" + words[c]
}
