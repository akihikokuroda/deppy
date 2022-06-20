/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	deppyv1alpha1 "github.com/operator-framework/deppy/api/v1alpha1"
	"github.com/operator-framework/deppy/internal/solver"
)

// InputReconciler reconciles a Input object
type InputReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	ConstraintMapper map[string]Evaluator
}

//+kubebuilder:rbac:groups=core.deppy.io,resources=inputs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.deppy.io,resources=inputs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.deppy.io,resources=inputs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *InputReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	l.Info("reconciling request")
	defer l.Info("finished reconciling request")

	input := &deppyv1alpha1.Input{}
	if err := r.Get(ctx, req.NamespacedName, input); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	fmt.Printf("processing input %+v\n", input)
	constraints, err := r.EvaluateConstraints(input.Spec.Constraints)
	if err != nil {
		fmt.Printf("error: %+v\n", err)
	} else {
		fmt.Printf("solverConstraint: %+v\n", constraints)
	}
	id, propertyConstraints, err := r.EvaluateProperties(input.Spec.Properties)
	if err != nil {
		fmt.Printf("error: %+v\n", err)
	} else {
		fmt.Printf("id: %v, solverConstraint: %+v\n", id, propertyConstraints)
	}
	v := variable{id: string(id),}
	v.constraints = append(constraints, propertyConstraints...)
	fmt.Printf("variable: %+v\n", v)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *InputReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&deppyv1alpha1.Input{}).
		Complete(r)
}

type Evaluator interface {
	Evaluate(constraints map[string]string) (solver.Constraint, error)
}

func (r *InputReconciler) EvaluateConstraints(inputConstraints []deppyv1alpha1.Constraint) ([]solver.Constraint, error) {
	constraints := []solver.Constraint{} 
	for _, constraint := range inputConstraints {
		eval, ok := r.ConstraintMapper[constraint.Type]
		if !ok {
			return nil, errors.New("unknown constraint type")	
		}
		solverConstraint, err := eval.Evaluate(constraint.Value)
		if err != nil {
			fmt.Printf("error: %+v\n", err)
			return nil, fmt.Errorf("error: %+v\n", err)
		}
		fmt.Printf("solverConstraint: %+v\n", solverConstraint.String("subject"))
		constraints = append(constraints, solverConstraint) 
		
	}
	return constraints, nil

}

func InitConstraintMapper() map[string]Evaluator {
	return map[string]Evaluator {
		"deppy.package": &packageMapper,
	}
}

func (r *InputReconciler) EvaluateProperties(inputProperties []deppyv1alpha1.Property) (solver.Identifier, []solver.Constraint, error) {
	constraints := []solver.Constraint{}
	var identifier solver.Identifier
	for _, property := range inputProperties {
		eval, ok := r.ConstraintMapper[property.Type]
		if !ok {
			return "", nil, errors.New("unknown property type")	
		}
		solverConstraint, err := eval.Evaluate(property.Value)
		if err != nil {
			fmt.Printf("error: %+v\n", err)
			return "", nil, fmt.Errorf("error: %+v\n", err)
		}
		fmt.Printf("solverConstraint: %+v\n", solverConstraint.String("subject"))
		constraints = append(constraints, solverConstraint)
		for _, value := range property.Value {
			identifier = identifier + solver.Identifier("/" +value)
		}
		
	}
	return identifier, constraints, nil

}

// variable
type variable struct {
	id          string
	constraints []solver.Constraint
}

func (v *variable) Identifier() solver.Identifier {
	return solver.IdentifierFromString(v.id)
}

func (v *variable) Constraints() []solver.Constraint {
	return v.constraints
}


// Constraint Evaluators
type Package struct {
}

var packageMapper Package

func (p *Package) Evaluate(constraints map[string]string) (solver.Constraint, error){
	return solver.Mandatory(), nil
}

type Dependency struct {
}

var dependencyMapper Dependency

func (d *Dependency) Evaluate(constraints map[string]string) (solver.Constraint, error){
//	constraints["required"]
	return solver.Dependency(), nil
}


