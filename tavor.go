package tavor

import (
	"fmt"
	"io"
	"strings"

	"github.com/zimmski/container/list/linkedlist"

	"github.com/zimmski/tavor/log"
	"github.com/zimmski/tavor/token"
	"github.com/zimmski/tavor/token/lists"
	"github.com/zimmski/tavor/token/primitives"
)

const (
	Version = "0.2"
)

const (
	MaxRepeat = 2
)

func PrettyPrintTree(w io.Writer, root token.Token) {
	prettyPrintTreeRek(w, root, 0)
}

func prettyPrintTreeRek(w io.Writer, tok token.Token, level int) {
	fmt.Fprintf(w, "%s(%p)%#v\n", strings.Repeat("\t", level), tok, tok)

	switch t := tok.(type) {
	case token.ForwardToken:
		if v := t.Get(); v != nil {
			prettyPrintTreeRek(w, v, level+1)
		}
	case lists.List:
		for i := 0; i < t.Len(); i++ {
			c, _ := t.Get(i)

			prettyPrintTreeRek(w, c, level+1)
		}
	}
}

func PrettyPrintInternalTree(w io.Writer, root token.Token) {
	prettyPrintInternalTreeRek(w, root, 0)
}

func prettyPrintInternalTreeRek(w io.Writer, tok token.Token, level int) {
	fmt.Fprintf(w, "%s(%p)%#v\n", strings.Repeat("\t", level), tok, tok)

	switch t := tok.(type) {
	case token.ForwardToken:
		if v := t.InternalGet(); v != nil {
			prettyPrintInternalTreeRek(w, v, level+1)
		}
	case lists.List:
		for i := 0; i < t.InternalLen(); i++ {
			c, _ := t.InternalGet(i)

			prettyPrintInternalTreeRek(w, c, level+1)
		}
	}
}

func LoopExists(root token.Token) bool {
	lookup := make(map[token.Token]struct{})
	queue := linkedlist.New()

	queue.Push(root)

	for !queue.Empty() {
		v, _ := queue.Shift()
		t, _ := v.(token.Token)

		lookup[t] = struct{}{}

		switch tok := t.(type) {
		case *primitives.Pointer:
			if v := tok.InternalGet(); v != nil {
				if _, ok := lookup[v]; ok {
					log.Debugf("found a loop through (%p)%#v", t, t)

					return true
				}

				queue.Push(v)
			}
		case token.ForwardToken:
			if v := tok.InternalGet(); v != nil {
				queue.Push(v)
			}
		case lists.List:
			for i := 0; i < tok.InternalLen(); i++ {
				c, _ := tok.InternalGet(i)

				queue.Push(c)
			}
		}
	}

	return false
}

func UnrollPointers(root token.Token) token.Token {
	type unrollToken struct {
		tok    token.Token
		parent *unrollToken
	}

	log.Debug("start unrolling pointers by cloning them")

	checked := make(map[token.Token]token.Token)
	counters := make(map[token.Token]int)

	parents := make(map[token.Token]token.Token)
	changed := make(map[token.Token]struct{})

	queue := linkedlist.New()

	queue.Push(&unrollToken{
		tok:    root,
		parent: nil,
	})
	parents[root] = nil

	for !queue.Empty() {
		v, _ := queue.Shift()
		iTok, _ := v.(*unrollToken)

		switch t := iTok.tok.(type) {
		case *primitives.Pointer:
			o := t.InternalGet()

			parent, ok := checked[o]
			times := 0

			if ok {
				times = counters[parent]
			} else {
				parent = o.Clone()
				checked[o] = parent
			}

			if times != MaxRepeat {
				log.Debugf("clone (%p)%#v with parent (%p)%#v", t, t, parent, parent)

				c := parent.Clone()

				t.Set(c)

				counters[parent] = times + 1
				checked[c] = parent

				if iTok.parent != nil {
					log.Debugf("replace in (%p)%#v", iTok.parent.tok, iTok.parent.tok)

					changed[iTok.parent.tok] = struct{}{}

					switch tt := iTok.parent.tok.(type) {
					case token.ForwardToken:
						tt.InternalReplace(t, c)
					case lists.List:
						tt.InternalReplace(t, c)
					}
				} else {
					log.Debugf("replace as root")

					root = c
				}

				queue.Unshift(&unrollToken{
					tok:    c,
					parent: iTok.parent,
				})
			} else {
				log.Debugf("reached max repeat of %d for (%p)%#v with parent (%p)%#v", MaxRepeat, t, t, parent, parent)

				t.Set(nil)

				ta := iTok.tok
				tt := iTok.parent

			REMOVE:
				for tt != nil {
					delete(parents, tt.tok)
					delete(changed, tt.tok)

					switch l := tt.tok.(type) {
					case token.ForwardToken:
						log.Debugf("remove (%p)%#v from (%p)%#v", ta, ta, l, l)

						c := l.InternalLogicalRemove(ta)

						if c != nil {
							break REMOVE
						}

						ta = l
						tt = tt.parent
					case lists.List:
						log.Debugf("remove (%p)%#v from (%p)%#v", ta, ta, l, l)

						c := l.InternalLogicalRemove(ta)

						if c != nil {
							break REMOVE
						}

						ta = l
						tt = tt.parent
					}
				}
			}
		case token.ForwardToken:
			if v := t.InternalGet(); v != nil {
				queue.Push(&unrollToken{
					tok:    v,
					parent: iTok,
				})

				parents[v] = iTok.tok
			}
		case lists.List:
			for i := 0; i < t.InternalLen(); i++ {
				c, _ := t.InternalGet(i)

				queue.Push(&unrollToken{
					tok:    c,
					parent: iTok,
				})

				parents[c] = iTok.tok
			}
		}
	}

	// we need to update some tokens with the same child to regenerate clones
	for child := range changed {
		parent := parents[child]

		if parent == nil {
			continue
		}

		log.Debugf("update (%p)%#v with child (%p)%#v", parent, parent, child, child)

		switch tt := parent.(type) {
		case token.ForwardToken:
			tt.InternalReplace(child, child)
		case lists.List:
			tt.InternalReplace(child, child)
		}
	}

	log.Debug("finished unrolling")

	return root
}
