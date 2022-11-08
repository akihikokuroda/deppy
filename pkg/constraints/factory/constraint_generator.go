package cfactory

import (
	internalconstraints "github.com/operator-framework/deppy/internal/constraints"
	pkgconstraints "github.com/operator-framework/deppy/pkg/constraints"
)

func INewConstraintAggregator(constraintGenerators ...pkgconstraints.IConstraintGenerator) pkgconstraints.IConstraintGenerator {
	return internalconstraints.INewConstraintAggregator(constraintGenerators)
}
