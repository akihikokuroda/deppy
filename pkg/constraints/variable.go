package constraints

import (
	pkgsat "github.com/operator-framework/deppy/pkg/sat"
)

var _ pkgsat.Variable = &Variable{}

// Variable is a simple implementation of sat.Variable
type Variable struct {
	id          pkgsat.Identifier
	constraints []pkgsat.Constraint
}

func (v *Variable) Identifier() pkgsat.Identifier {
	return v.id
}

func (v *Variable) Constraints() []pkgsat.Constraint {
	return v.constraints
}

func (v *Variable) AddConstraint(constraint ...pkgsat.Constraint) {
	v.constraints = append(v.constraints, constraint...)
}

func NewVariable(id pkgsat.Identifier, constraints ...pkgsat.IConstraint) *IVariable {
	return &IVariable{
		id:          id,
		constraints: constraints,
	}
}

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
