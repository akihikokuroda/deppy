package sat_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/operator-framework/deppy/pkg/constraints"
)

func TestOrder(t *testing.T) {
	type tc struct {
		Name       string
		Constraint constraints.IConstraint
		Expected   []constraints.Identifier
	}

	for _, tt := range []tc{
		{
			Name:       "mandatory",
			Constraint: constraints.IMandatory(),
		},
		{
			Name:       "prohibited",
			Constraint: constraints.IProhibited(),
		},
		{
			Name:       "dependency",
			Constraint: constraints.IDependency("a", "b", "c"),
			Expected:   []constraints.Identifier{"a", "b", "c"},
		},
		{
			Name:       "conflict",
			Constraint: constraints.IConflict("a"),
		},
	} {
		t.Run(tt.Name, func(t *testing.T) {
			assert.Equal(t, tt.Expected, tt.Constraint.Order)
		})
	}
}
