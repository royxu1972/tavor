package logicals

import (
	"github.com/zimmski/tavor/rand"
	"github.com/zimmski/tavor/token"
)

type One struct {
	tokens []token.Token
	value  token.Token
}

func NewOne(tokens ...token.Token) *One {
	if len(tokens) == 0 {
		panic("at least one token needed")
	}

	return &One{
		tokens: tokens,
		value:  tokens[0],
	}
}

func (o *One) Fuzz(r rand.Rand) {
	i := r.Intn(len(o.tokens))

	o.value = o.tokens[i]

	o.value.Fuzz(r)
}

func (o *One) String() string {
	return o.value.String()
}
