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
	semver "github.com/blang/semver/v4"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	
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
	variables, err := r.EvaluateConstraints(inputList)
	if err != nil {
		fmt.Printf("error: %+v\n", err)
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
		"deppy.package":                          &packageInstanceMapper,
		"core.deppy.io/v1alpha1/Mandatory":       &mandatoryMapper,
		"core.deppy.io/v1alpha1/RequireKeyValue": &requireKeyValueMapper,
		"core.deppy.io/v1alpha1/Unique":          &uniqueMapper,
		"core.deppy.io/v1alpha1/ConflictPackage": &conflictPackageMapper,
		"core.deppy.io/v1alpha1/RequirePackage":  &requirePackageMapper,
		"core.deppy.io/v1alpha1/RequireFilterJS": &requireFilterCelMapper,
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
	fmt.Printf("==============================\n")
	fmt.Printf("ID: %v\n", exclude)
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

// Mandatory
type Mandatory struct {
}

var mandatoryMapper Mandatory

func (m *Mandatory) Evaluate(constraint map[string]string, ids []string, properties []map[string]string, exclude int) ([]solver.Constraint, error){
	fmt.Printf("Mandatory\n")
	return []solver.Constraint{solver.Mandatory()}, nil
}

// Require Package
type RequirePackage struct {
}

var requirePackageMapper RequirePackage

func (r *RequirePackage) Evaluate(constraint map[string]string, ids []string, properties []map[string]string, exclude int) ([]solver.Constraint, error){
	fmt.Printf("RequirePackage\n")
	onever, _ := semver.Make(constraint["core.deppy.io/v1alpha1/Version"])
	verrange, _ := semver.ParseRange(constraint["core.deppy.io/v1alpha1/Version"])
	require := []solver.Identifier{}
	for i, id := range ids {
		var pkg string
		var vars semver.Version
		property := properties[i]
		if s, ok := property["packageName"]; ok {
			pkg = s
		}
		if s, ok := property["versionRange"]; ok {
			vars, _ = semver.Make(s)
		}
		
		if constraint["core.deppy.io/v1alpha1/Package"] == pkg && (onever.Compare(vars) == 0 || (verrange != nil && verrange(vars))) {
			require = append(require, solver.Identifier(id))
		}
	}
	return []solver.Constraint{solver.Dependency(require...)}, nil
}

// Conflict Package
type ConflictPackage struct {
}

var conflictPackageMapper ConflictPackage

func (r *ConflictPackage) Evaluate(constraint map[string]string, ids []string, properties []map[string]string, exclude int) ([]solver.Constraint, error){
	fmt.Printf("RequirePackage\n")
	onever, _ := semver.Make(constraint["core.deppy.io/v1alpha1/Version"])
	verrange, _ := semver.ParseRange(constraint["core.deppy.io/v1alpha1/Version"])
	conflict := []solver.Constraint{}
	for i, id := range ids {
		var pkg string
		var vars semver.Version
		property := properties[i]
		if s, ok := property["packageName"]; ok {
			pkg = s
		}
		if s, ok := property["versionRange"]; ok {
			vars, _ = semver.Make(s)
		}
		
		if constraint["core.deppy.io/v1alpha1/Package"] == pkg && (onever.Compare(vars) == 0 || (verrange != nil && verrange(vars))) {
			conflict = append(conflict, solver.Conflict(solver.Identifier(id)))
		}
	}
	return conflict, nil
}

// Require Key Value
type RequireKeyValue struct {
}

var requireKeyValueMapper RequireKeyValue

func (r *RequireKeyValue) Evaluate(constraint map[string]string, ids []string, properties []map[string]string, exclude int) ([]solver.Constraint, error){
	fmt.Printf("RequireKeyValue\n")
	require := []solver.Identifier{}
	for i, id := range ids {
		property := properties[i]
		if value, ok := property[constraint["key"]]; ok {
			if value == constraint["value"] {
				require = append(require, solver.Identifier(id))
			}
		}
	}
	return []solver.Constraint{solver.Dependency(require...)}, nil
}

// Unique
type Unique struct {
}

var uniqueMapper Unique

func (r *Unique) Evaluate(constraint map[string]string, ids []string, properties []map[string]string, exclude int) ([]solver.Constraint, error){
	fmt.Printf("Unique\n")
	unique := []solver.Constraint{}
	valueToIDList := map[string][]solver.Identifier{}
	for i, id := range ids {
		property := properties[i]
		if value, ok := property[constraint["key"]]; ok {
			iDList, ok := valueToIDList[value];
			if !ok {
				iDList = []solver.Identifier{}
				valueToIDList[value] = iDList
			}
			iDList = append(iDList, solver.Identifier(id))
		}
	}
	for _, list := range valueToIDList {
		unique = append(unique, solver.AtMost(1, list...)) 
	}
	return unique, nil
}

// Require Filter Cel
type RequireFilterCel struct {
}

var requireFilterCelMapper RequireFilterCel

func (r *RequireFilterCel) Evaluate(constraint map[string]string, ids []string, properties []map[string]string, exclude int) ([]solver.Constraint, error){
	fmt.Printf("RequireFilterCel\n")

	d := cel.Declarations(decls.NewVar("property", decls.NewMapType(decls.String, decls.String)))
	env, err := cel.NewEnv(d)
	if err != nil {
		return nil, err
	}
	ast, iss := env.Compile(constraint["filterFunc"])
	if iss.Err() != nil {
		return nil, iss.Err() 
	}
	prg, err := env.Program(ast)
	if err != nil {
		return nil, err
	}

	require := []solver.Identifier{}
	for i, id := range ids {
		out, _, err := prg.Eval(map[string]interface{}{"property": properties[i]})
		fmt.Printf("*****  %+v\n", out)
		if err != nil {
			return nil, err
		}
		if out != nil {
			require = append(require, solver.Identifier(id))
		}
	}
	return []solver.Constraint{solver.Dependency(require...)}, nil
}
