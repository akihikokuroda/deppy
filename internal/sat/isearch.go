package sat

import (
	"context"

	"github.com/go-air/gini/inter"
	"github.com/go-air/gini/z"

	pkgconstraints "github.com/operator-framework/deppy/pkg/constraints"
)

type Choice struct {
	prev, next *Choice
	index      int // index of next unguessed literal
	candidates []z.Lit
}

type guess struct {
	m          z.Lit // if z.LitNull, this choice was satisfied by a previous assumption
	index      int   // index of guessed literal in candidates
	children   int   // number of choices introduced by making this guess
	candidates []z.Lit
}

type ISearch struct {
	S                      inter.S
	Slits                  *ILitMapping
	assumptions            map[z.Lit]struct{} // set of assumed lits - duplicates guess stack - for fast lookup
	guesses                []guess            // stack of assumed guesses
	headChoice, tailChoice *Choice            // deque of unmade Choices
	Tracer                 ITracer
	result                 int
	buffer                 []z.Lit
}

func (h *ISearch) PushGuess() {
	c := h.PopChoiceFront()
	g := guess{
		m:          z.LitNull,
		index:      c.index,
		candidates: c.candidates,
	}
	if g.index < len(g.candidates) {
		g.m = g.candidates[g.index]
	}

	// Check whether or not this choice can be satisfied by an
	// existing assumption.
	for _, m := range g.candidates {
		if _, ok := h.assumptions[m]; ok {
			g.m = z.LitNull
			break
		}
	}

	h.guesses = append(h.guesses, g)
	if g.m == z.LitNull {
		return
	}

	variable := h.Slits.VariableOf(g.m)
	for _, constraint := range variable.Constraints() {
		var ms []z.Lit
		for _, dependency := range constraint.Order {
			ms = append(ms, h.Slits.LitOf(dependency))
		}
		if len(ms) > 0 {
			h.guesses[len(h.guesses)-1].children++
			h.PushChoiceBack(Choice{candidates: ms})
		}
	}

	if h.assumptions == nil {
		h.assumptions = make(map[z.Lit]struct{})
	}
	h.assumptions[g.m] = struct{}{}
	h.S.Assume(g.m)
	h.result, h.buffer = h.S.Test(h.buffer)
}

func (h *ISearch) PopGuess() {
	g := h.guesses[len(h.guesses)-1]
	h.guesses = h.guesses[:len(h.guesses)-1]
	if g.m != z.LitNull {
		delete(h.assumptions, g.m)
		h.result = h.S.Untest()
	}
	for g.children > 0 {
		g.children--
		h.PopChoiceBack()
	}
	c := Choice{
		index:      g.index,
		candidates: g.candidates,
	}
	if g.m != z.LitNull {
		c.index++
	}
	h.PushChoiceFront(c)
}

func (h *ISearch) PushChoiceFront(c Choice) {
	if h.headChoice == nil {
		h.headChoice = &c
		h.tailChoice = &c
		return
	}
	h.headChoice.prev = &c
	c.next = h.headChoice
	h.headChoice = &c
}

func (h *ISearch) PopChoiceFront() Choice {
	c := h.headChoice
	if c.next != nil {
		c.next.prev = nil
	} else {
		h.tailChoice = nil
	}
	h.headChoice = c.next
	return *c
}

func (h *ISearch) PushChoiceBack(c Choice) {
	if h.tailChoice == nil {
		h.headChoice = &c
		h.tailChoice = &c
		return
	}
	h.tailChoice.next = &c
	c.prev = h.tailChoice
	h.tailChoice = &c
}

func (h *ISearch) PopChoiceBack() Choice {
	c := h.tailChoice
	if c.prev != nil {
		c.prev.next = nil
	} else {
		h.headChoice = nil
	}
	h.tailChoice = c.prev
	return *c
}

func (h *ISearch) Result() int {
	return h.result
}

func (h *ISearch) Lits() []z.Lit {
	result := make([]z.Lit, 0, len(h.guesses))
	for _, g := range h.guesses {
		if g.m != z.LitNull {
			result = append(result, g.m)
		}
	}
	return result
}

func (h *ISearch) Do(ctx context.Context, anchors []z.Lit) (int, []z.Lit, map[z.Lit]struct{}) {
	for _, m := range anchors {
		h.PushChoiceBack(Choice{candidates: []z.Lit{m}})
	}

	for {
		// Need to have a definitive result once all choices
		// have been made to decide whether to end or
		// backtrack.
		if h.headChoice == nil && h.result == unknown {
			h.result = h.S.Solve()
		}

		// Backtrack if possible, otherwise end.
		if h.result == unsatisfiable {
			h.Tracer.Trace(h)
			if len(h.guesses) == 0 {
				break
			}
			h.PopGuess()
			continue
		}

		// Satisfiable and no decisions left!
		if h.headChoice == nil {
			break
		}

		// Possibly SAT, keep guessing.
		h.PushGuess()
	}

	lits := h.Lits()
	set := make(map[z.Lit]struct{}, len(lits))
	for _, m := range lits {
		set[m] = struct{}{}
	}
	result := h.Result()

	// Go back to the initial test scope.
	for len(h.guesses) > 0 {
		h.PopGuess()
	}

	return result, lits, set
}

func (h *ISearch) Variables() []pkgconstraints.IVariable {
	result := make([]pkgconstraints.IVariable, 0, len(h.guesses))
	for _, g := range h.guesses {
		if g.m != z.LitNull {
			result = append(result, h.Slits.VariableOf(g.candidates[g.index]))
		}
	}
	return result
}

func (h *ISearch) Conflicts() []pkgconstraints.IAppliedConstraint {
	return h.Slits.Conflicts(h.S)
}
