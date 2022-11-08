package sat_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/operator-framework/deppy/internal/sat"
	"github.com/operator-framework/deppy/pkg/constraints"
)

type TestVariable struct {
	identifier  constraints.Identifier
	constraints []constraints.IConstraint
}

func (i TestVariable) Identifier() constraints.Identifier {
	return i.identifier
}

func (i TestVariable) Constraints() []constraints.IConstraint {
	return i.constraints
}

func (i TestVariable) GoString() string {
	return fmt.Sprintf("%q", i.Identifier())
}

func variable(id constraints.Identifier, constraints ...constraints.IConstraint) constraints.IVariable {
	return TestVariable{
		identifier:  id,
		constraints: constraints,
	}
}

func TestNotSatisfiableError(t *testing.T) {
	type tc struct {
		Name   string
		Error  sat.INotSatisfiable
		String string
	}

	for _, tt := range []tc{
		{
			Name:   "nil",
			String: "constraints not satisfiable",
		},
		{
			Name:   "empty",
			String: "constraints not satisfiable",
			Error:  sat.INotSatisfiable{},
		},
		{
			Name: "single failure",
			Error: sat.INotSatisfiable{
				constraints.IAppliedConstraint{
					Variable:   variable("a", constraints.IMandatory()),
					Constraint: constraints.IMandatory(),
				},
			},
			String: fmt.Sprintf("constraints not satisfiable: %s",
				"a"),
		},
		{
			Name: "multiple failures",
			Error: sat.INotSatisfiable{
				constraints.IAppliedConstraint{
					Variable:   variable("a", constraints.IMandatory()),
					Constraint: constraints.IMandatory(),
				},
				constraints.IAppliedConstraint{
					Variable:   variable("b", constraints.IProhibited()),
					Constraint: constraints.IProhibited(),
				},
			},
			String: fmt.Sprintf("constraints not satisfiable: %s, %s",
				//				constraints.IMandatory().String("a"), constraints.IProhibited().String("b")),
				"a", "b"),
		},
	} {
		t.Run(tt.Name, func(t *testing.T) {
			assert.Equal(t, tt.String, tt.Error.Error())
		})
	}
}

func TestSolve(t *testing.T) {
	type tc struct {
		Name      string
		Variables []constraints.IVariable
		Installed []constraints.Identifier
		Error     error
	}

	for _, tt := range []tc{
		{
			Name: "no variables",
		},
		{
			Name:      "unnecessary variable is not installed",
			Variables: []constraints.IVariable{variable("a")},
		},
		{
			Name:      "single mandatory variable is installed",
			Variables: []constraints.IVariable{variable("a", constraints.IMandatory())},
			Installed: []constraints.Identifier{"a"},
		},
		{
			Name:      "both mandatory and prohibited produce error",
			Variables: []constraints.IVariable{variable("a", constraints.IMandatory(), constraints.IProhibited())},
			Error: sat.INotSatisfiable{
				{
					Variable:   variable("a", constraints.IMandatory(), constraints.IProhibited()),
					Constraint: constraints.IMandatory(),
				},
				{
					Variable:   variable("a", constraints.IMandatory(), constraints.IProhibited()),
					Constraint: constraints.IProhibited(),
				},
			},
		},
		{
			Name: "dependency is installed",
			Variables: []constraints.IVariable{
				variable("a"),
				variable("b", constraints.IMandatory(), constraints.IDependency("a")),
			},
			Installed: []constraints.Identifier{"a", "b"},
		},
		{
			Name: "transitive dependency is installed",
			Variables: []constraints.IVariable{
				variable("a"),
				variable("b", constraints.IDependency("a")),
				variable("c", constraints.IMandatory(), constraints.IDependency("b")),
			},
			Installed: []constraints.Identifier{"a", "b", "c"},
		},
		{
			Name: "both dependencies are installed",
			Variables: []constraints.IVariable{
				variable("a"),
				variable("b"),
				variable("c", constraints.IMandatory(), constraints.IDependency("a"), constraints.IDependency("b")),
			},
			Installed: []constraints.Identifier{"a", "b", "c"},
		},
		{
			Name: "solution with first dependency is selected",
			Variables: []constraints.IVariable{
				variable("a"),
				variable("b", constraints.IConflict("a")),
				variable("c", constraints.IMandatory(), constraints.IDependency("a", "b")),
			},
			Installed: []constraints.Identifier{"a", "c"},
		},
		{
			Name: "solution with only first dependency is selected",
			Variables: []constraints.IVariable{
				variable("a"),
				variable("b"),
				variable("c", constraints.IMandatory(), constraints.IDependency("a", "b")),
			},
			Installed: []constraints.Identifier{"a", "c"},
		},
		{
			Name: "solution with first dependency is selected (reverse)",
			Variables: []constraints.IVariable{
				variable("a"),
				variable("b", constraints.IConflict("a")),
				variable("c", constraints.IMandatory(), constraints.IDependency("b", "a")),
			},
			Installed: []constraints.Identifier{"b", "c"},
		},
		{
			Name: "two mandatory but conflicting packages",
			Variables: []constraints.IVariable{
				variable("a", constraints.IMandatory()),
				variable("b", constraints.IMandatory(), constraints.IConflict("a")),
			},
			Error: sat.INotSatisfiable{
				{
					Variable:   variable("a", constraints.IMandatory()),
					Constraint: constraints.IMandatory(),
				},
				{
					Variable:   variable("b", constraints.IMandatory(), constraints.IConflict("a")),
					Constraint: constraints.IMandatory(),
				},
				{
					Variable:   variable("b", constraints.IMandatory(), constraints.IConflict("a")),
					Constraint: constraints.IConflict("a"),
				},
			},
		},
		{
			Name: "irrelevant dependencies don't influence search order",
			Variables: []constraints.IVariable{
				variable("a", constraints.IDependency("x", "y")),
				variable("b", constraints.IMandatory(), constraints.IDependency("y", "x")),
				variable("x"),
				variable("y"),
			},
			Installed: []constraints.Identifier{"b", "y"},
		},
		{
			Name: "cardinality constraint prevents resolution",
			Variables: []constraints.IVariable{
				variable("a", constraints.IMandatory(), constraints.IDependency("x", "y"), constraints.IAtMost(1, "x", "y")),
				variable("x", constraints.IMandatory()),
				variable("y", constraints.IMandatory()),
			},
			Error: sat.INotSatisfiable{
				{
					Variable:   variable("a", constraints.IMandatory(), constraints.IDependency("x", "y"), constraints.IAtMost(1, "x", "y")),
					Constraint: constraints.IAtMost(1, "x", "y"),
				},
				{
					Variable:   variable("x", constraints.IMandatory()),
					Constraint: constraints.IMandatory(),
				},
				{
					Variable:   variable("y", constraints.IMandatory()),
					Constraint: constraints.IMandatory(),
				},
			},
		},
		{
			Name: "cardinality constraint forces alternative",
			Variables: []constraints.IVariable{
				variable("a", constraints.IMandatory(), constraints.IDependency("x", "y"), constraints.IAtMost(1, "x", "y")),
				variable("b", constraints.IMandatory(), constraints.IDependency("y")),
				variable("x"),
				variable("y"),
			},
			Installed: []constraints.Identifier{"a", "b", "y"},
		},
		{
			Name: "two dependencies satisfied by one variable",
			Variables: []constraints.IVariable{
				variable("a", constraints.IMandatory(), constraints.IDependency("y")),
				variable("b", constraints.IMandatory(), constraints.IDependency("x", "y")),
				variable("x"),
				variable("y"),
			},
			Installed: []constraints.Identifier{"a", "b", "y"},
		},
		{
			Name: "foo two dependencies satisfied by one variable",
			Variables: []constraints.IVariable{
				variable("a", constraints.IMandatory(), constraints.IDependency("y", "z", "m")),
				variable("b", constraints.IMandatory(), constraints.IDependency("x", "y")),
				variable("x"),
				variable("y"),
				variable("z"),
				variable("m"),
			},
			Installed: []constraints.Identifier{"a", "b", "y"},
		},
		{
			Name: "result size larger than minimum due to preference",
			Variables: []constraints.IVariable{
				variable("a", constraints.IMandatory(), constraints.IDependency("x", "y")),
				variable("b", constraints.IMandatory(), constraints.IDependency("y")),
				variable("x"),
				variable("y"),
			},
			Installed: []constraints.Identifier{"a", "b", "x", "y"},
		},
		{
			Name: "only the least preferable choice is acceptable",
			Variables: []constraints.IVariable{
				variable("a", constraints.IMandatory(), constraints.IDependency("a1", "a2")),
				variable("a1", constraints.IConflict("c1"), constraints.IConflict("c2")),
				variable("a2", constraints.IConflict("c1")),
				variable("b", constraints.IMandatory(), constraints.IDependency("b1", "b2")),
				variable("b1", constraints.IConflict("c1"), constraints.IConflict("c2")),
				variable("b2", constraints.IConflict("c1")),
				variable("c", constraints.IMandatory(), constraints.IDependency("c1", "c2")),
				variable("c1"),
				variable("c2"),
			},
			Installed: []constraints.Identifier{"a", "a2", "b", "b2", "c", "c2"},
		},
		{
			Name: "preferences respected with multiple dependencies per variable",
			Variables: []constraints.IVariable{
				variable("a", constraints.IMandatory(), constraints.IDependency("x1", "x2"), constraints.IDependency("y1", "y2")),
				variable("x1"),
				variable("x2"),
				variable("y1"),
				variable("y2"),
			},
			Installed: []constraints.Identifier{"a", "x1", "y1"},
		},
	} {
		t.Run(tt.Name, func(t *testing.T) {
			assert := assert.New(t)

			var traces bytes.Buffer
			s, err := sat.INewSolver(sat.IWithInput(tt.Variables), sat.WithTracer(sat.ILoggingTracer{Writer: &traces}))
			if err != nil {
				t.Fatalf("failed to initialize solver: %s", err)
			}

			installed, err := s.Solve(context.TODO())

			if installed != nil {
				sort.SliceStable(installed, func(i, j int) bool {
					return installed[i].Identifier() < installed[j].Identifier()
				})
			}

			// Failed constraints are sorted in lexically
			// increasing order of the identifier of the
			// constraint's variable, with ties broken
			// in favor of the constraint that appears
			// earliest in the variable's list of
			// constraints.
			var ns sat.INotSatisfiable
			if errors.As(err, &ns) {
				sort.SliceStable(ns, func(i, j int) bool {
					if ns[i].Variable.Identifier() != ns[j].Variable.Identifier() {
						return ns[i].Variable.Identifier() < ns[j].Variable.Identifier()
					}
					var x, y int
					for ii, c := range ns[i].Variable.Constraints() {
						if reflect.DeepEqual(c, ns[i].Constraint) {
							x = ii
							break
						}
					}
					for ij, c := range ns[j].Variable.Constraints() {
						if reflect.DeepEqual(c, ns[j].Constraint) {
							y = ij
							break
						}
					}
					return x < y
				})
			}

			var ids []constraints.Identifier
			for _, variable := range installed {
				ids = append(ids, variable.Identifier())
			}
			assert.Equal(tt.Installed, ids)
			assert.Equal(tt.Error, err)

			if t.Failed() {
				t.Logf("\n%s", traces.String())
			}
		})
	}
}

func TestDuplicateIdentifier(t *testing.T) {
	_, err := sat.INewSolver(sat.IWithInput([]constraints.IVariable{
		variable("a"),
		variable("a"),
	}))
	assert.Equal(t, sat.DuplicateIdentifier("a"), err)
}
