package constraints

// Mandatory returns a Constraint that will permit only solutions that
// contain a particular Variable.
func IMandatory() IConstraint {
	return IConstraint{
		ConstraintType: "mandatory",
		Anchor:         true,
		Order:          nil,
	}
}

// Prohibited returns a Constraint that will reject any solution that
// contains a particular Variable. Callers may also decide to omit
// an Variable from input to Solve rather than apply such a
// Constraint.
func IProhibited() IConstraint {
	return IConstraint{
		ConstraintType: "prohibited",
		Anchor:         false,
		Order:          nil,
	}
}

func INot() IConstraint {
	return IConstraint{
		ConstraintType: "not",
		Anchor:         false,
		Order:          nil,
	}
}

// Dependency returns a Constraint that will only permit solutions
// containing a given Variable on the condition that at least one
// of the Variables identified by the given Identifiers also
// appears in the solution. Identifiers appearing earlier in the
// argument list have higher preference than those appearing later.
func IDependency(ids ...Identifier) IConstraint {
	return IConstraint{
		ConstraintType: "dependency",
		Properties:     map[string]interface{}{"ids": ids},
		Anchor:         false,
		Order:          ids,
	}
}

// Conflict returns a Constraint that will permit solutions containing
// either the constrained Variable, the Variable identified by
// the given Identifier, or neither, but not both.
func IConflict(id Identifier) IConstraint {
	return IConstraint{
		ConstraintType: "conflict",
		Properties:     map[string]interface{}{"id": id},
		Anchor:         false,
		Order:          nil,
	}
}

// AtMost returns a Constraint that forbids solutions that contain
// more than n of the Variables identified by the given
// Identifiers.
func IAtMost(n int, ids ...Identifier) IConstraint {
	return IConstraint{
		ConstraintType: "atmost",
		Properties:     map[string]interface{}{"n": n, "ids": ids},
		Anchor:         false,
		Order:          nil,
	}
}

// Or returns a constraints in the form subject OR identifier
// if isSubjectNegated = true, ~subject OR identifier
// if isOperandNegated = true, subject OR ~identifier
// if both are true: ~subject OR ~identifier
func IOr(identifier Identifier, isSubjectNegated bool, isOperandNegated bool) IConstraint {
	return IConstraint{
		ConstraintType: "or",
		Properties:     map[string]interface{}{"id": identifier, "isnubjectnegated": isSubjectNegated, "isoperandnegated": isOperandNegated},
		Anchor:         false,
		Order:          nil,
	}
}
