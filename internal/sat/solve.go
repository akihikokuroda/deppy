package sat

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/go-air/gini"
	"github.com/go-air/gini/inter"
	"github.com/go-air/gini/z"

	pkgconstraints "github.com/operator-framework/deppy/pkg/constraints"
)

var ErrIncomplete = errors.New("cancelled before a solution could be found")

const (
	satisfiable   = 1
	unsatisfiable = -1
	unknown       = 0
)

type INotSatisfiable []pkgconstraints.IAppliedConstraint

func (e INotSatisfiable) Error() string {
	const msg = "constraints not satisfiable"
	if len(e) == 0 {
		return msg
	}
	s := make([]string, len(e))
	for i, a := range e {
		s[i] = a.String()
	}
	return fmt.Sprintf("%s: %s", msg, strings.Join(s, ", "))
}

type ISolver interface {
	Solve(context.Context) ([]pkgconstraints.IVariable, error)
}

type Isolver struct {
	g      inter.S
	litMap *ILitMapping
	tracer ITracer
	buffer []z.Lit
}

func INewSolver(options ...IOption) (ISolver, error) {
	s := Isolver{g: gini.New()}
	for _, option := range append(options, Idefaults...) {
		if err := option(&s); err != nil {
			return nil, err
		}
	}
	return &s, nil
}

type IOption func(s *Isolver) error

func IWithInput(input []pkgconstraints.IVariable) IOption {
	return func(s *Isolver) error {
		var err error
		s.litMap, err = NewILitMapping(input)
		return err
	}
}

var Idefaults = []IOption{
	func(s *Isolver) error {
		if s.litMap == nil {
			var err error
			s.litMap, err = NewILitMapping(nil)
			return err
		}
		return nil
	},
	func(s *Isolver) error {
		if s.tracer == nil {
			s.tracer = IDefaultTracer{}
		}
		return nil
	},
}

func (s *Isolver) Solve(ctx context.Context) (result []pkgconstraints.IVariable, err error) {
	defer func() {
		// This likely indicates a bug, so discard whatever
		// return values were produced.
		if derr := s.litMap.Error(); derr != nil {
			result = nil
			err = derr
		}
	}()

	// teach all constraints to the solver
	s.litMap.AddConstraints(s.g)

	// collect literals of all mandatory variables to assume as a baseline
	anchors := s.litMap.AnchorIdentifiers()
	assumptions := make([]z.Lit, len(anchors))
	for i := range s.litMap.AnchorIdentifiers() {
		assumptions[i] = s.litMap.LitOf(anchors[i])
	}

	// assume that all constraints hold
	s.litMap.AssumeConstraints(s.g)
	s.g.Assume(assumptions...)

	var aset map[z.Lit]struct{}
	// push a new test scope with the baseline assumptions, to prevent them from being cleared during search
	outcome, _ := s.g.Test(nil)
	if outcome != satisfiable && outcome != unsatisfiable {
		// searcher for solutions in input order, so that preferences
		// can be taken into acount (i.e. prefer one catalog to another)
		outcome, assumptions, aset = (&ISearch{S: s.g, Slits: s.litMap, Tracer: s.tracer}).Do(context.Background(), assumptions)
	}
	switch outcome {
	case satisfiable:
		s.buffer = s.litMap.Lits(s.buffer)
		var extras, excluded []z.Lit
		for _, m := range s.buffer {
			if _, ok := aset[m]; ok {
				continue
			}
			if !s.g.Value(m) {
				excluded = append(excluded, m.Not())
				continue
			}
			extras = append(extras, m)
		}
		s.g.Untest()
		cs := s.litMap.CardinalityConstrainer(s.g, extras)
		s.g.Assume(assumptions...)
		s.g.Assume(excluded...)
		s.litMap.AssumeConstraints(s.g)
		_, s.buffer = s.g.Test(s.buffer)
		for w := 0; w <= cs.N(); w++ {
			s.g.Assume(cs.Leq(w))
			if s.g.Solve() == satisfiable {
				return s.litMap.Variables(s.g), nil
			}
		}
		// Something is wrong if we can't find a model anymore
		// after optimizing for cardinality.
		return nil, fmt.Errorf("unexpected internal error")
	case unsatisfiable:
		return nil, INotSatisfiable(s.litMap.Conflicts(s.g))
	}

	return nil, ErrIncomplete
}
