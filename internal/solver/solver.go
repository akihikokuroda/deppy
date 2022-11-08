package solver

import (
	"context"

	"github.com/operator-framework/deppy/internal/sat"
	pkgconstraints "github.com/operator-framework/deppy/pkg/constraints"
	"github.com/operator-framework/deppy/pkg/entitysource"
	pkgsolver "github.com/operator-framework/deppy/pkg/solver"
)

type IDeppySolver struct {
	entitySourceGroup    entitysource.EntitySource
	constraintAggregator pkgconstraints.IConstraintGenerator
}

func INewDeppySolver(entitySourceGroup entitysource.EntitySource, constraintAggregator pkgconstraints.IConstraintGenerator) (IDeppySolver, error) {
	return IDeppySolver{
		entitySourceGroup:    entitySourceGroup,
		constraintAggregator: constraintAggregator,
	}, nil
}

func (d IDeppySolver) Solve(ctx context.Context) (pkgsolver.Solution, error) {
	vars, err := d.constraintAggregator.GetVariables(ctx, d.entitySourceGroup)
	if err != nil {
		return nil, err
	}

	satSolver, err := sat.INewSolver(sat.IWithInput(vars))
	if err != nil {
		return nil, err
	}

	selection, err := satSolver.Solve(ctx)
	if err != nil {
		return nil, err
	}

	solution := pkgsolver.Solution{}
	for _, variable := range vars {
		if entity := d.entitySourceGroup.Get(ctx, entitysource.EntityID(variable.Identifier())); entity != nil {
			solution[entity.ID()] = false
		}
	}
	for _, variable := range selection {
		if entity := d.entitySourceGroup.Get(ctx, entitysource.EntityID(variable.Identifier())); entity != nil {
			solution[entity.ID()] = true
		}
	}
	return solution, nil
}
