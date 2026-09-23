package noeval

import (
	"testing"

	espree "github.com/jclyons52/espree-go"
)

func TestProbeScript(t *testing.T) {
	srcs := []string{
		`arguments;`,
		`function f() { return arguments.callee; }`,
		`function f() { return this.eval; }`,
		`var e = eval;`,
	}
	for _, src := range srcs {
		for _, st := range []string{"script", "module"} {
			_, err := espree.Parse(src, &espree.Options{SourceType: st, EcmaVersion: "latest"})
			t.Logf("[%s] %-45q err=%v", st, src, err)
		}
	}
}
