package constraints

type Identifier string

func (id Identifier) String() string {
	return string(id)
}

// Constraint implementations limit the circumstances under which a
// particular Variable can appear in a solution.
type IConstraint struct {
	ConstraintType string
	Properties     map[string]interface{}
	Anchor         bool
	Order          []Identifier
}

// Variable values are the basic unit of problems and solutions
// understood by this package.
type IVariable interface {
	// Identifier returns the Identifier that uniquely identifies
	// this Variable among all other Variables in a given
	// problem.
	Identifier() Identifier
	// Constraints returns the set of constraints that apply to
	// this Variable.
	Constraints() []IConstraint
}

// AppliedConstraint values compose a single Constraint with the
// Variable it applies to.
type IAppliedConstraint struct {
	Variable   IVariable
	Constraint IConstraint
}

// String implements fmt.Stringer and returns a human-readable message
// representing the receiver.
func (a IAppliedConstraint) String() string {
	return a.Variable.Identifier().String()
}

var _ IVariable = &Variable{}

// Variable is a simple implementation of sat.Variable
type Variable struct {
	id          Identifier
	constraints []IConstraint
}

func (v *Variable) Identifier() Identifier {
	return v.id
}

func (v *Variable) Constraints() []IConstraint {
	return v.constraints
}

func (v *Variable) AddConstraint(constraint ...IConstraint) {
	v.constraints = append(v.constraints, constraint...)
}

func INewVariable(id Identifier, constraints ...IConstraint) *Variable {
	return &Variable{
		id:          id,
		constraints: constraints,
	}
}
