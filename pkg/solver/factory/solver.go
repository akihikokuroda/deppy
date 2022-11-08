package factory

import (
	internalsolver "github.com/operator-framework/deppy/internal/solver"
	"github.com/operator-framework/deppy/pkg/constraints"
	"github.com/operator-framework/deppy/pkg/entitysource"
	pkgsolver "github.com/operator-framework/deppy/pkg/solver"
)

func NewDeppySolver(entitySourceGroup entitysource.EntitySource, constraintAggregator constraints.IConstraintGenerator) (pkgsolver.Solver, error) {
	return internalsolver.INewDeppySolver(entitySourceGroup, constraintAggregator)
}
