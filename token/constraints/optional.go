package constraints

import (
	"github.com/zimmski/tavor/rand"
	"github.com/zimmski/tavor/token"
)

type Optional struct {
	token token.Token
	value bool

	reducing              bool
	reducingOriginalValue bool
}

func NewOptional(tok token.Token) *Optional {
	return &Optional{
		token: tok,
		value: false,
	}
}

// Token interface methods

// Clone returns a copy of the token and all its children
func (c *Optional) Clone() token.Token {
	return &Optional{
		token: c.token.Clone(),
		value: c.value,
	}
}

func (c *Optional) Fuzz(r rand.Rand) {
	c.permutation(uint(r.Int() % 2))
}

func (c *Optional) FuzzAll(r rand.Rand) {
	c.Fuzz(r)

	if !c.value {
		c.token.FuzzAll(r)
	}
}

func (c *Optional) Parse(pars *token.InternalParser, cur int) (int, []error) {
	nex, errs := c.token.Parse(pars, cur)

	if len(errs) == 0 {
		c.value = false

		return nex, nil
	}

	c.value = true

	return cur, nil
}

func (c *Optional) permutation(i uint) {
	c.value = i == 0
}

func (c *Optional) Permutation(i uint) error {
	permutations := c.Permutations()

	if i < 1 || i > permutations {
		return &token.PermutationError{
			Type: token.PermutationErrorIndexOutOfBound,
		}
	}

	c.permutation(i - 1)

	return nil
}

func (c *Optional) Permutations() uint {
	return 2
}

func (c *Optional) PermutationsAll() uint {
	return 1 + c.token.PermutationsAll()
}

func (c *Optional) String() string {
	if c.value {
		return ""
	}

	return c.token.String()
}

// ForwardToken interface methods

func (c *Optional) Get() token.Token {
	if c.value {
		return nil
	}

	return c.token
}

func (c *Optional) InternalGet() token.Token {
	return c.token
}

func (c *Optional) InternalLogicalRemove(tok token.Token) token.Token {
	if c.token == tok {
		return nil
	}

	return c
}

func (c *Optional) InternalReplace(oldToken, newToken token.Token) {
	if c.token == oldToken {
		c.token = newToken
	}
}

// OptionalToken interface methods

func (c *Optional) IsOptional() bool { return true }
func (c *Optional) Activate()        { c.value = false }
func (c *Optional) Deactivate()      { c.value = true }

// ReduceToken interface methods

func (c *Optional) Reduce(i uint) error {
	reduces := c.Permutations()

	if reduces == 0 || i < 1 || i > reduces {
		return &token.ReduceError{
			Type: token.ReduceErrorIndexOutOfBound,
		}
	}

	if !c.reducing {
		c.reducing = true
		c.reducingOriginalValue = c.value
	}

	c.permutation(i - 1)

	return nil
}

func (c *Optional) Reduces() uint {
	if c.reducing || !c.value {
		return 2
	}

	return 0
}
