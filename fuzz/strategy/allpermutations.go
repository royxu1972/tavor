package strategy

import (
	"fmt"

	"github.com/zimmski/container/list/linkedlist"

	"github.com/zimmski/tavor"
	"github.com/zimmski/tavor/rand"
	"github.com/zimmski/tavor/token"
	"github.com/zimmski/tavor/token/lists"
)

type allPermutationsLevel struct {
	token           token.Token
	permutation     int
	maxPermutations int
}

type AllPermutationsStrategy struct {
	root token.Token

	resetedLookup map[token.Token]int
}

func NewAllPermutationsStrategy(tok token.Token) *AllPermutationsStrategy {
	s := &AllPermutationsStrategy{
		root: tok,

		resetedLookup: make(map[token.Token]int),
	}

	return s
}

func init() {
	Register("AllPermutations", func(tok token.Token) Strategy {
		return NewAllPermutationsStrategy(tok)
	})
}

func (s *AllPermutationsStrategy) getLevel(root token.Token, fromChilds bool) []allPermutationsLevel {
	var level []allPermutationsLevel
	var queue = linkedlist.New()

	if fromChilds {
		switch t := root.(type) {
		case token.ForwardToken:
			queue.Push(t.Get())
		case lists.List:
			l := t.Len()

			for i := 0; i < l; i++ {
				c, _ := t.Get(i)
				queue.Push(c)
			}
		}
	} else {
		queue.Push(root)
	}

	for !queue.Empty() {
		v, _ := queue.Shift()
		tok, _ := v.(token.Token)

		s.setTokenPermutation(tok, 1)

		level = append(level, allPermutationsLevel{
			token:           tok,
			permutation:     1,
			maxPermutations: tok.Permutations(),
		})
	}

	return level
}

func (s *AllPermutationsStrategy) Fuzz(r rand.Rand) chan struct{} {
	continueFuzzing := make(chan struct{})

	s.resetedLookup = make(map[token.Token]int)

	go func() {
		if tavor.DEBUG {
			fmt.Println("Start all permutations routine")
		}

		level := s.getLevel(s.root, false)

		if len(level) != 0 {
			if tavor.DEBUG {
				fmt.Println("Start fuzzing step")
			}

			if !s.fuzz(continueFuzzing, level) {
				return
			}
		}

		s.resetResetTokens()

		if tavor.DEBUG {
			fmt.Println("Done with fuzzing step")
		}

		// done with the last fuzzing step
		continueFuzzing <- struct{}{}

		if tavor.DEBUG {
			fmt.Println("Finished fuzzing. Wait till the outside is ready to close.")
		}

		if _, ok := <-continueFuzzing; ok {
			if tavor.DEBUG {
				fmt.Println("Close fuzzing channel")
			}

			close(continueFuzzing)
		}
	}()

	return continueFuzzing
}

func (s *AllPermutationsStrategy) resetResetTokens() {
	var queue = linkedlist.New()

	queue.Push(s.root)

	for !queue.Empty() {
		v, _ := queue.Shift()

		switch tok := v.(type) {
		case token.ResetToken:
			if tavor.DEBUG {
				fmt.Printf("Reset %#v(%p)\n", tok, tok)
			}

			tok.Reset()
		}

		if t, ok := v.(token.OptionalToken); !ok || !t.IsOptional() {
			switch tok := v.(type) {
			case token.ForwardToken:
				queue.Push(tok.Get())
			case lists.List:
				for i := 0; i < tok.Len(); i++ {
					c, _ := tok.Get(i)
					queue.Push(c)
				}
			}
		}
	}
}

func (s *AllPermutationsStrategy) setTokenPermutation(tok token.Token, permutation int) {
	if per, ok := s.resetedLookup[tok]; ok && per == permutation {
		// Permutation already set in this step
	} else {
		tok.Permutation(permutation)

		s.resetedLookup[tok] = permutation
	}
}

func (s *AllPermutationsStrategy) fuzz(continueFuzzing chan struct{}, level []allPermutationsLevel) bool {
	if tavor.DEBUG {
		fmt.Printf("Fuzzing level %d->%#v\n", len(level), level)
	}

	last := len(level) - 1

STEP:
	for {
		for i := range level {
			if level[i].permutation > level[i].maxPermutations {
				if i <= last {
					if tavor.DEBUG {
						fmt.Printf("Max reached redo everything <= %d and increment next\n", i)
					}

					level[i+1].permutation++
					s.setTokenPermutation(level[i+1].token, level[i+1].permutation)
					s.getLevel(level[i+1].token, true) // set all children to permutation 1
				}

				for k := 0; k <= i; k++ {
					level[k].permutation = 1
					s.setTokenPermutation(level[k].token, 1)
					s.getLevel(level[k].token, true) // set all children to permutation 1
				}

				continue STEP
			}

			if tavor.DEBUG {
				fmt.Printf("Permute %d->%#v\n", i, level[i])
			}

			s.setTokenPermutation(level[i].token, level[i].permutation)

			if t, ok := level[i].token.(token.OptionalToken); !ok || !t.IsOptional() || level[i].permutation != 1 {
				childs := s.getLevel(level[i].token, true) // set all children to permutation 1

				if len(childs) != 0 {
					if !s.fuzz(continueFuzzing, childs) {
						return false
					}
				}
			}

			if i == 0 {
				level[i].permutation++
			}
		}

		if level[0].permutation > level[0].maxPermutations {
			found := false
			for i := 1; i < len(level); i++ {
				if level[i].permutation < level[i].maxPermutations {
					found = true

					break
				}
			}
			if !found {
				if tavor.DEBUG {
					fmt.Println("Done with fuzzing this level")
				}

				break STEP
			}
		}

		s.resetResetTokens()

		if tavor.DEBUG {
			fmt.Println("Done with fuzzing step")
		}

		// done with this fuzzing step
		continueFuzzing <- struct{}{}

		// wait until we are allowed to continue
		if _, ok := <-continueFuzzing; !ok {
			if tavor.DEBUG {
				fmt.Println("Fuzzing channel closed from outside")
			}

			return false
		}

		if tavor.DEBUG {
			fmt.Println("Start fuzzing step")
		}

		s.resetedLookup = make(map[token.Token]int)
	}

	return true
}