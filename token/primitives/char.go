package primitives

import (
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/zimmski/tavor/rand"
	"github.com/zimmski/tavor/token"
)

var characterClassEscapes = map[rune][]rune{
	'd': []rune{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9'},
	's': []rune{'\t', '\n', '\f', '\r'},
	'w': []rune{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'N', 'M', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z', 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'n', 'm', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z', '_'},
}

// CharacterClass implements a char token which holds a pattern of characters and character classes
// Each character class characters is added to the set of characters of the token. Every permutations chooses one character out of the available set of characters as the current value of the token.
type CharacterClass struct {
	chars       []rune
	charsLookup map[rune]struct{}

	pattern string

	value rune
}

// NewCharacterClass returns a new instance of a CharacterClass token
func NewCharacterClass(pattern string) *CharacterClass {
	if pattern == "" {
		panic("pattern is empty")
	}

	var chars []rune
	charsLookup := make(map[rune]struct{})
	var first rune

	runes := strings.NewReader(pattern)

	add := func(c rune) {
		if _, ok := charsLookup[c]; !ok {
			if len(chars) == 0 {
				first = c
			}

			chars = append(chars, c)
			charsLookup[c] = struct{}{}
		}
	}

	c, _, err := runes.ReadRune()

	for err != io.EOF {
		if unicode.IsDigit(c) || unicode.IsLetter(c) || unicode.IsSpace(c) {
			add(c)
		} else {
			switch c {
			case '\\':
				c, _, err = runes.ReadRune()
				if err == io.EOF {
					panic(fmt.Sprintf("early EOF for escaped character"))
				}

				esc, ok := characterClassEscapes[c]
				if !ok {
					panic(fmt.Sprintf("Unknown escape character %q", c))
				}

				for _, v := range esc {
					add(v)
				}
			default:
				panic(fmt.Sprintf("Unknown character %q", c))
			}
		}

		c, _, err = runes.ReadRune()
	}

	return &CharacterClass{
		chars:       chars,
		charsLookup: charsLookup,

		pattern: pattern,

		value: first,
	}
}

// Clone returns a copy of the token and all its children
func (c *CharacterClass) Clone() token.Token {
	chars := make([]rune, len(c.chars))

	copy(chars, c.chars)

	charsLookup := make(map[rune]struct{})

	for k := range c.charsLookup {
		charsLookup[k] = struct{}{}
	}

	return &CharacterClass{
		chars:       chars,
		charsLookup: charsLookup,

		pattern: c.pattern,

		value: c.value,
	}
}

// Fuzz fuzzes this token using the random generator by choosing one of the possible permutations for this token
func (c *CharacterClass) Fuzz(r rand.Rand) {
	i := uint(r.Intn(len(c.chars)))

	c.permutation(i)
}

// FuzzAll calls Fuzz for this token and then FuzzAll for all children of this token
func (c *CharacterClass) FuzzAll(r rand.Rand) {
	c.Fuzz(r)
}

// Parse tries to parse the token beginning from the current position in the parser data.
// If the parsing is successful the error argument is nil and the next current position after the token is returned.
func (c *CharacterClass) Parse(pars *token.InternalParser, cur int) (int, []error) {
	// TODO FIXME NOW we can see the need to put pars.Data into a reader... since we cannot do a readRune here!
	v := rune(pars.Data[cur])

	if _, ok := c.charsLookup[v]; !ok {
		return cur, []error{&token.ParserError{
			Message: fmt.Sprintf("expected %q but got %q", c.charsLookup, v),
			Type:    token.ParseErrorUnexpectedData,
		}}
	}

	c.value = v

	return cur + 1, nil
}

func (c *CharacterClass) permutation(i uint) {
	c.value = c.chars[i]
}

// Permutation sets a specific permutation for this token
func (c *CharacterClass) Permutation(i uint) error {
	permutations := c.Permutations()

	if i < 1 || i > permutations {
		return &token.PermutationError{
			Type: token.PermutationErrorIndexOutOfBound,
		}
	}

	c.permutation(i - 1)

	return nil
}

// Permutations returns the number of permutations for this token
func (c *CharacterClass) Permutations() uint {
	return uint(len(c.chars))
}

// PermutationsAll returns the number of all possible permutations for this token including its children
func (c *CharacterClass) PermutationsAll() uint {
	return c.Permutations()
}

func (c *CharacterClass) String() string {
	return string(c.value)
}
