package sat

import (
	"fmt"
	"strings"

	"github.com/go-air/gini/inter"
	"github.com/go-air/gini/logic"
	"github.com/go-air/gini/z"
)

type DuplicateIdentifier Identifier

func (e DuplicateIdentifier) Error() string {
	return fmt.Sprintf("duplicate identifier %q in input", Identifier(e))
}

type inconsistentLitMapping []error

func (inconsistentLitMapping) Error() string {
	return "internal solver failure"
}

// LitMapping performs translation between the input and output types of
// Solve (Constraints, Variables, etc.) and the variables that
// appear in the SAT formula.
type LitMapping struct {
	inorder     []Variable
	variables   map[z.Lit]Variable
	lits        map[Identifier]z.Lit
	constraints map[z.Lit]AppliedConstraint
	c           *logic.C
	errs        inconsistentLitMapping
}

// LitMapping performs translation between the input and output types of
// Solve (Constraints, Variables, etc.) and the variables that
// appear in the SAT formula.
type ILitMapping struct {
	inorder     []IVariable
	variables   map[z.Lit]IVariable
	lits        map[Identifier]z.Lit
	constraints map[z.Lit]IAppliedConstraint
	c           *logic.C
	errs        inconsistentLitMapping
}

// newLitMapping returns a new LitMapping with its state initialized based on
// the provided slice of Variables. This includes construction of
// the translation tables between Variables/Constraints and the
// inputs to the underlying solver.
func NewLitMapping(variables []Variable) (*LitMapping, error) {
	d := LitMapping{
		inorder:     variables,
		variables:   make(map[z.Lit]Variable, len(variables)),
		lits:        make(map[Identifier]z.Lit, len(variables)),
		constraints: make(map[z.Lit]AppliedConstraint),
		c:           logic.NewCCap(len(variables)),
	}

	// First pass to assign lits:
	for _, variable := range variables {
		im := d.c.Lit()
		if _, ok := d.lits[variable.Identifier()]; ok {
			return nil, DuplicateIdentifier(variable.Identifier())
		}
		d.lits[variable.Identifier()] = im
		d.variables[im] = variable
	}

	for _, variable := range variables {
		for _, constraint := range variable.Constraints() {
			m := constraint.Apply(d.c, d, variable.Identifier())
			if m == z.LitNull {
				// This constraint doesn't have a
				// useful representation in the SAT
				// inputs.
				continue
			}

			d.constraints[m] = AppliedConstraint{
				Variable:   variable,
				Constraint: constraint,
			}
		}
	}

	return &d, nil
}

// newLitMapping returns a new LitMapping with its state initialized based on
// the provided slice of Variables. This includes construction of
// the translation tables between Variables/Constraints and the
// inputs to the underlying solver.
func NewILitMapping(variables []IVariable) (*ILitMapping, error) {
	d := ILitMapping{
		inorder:     variables,
		variables:   make(map[z.Lit]IVariable, len(variables)),
		lits:        make(map[Identifier]z.Lit, len(variables)),
		constraints: make(map[z.Lit]IAppliedConstraint),
		c:           logic.NewCCap(len(variables)),
	}

	// First pass to assign lits:
	for _, variable := range variables {
		im := d.c.Lit()
		if _, ok := d.lits[variable.Identifier()]; ok {
			return nil, DuplicateIdentifier(variable.Identifier())
		}
		d.lits[variable.Identifier()] = im
		d.variables[im] = variable
	}

	for _, variable := range variables {
		for _, constraint := range variable.Constraints() {
			var m z.Lit
			switch constraint.ConstraintType {
			case "mandatory":
				m = d.LitOf(variable.Identifier())
			case "prohibited":
				m = d.LitOf(variable.Identifier()).Not()
			case "not":
				m = d.LitOf(variable.Identifier()).Not()
			case "dependency":
				m = d.LitOf(variable.Identifier()).Not()
				for _, each := range constraint.Properties["ids"].([]Identifier) {
					m = d.c.Or(m, d.LitOf(each))
				}
			case "conflict":
				m = d.c.Or(d.LitOf(variable.Identifier()).Not(), d.LitOf(Identifier(constraint.Properties["id"].(Identifier))).Not())
			case "atmost":
				ms := make([]z.Lit, len(constraint.Properties["ids"].([]Identifier)))
				for i, each := range constraint.Properties["ids"].([]Identifier) {
					ms[i] = d.LitOf(each)
				}
				m = d.c.CardSort(ms).Leq(constraint.Properties["n"].(int))
			case "or":
				subjectLit := d.LitOf(variable.Identifier())
				_, ok := constraint.Properties["issubjectnegated"]
				if ok && constraint.Properties["issubjectnegated"].(bool) {
					subjectLit = subjectLit.Not()
				}
				var operandLit z.Lit
				_, ok = constraint.Properties["id"]
				if ok {
					operandLit = d.LitOf(constraint.Properties["id"].(Identifier))
					_, ok = constraint.Properties["isoperandnegated"]
					if ok && constraint.Properties["isoperandnegated"].(bool) {
						operandLit = operandLit.Not()
					}
				}
				m = d.c.Or(subjectLit, operandLit)
			}
			if m == z.LitNull {
				// This constraint doesn't have a
				// useful representation in the SAT
				// inputs.
				fmt.Printf("Continue %v\n", variable.Identifier())
				continue
			}

			d.constraints[m] = IAppliedConstraint{
				Variable:   variable,
				Constraint: constraint,
			}
		}
	}

	return &d, nil
}

// LitOf returns the positive literal corresponding to the Variable
// with the given Identifier.
func (d *LitMapping) LitOf(id Identifier) z.Lit {
	m, ok := d.lits[id]
	if ok {
		return m
	}
	d.errs = append(d.errs, fmt.Errorf("variable %q referenced but not provided", id))
	return z.LitNull
}

// VariableOf returns the Variable corresponding to the provided
// literal, or a zeroVariable if no such Variable exists.
func (d *LitMapping) VariableOf(m z.Lit) Variable {
	i, ok := d.variables[m]
	if ok {
		return i
	}
	d.errs = append(d.errs, fmt.Errorf("no variable corresponding to %s", m))
	return zeroVariable{}
}

// ConstraintOf returns the constraint application corresponding to
// the provided literal, or a zeroConstraint if no such constraint
// exists.
func (d *LitMapping) ConstraintOf(m z.Lit) AppliedConstraint {
	if a, ok := d.constraints[m]; ok {
		return a
	}
	d.errs = append(d.errs, fmt.Errorf("no constraint corresponding to %s", m))
	return AppliedConstraint{
		Variable:   zeroVariable{},
		Constraint: zeroConstraint{},
	}
}

// Error returns a single error value that is an aggregation of all
// errors encountered during a LitMapping's lifetime, or nil if there have
// been no errors. A non-nil return value likely indicates a problem
// with the solver or constraint implementations.
func (d *LitMapping) Error() error {
	if len(d.errs) == 0 {
		return nil
	}
	s := make([]string, len(d.errs))
	for i, err := range d.errs {
		s[i] = err.Error()
	}
	return fmt.Errorf("%d errors encountered: %s", len(s), strings.Join(s, ", "))
}

// AddConstraints adds the current constraints encoded in the embedded circuit to the
// solver g
func (d *LitMapping) AddConstraints(g inter.S) {
	d.c.ToCnf(g)
}

func (d *LitMapping) AssumeConstraints(s inter.S) {
	for m := range d.constraints {
		s.Assume(m)
	}
}

// CardinalityConstrainer constructs a sorting network to provide
// cardinality constraints over the provided slice of literals. Any
// new clauses and variables are translated to CNF and taught to the
// given inter.Adder, so this function will panic if it is in a test
// context.
func (d *LitMapping) CardinalityConstrainer(g inter.Adder, ms []z.Lit) *logic.CardSort {
	clen := d.c.Len()
	cs := d.c.CardSort(ms)
	marks := make([]int8, clen, d.c.Len())
	for i := range marks {
		marks[i] = 1
	}
	for w := 0; w <= cs.N(); w++ {
		marks, _ = d.c.CnfSince(g, marks, cs.Leq(w))
	}
	return cs
}

// AnchorIdentifiers returns a slice containing the Identifiers of
// every Variable with at least one "anchor" constraint, in the
// order they appear in the input.
func (d *LitMapping) AnchorIdentifiers() []Identifier {
	var ids []Identifier
	for _, variable := range d.inorder {
		for _, constraint := range variable.Constraints() {
			if constraint.Anchor() {
				ids = append(ids, variable.Identifier())
				break
			}
		}
	}
	return ids
}

func (d *LitMapping) Variables(g inter.S) []Variable {
	var result []Variable
	for _, i := range d.inorder {
		if g.Value(d.LitOf(i.Identifier())) {
			result = append(result, i)
		}
	}
	return result
}

func (d *LitMapping) Lits(dst []z.Lit) []z.Lit {
	if cap(dst) < len(d.inorder) {
		dst = make([]z.Lit, 0, len(d.inorder))
	}
	dst = dst[:0]
	for _, i := range d.inorder {
		m := d.LitOf(i.Identifier())
		dst = append(dst, m)
	}
	return dst
}

func (d *LitMapping) Conflicts(g inter.Assumable) []AppliedConstraint {
	whys := g.Why(nil)
	as := make([]AppliedConstraint, 0, len(whys))
	for _, why := range whys {
		if a, ok := d.constraints[why]; ok {
			as = append(as, a)
		}
	}
	return as
}

//	=============================================
//
// LitOf returns the positive literal corresponding to the Variable
// with the given Identifier.
func (d *ILitMapping) LitOf(id Identifier) z.Lit {
	m, ok := d.lits[id]
	if ok {
		return m
	}
	d.errs = append(d.errs, fmt.Errorf("variable %q referenced but not provided", id))
	return z.LitNull
}

// VariableOf returns the Variable corresponding to the provided
// literal, or a zeroVariable if no such Variable exists.
func (d *ILitMapping) VariableOf(m z.Lit) IVariable {
	i, ok := d.variables[m]
	if ok {
		return i
	}
	d.errs = append(d.errs, fmt.Errorf("no variable corresponding to %s", m))
	return zeroIVariable{}
}

// ConstraintOf returns the constraint application corresponding to
// the provided literal, or a zeroConstraint if no such constraint
// exists.
func (d *ILitMapping) ConstraintOf(m z.Lit) IAppliedConstraint {
	if a, ok := d.constraints[m]; ok {
		return a
	}
	d.errs = append(d.errs, fmt.Errorf("no constraint corresponding to %s", m))
	return IAppliedConstraint{
		Variable:   zeroIVariable{},
		Constraint: IConstraint{ConstraintType: "zero", Order: nil},
	}
}

// Error returns a single error value that is an aggregation of all
// errors encountered during a LitMapping's lifetime, or nil if there have
// been no errors. A non-nil return value likely indicates a problem
// with the solver or constraint implementations.
func (d *ILitMapping) Error() error {
	if len(d.errs) == 0 {
		return nil
	}
	s := make([]string, len(d.errs))
	for i, err := range d.errs {
		s[i] = err.Error()
	}
	return fmt.Errorf("%d errors encountered: %s", len(s), strings.Join(s, ", "))
}

// AddConstraints adds the current constraints encoded in the embedded circuit to the
// solver g
func (d *ILitMapping) AddConstraints(g inter.S) {
	d.c.ToCnf(g)
}

func (d *ILitMapping) AssumeConstraints(s inter.S) {
	for m := range d.constraints {
		s.Assume(m)
	}
}

// CardinalityConstrainer constructs a sorting network to provide
// cardinality constraints over the provided slice of literals. Any
// new clauses and variables are translated to CNF and taught to the
// given inter.Adder, so this function will panic if it is in a test
// context.
func (d *ILitMapping) CardinalityConstrainer(g inter.Adder, ms []z.Lit) *logic.CardSort {
	clen := d.c.Len()
	cs := d.c.CardSort(ms)
	marks := make([]int8, clen, d.c.Len())
	for i := range marks {
		marks[i] = 1
	}
	for w := 0; w <= cs.N(); w++ {
		marks, _ = d.c.CnfSince(g, marks, cs.Leq(w))
	}
	return cs
}

// AnchorIdentifiers returns a slice containing the Identifiers of
// every Variable with at least one "anchor" constraint, in the
// order they appear in the input.
func (d *ILitMapping) AnchorIdentifiers() []Identifier {
	var ids []Identifier
	for _, variable := range d.inorder {
		for _, constraint := range variable.Constraints() {
			if constraint.Anchor {
				ids = append(ids, variable.Identifier())
				break
			}
		}
	}
	return ids
}

func (d *ILitMapping) Variables(g inter.S) []IVariable {
	var result []IVariable
	for _, i := range d.inorder {
		if g.Value(d.LitOf(i.Identifier())) {
			result = append(result, i)
		}
	}
	return result
}

func (d *ILitMapping) Lits(dst []z.Lit) []z.Lit {
	if cap(dst) < len(d.inorder) {
		dst = make([]z.Lit, 0, len(d.inorder))
	}
	dst = dst[:0]
	for _, i := range d.inorder {
		m := d.LitOf(i.Identifier())
		dst = append(dst, m)
	}
	return dst
}

func (d *ILitMapping) Conflicts(g inter.Assumable) []IAppliedConstraint {
	whys := g.Why(nil)
	as := make([]IAppliedConstraint, 0, len(whys))
	for _, why := range whys {
		if a, ok := d.constraints[why]; ok {
			as = append(as, a)
		}
	}
	return as
}
