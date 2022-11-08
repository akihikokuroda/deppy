package constraints

import (
	pkgsat "github.com/operator-framework/deppy/pkg/sat"
)

var _ pkgsat.IVariable = &IVariable{}

// Variable is a simple implementation of sat.Variable
type IVariable struct {
	id          pkgsat.Identifier
	constraints []pkgsat.IConstraint
}

func (v *IVariable) Identifier() pkgsat.Identifier {
	return v.id
}

func (v *IVariable) Constraints() []pkgsat.IConstraint {
	return v.constraints
}

func (v *IVariable) AddConstraint(constraint ...pkgsat.IConstraint) {
	v.constraints = append(v.constraints, constraint...)
}

func INewVariable(id pkgsat.Identifier, constraints ...pkgsat.IConstraint) *IVariable {
	return &IVariable{
		id:          id,
		constraints: constraints,
	}
}
