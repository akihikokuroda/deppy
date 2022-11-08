package sat

import (
	"fmt"
	"io"

	pkgsat "github.com/operator-framework/deppy/pkg/sat"
)

type ISearchPosition interface {
	Variables() []pkgsat.IVariable
	Conflicts() []pkgsat.IAppliedConstraint
}

type ITracer interface {
	Trace(p ISearchPosition)
}

type IDefaultTracer struct{}

func (IDefaultTracer) Trace(_ ISearchPosition) {
}

type ILoggingTracer struct {
	Writer io.Writer
}

func (t ILoggingTracer) Trace(p ISearchPosition) {
	fmt.Fprintf(t.Writer, "---\nAssumptions:\n")
	for _, i := range p.Variables() {
		fmt.Fprintf(t.Writer, "- %s\n", i.Identifier())
	}
	fmt.Fprintf(t.Writer, "Conflicts:\n")
	for _, a := range p.Conflicts() {
		fmt.Fprintf(t.Writer, "- %s\n", a)
	}
}
