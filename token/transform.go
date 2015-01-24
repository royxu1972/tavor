package token

import (
	"github.com/zimmski/container/list/linkedlist"

	"github.com/zimmski/tavor"
	"github.com/zimmski/tavor/log"
)

// MinimizeTokens traverses the token graph and replaces unnecessary complicated constructs with their simpler form
// One good example is an All list token with one token which can be replaced by this one token. The minimize checks and operation is done by the token itself which has to implement the MinimizeToken interface, since it is not always predictable if a token with one child is doing something special,
func MinimizeTokens(root Token) Token {
	log.Debug("start minimizing")

	parents := make(map[Token]Token)
	queue := linkedlist.New()

	queue.Unshift(root)
	parents[root] = nil

	for !queue.Empty() {
		v, _ := queue.Shift()

		if tok, ok := v.(MinimizeToken); ok {
			r := tok.Minimize()
			if r != nil {
				p := parents[tok]

				switch pTok := p.(type) {
				case ForwardToken:
					pTok.InternalReplace(tok, r)
				case ListToken:
					pTok.InternalReplace(tok, r)
				}

				queue.Unshift(r)
				parents[r] = p

				continue
			}
		}

		switch tok := v.(type) {
		case ForwardToken:
			if v := tok.InternalGet(); v != nil {
				queue.Unshift(v)
				parents[v] = tok
			}
		case ListToken:
			for i := tok.InternalLen() - 1; i >= 0; i-- {
				c, _ := tok.InternalGet(i)

				queue.Unshift(c)
				parents[c] = tok
			}
		}
	}

	log.Debug("finished minimizing")

	return root
}

// UnrollPointers unrolls pointer tokens by copying their referenced graphs.
// Pointers that lead to themselves are unrolled at maximum tavor.MaxRepeat times.
func UnrollPointers(root Token) Token {
	type unrollToken struct {
		tok    Token
		parent *unrollToken
		counts map[Token]int
	}

	log.Debug("start unrolling pointers by cloning them")

	parents := make(map[Token]Token)

	originals := make(map[Token]Token)
	originalClones := make(map[Token]Token)

	queue := linkedlist.New()

	queue.Unshift(&unrollToken{
		tok:    root,
		parent: nil,
		counts: make(map[Token]int),
	})
	parents[root] = nil

	for !queue.Empty() {
		v, _ := queue.Shift()
		iTok, _ := v.(*unrollToken)

		switch t := iTok.tok.(type) {
		case PointerToken:
			child := t.InternalGet()
			if child == nil {
				log.Panicf("Child of (%p)%#v is nil", t, t)

				continue
			}

			replace := true

			if p, ok := child.(PointerToken); ok {
				checked := map[PointerToken]struct{}{
					p: struct{}{},
				}

				for {
					log.Debugf("Child (%p)%#v is a pointer lets go one further", p, p)

					child = p.InternalGet()
					if child == nil {
						log.Panicf("Child of (%p)%#v is nil", p, p)

						continue
					}

					p, ok = child.(PointerToken)
					if !ok {
						break
					}

					if _, found := checked[p]; found {
						log.Debugf("Endless pointer loop with (%p)%#v", p, p)

						replace = false

						break
					}

					checked[p] = struct{}{}
				}
			}

			var original Token
			counted := 0

			if replace {
				if o, found := originals[child]; found {
					log.Debugf("Found original (%p)%#v for child (%p)%#v", o, o, child, child)
					original = o
					counted = iTok.counts[original]

					if counted >= tavor.MaxRepeat {
						replace = false
					}
				} else {
					log.Debugf("Found no original for child (%p)%#v, must be new!", child, child)
					originals[child] = child
					original = child

					// we want to clone only original structures so we always clone the clone since the original could have been changed in the meantime
					originalClones[child] = child.Clone()
				}
			}

			if replace {
				log.Debugf("clone (%p)%#v with child (%p)%#v", t, t, child, child)

				c := originalClones[original].Clone()

				counts := make(map[Token]int)
				for k, v := range iTok.counts {
					counts[k] = v
				}

				counts[original] = counted + 1
				originals[c] = original

				log.Debugf("clone is (%p)%#v", c, c)

				if err := t.Set(c); err != nil {
					panic(err)
				}

				if iTok.parent != nil {
					log.Debugf("replace in (%p)%#v", iTok.parent.tok, iTok.parent.tok)

					switch tt := iTok.parent.tok.(type) {
					case ForwardToken:
						tt.InternalReplace(t, c)
					case ListToken:
						tt.InternalReplace(t, c)
					}
				} else {
					log.Debugf("replace as root")

					root = c
				}

				queue.Unshift(&unrollToken{
					tok:    c,
					parent: iTok.parent,
					counts: counts,
				})
			} else {
				// we reached a maximum of repetition, we cut and remove dangling tokens
				log.Debugf("reached max repeat of %d for (%p)%#v with child (%p)%#v", tavor.MaxRepeat, t, t, child, child)

				_ = t.Set(nil)

				ta := iTok.tok
				tt := iTok.parent

				/* TODO bring back replacing the returns of InternalLogicalRemove. This was removed because of https://github.com/zimmski/tavor/issues/13 which hit a bug because replaced tokens where still referenced during unrolling somewhere (maybe in the queue?) and I had to move quick.

				repl := func(parent Token, this Token, that Token) {
					log.Debugf("replace (%p)%#v by (%p)%#v", this, this, that, that)

					if parent != nil {
						changed[parent] = struct{}{}

						switch tt := parent.(type) {
						case ForwardToken:
							tt.InternalReplace(this, that)
						case ListToken:
							tt.InternalReplace(this, that)
						}
					} else {
						log.Debugf("replace as root")

						root = that
					}
				}
				*/
			REMOVE:
				for tt != nil {
					delete(parents, tt.tok)

					switch l := tt.tok.(type) {
					case ForwardToken:
						log.Debugf("remove (%p)%#v from (%p)%#v", ta, ta, l, l)

						c := l.InternalLogicalRemove(ta)

						if c != nil {
							/*if c != l {
								repl(tt.parent.tok, l, c)
							}*/

							break REMOVE
						}

						ta = l
						tt = tt.parent
					case ListToken:
						log.Debugf("remove (%p)%#v from (%p)%#v", ta, ta, l, l)

						c := l.InternalLogicalRemove(ta)

						if c != nil {
							/*if c != l {
								repl(tt.parent.tok, l, c)
							}*/

							break REMOVE
						}

						ta = l
						tt = tt.parent
					}
				}
			}
		case ForwardToken:
			if v := t.InternalGet(); v != nil {
				queue.Unshift(&unrollToken{
					tok:    v,
					parent: iTok,
					counts: iTok.counts,
				})

				parents[v] = iTok.tok
			}
		case ListToken:
			for i := t.InternalLen() - 1; i >= 0; i-- {
				c, _ := t.InternalGet(i)

				queue.Unshift(&unrollToken{
					tok:    c,
					parent: iTok,
					counts: iTok.counts,
				})

				parents[c] = iTok.tok
			}
		}
	}

	// force regeneration of possible cloned tokens
	WalkInternalTail(root, func(tok Token) error {
		switch t := tok.(type) {
		case ForwardToken:
			c := t.InternalGet()
			t.InternalReplace(c, c)
		case ListToken:
			for i := 0; i < t.InternalLen(); i++ {
				c, _ := t.InternalGet(i)
				t.InternalReplace(c, c)
			}
		}

		return nil
	})

	log.Debug("finished unrolling")

	return root
}
