package novar

import (
	"fmt"
	"testing"

	eslint "github.com/jclyons52/eslint-go"
)

func TestProbe2(t *testing.T) {
	codes := []string{
		"class C { static { var x = 1; } }",
		"class C { static { let x = 1; } }",
		"class C { static { } }",
		"class C { static { foo(); } }",
		"class C { static { var x; } }",
	}
	for _, code := range codes {
		func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("PANIC %q: %v\n", code, r)
				}
			}()
			linter := eslint.NewLinter()
			cfg, err := eslint.ParseConfigJSON([]byte(`{"rules":{}}`))
			if err != nil {
				t.Fatal(err)
			}
			msgs := linter.Verify(code, cfg, "case.js")
			fmt.Printf("OK %q -> %d msgs\n", code, len(msgs))
		}()
	}
}
