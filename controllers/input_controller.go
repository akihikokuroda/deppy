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

	inputList := &deppyv1alpha1.InputList{}
	if err := r.List(ctx, inputList); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	fmt.Printf("processing input %+v\n", inputList)
	variables, err := r.EvaluateConstraints(inputList)
	if err != nil {
		fmt.Printf("error: %+v\n", err)
	} else {
		fmt.Printf("variables: %+v\n", variables)
	}

	for _, v := range variables { 
		fmt.Printf("variable: %+v\n", v)
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *InputReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&deppyv1alpha1.Input{}).
		Complete(r)
}

type Evaluator interface {
	Evaluate(constraint map[string]string, ids []string, properties []map[string]string, exclude int) ([]solver.Constraint, error)
}

func (r *InputReconciler) EvaluateConstraints(inputs *deppyv1alpha1.InputList) ([]solver.Variable, error) {
	ids := make([]string, len(inputs.Items))
	properties := make([]map[string]string, len(inputs.Items))
	for i, input := range inputs.Items {
		ids[i] = input.GetName()
		properties[i] = input.Spec.Properties
	}

	variables := []solver.Variable{}
	for currentInput, input := range inputs.Items {
		allConstraints := []solver.Constraint{}
		for _, constraint := range input.Spec.Constraints {
			eval, ok := r.ConstraintMapper[constraint.Type]
			if !ok {
				return nil, errors.New("unknown constraint type")	
			}
			constraints, err := eval.Evaluate(constraint.Value, ids, properties, currentInput)
			if err!= nil {
				return nil, fmt.Errorf("constraints evaluation error: %w", err)	
			}
			allConstraints = append(allConstraints, constraints...)
		}
		variable := variable{
			id: input.GetName(),
			constraints: allConstraints,
		}
		variables = append(variables, solver.Variable(&variable))
	}
	return variables, nil
}

func InitConstraintMapper() map[string]Evaluator {
	return map[string]Evaluator {
		"deppy.package":             &packageInstanceMapper,
	}
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

// Package Instance
type PackageInstance struct {
}

var packageInstanceMapper PackageInstance

func (p *PackageInstance) Evaluate(constraint map[string]string, ids []string, properties []map[string]string, exclude int) ([]solver.Constraint, error){
	for key, value := range constraint {
		fmt.Printf("constraint: %s:%s\n", key, value)
	}
	for i, id := range ids {
		fmt.Printf("id:%s\n", id)
		for key, value := range properties[i]{
			fmt.Printf("property[%v]: %s:%s\n", i, key, value)
		}
	}
	return nil, nil
}

