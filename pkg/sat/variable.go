package sat

// Identifier values uniquely identify particular Variables within
// the input to a single call to Solve.
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
