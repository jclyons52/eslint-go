package nounusedvars

import (
	"testing"

	"github.com/jclyons52/eslint-go/internal/ruletest"
)

// opts builds a single rule-options object.
func opts(m map[string]any) []any { return []any{m} }

// moduleParserOptions matches what the oracle driver applies to every case by
// default ({ecmaVersion: 2022, sourceType: "module"}). ruletest only forwards a
// case's SourceType/ECMAVersion to the oracle, so the Go side needs them
// spelled out in the config — without them every file would be analysed as an
// ES5 script, where the top-level scope is "global" instead of "module" (which
// changes `vars: "local"` and the switch/for scope shapes).
func moduleParserOptions() ruletest.Options {
	return ruletest.Options{ExtraConfig: map[string]any{
		"parserOptions": map[string]any{"sourceType": "module", "ecmaVersion": 2022},
	}}
}

// TestParity runs every case through real eslint@8.57.0 (the oracle) and this
// port, comparing the emitted messages exactly.
//
// Every case below is a behaviour taken from the JS rule (probed against the
// oracle, and from the rule's documented option semantics) — not from what this
// implementation happens to do.
//
// Four core gaps (none of them in this rule) are documented rather than papered
// over with a case that would fail for the wrong reason:
//
//  1. A class static block (`class A { static { var x = 1; } } new A();`) makes
//     the core's scope analysis recurse until the stack overflows, so it
//     crashes before create() runs. eslint-go's VisitorKeys omits "StaticBlock"
//     (eslint-visitor-keys has StaticBlock: ["body"]) and eslint-scope-go's
//     "iteration" fallback returns every key of the node map — including the
//     `parent` link the core's traverser attached — so the walk cycles between
//     a node and its parent. Oracle: one message, `'x' is assigned a value but
//     never used.` at 1:24. Same gap as rules/noalert, rules/noextrasemi,
//     rules/preferconst and rules/novar document.
//  2. `arguments` (and `eval`) cannot be used as an identifier *reference*:
//     acorn-go's checkUnreserved (expression.go) applies its isReservedStrict
//     list — which wrongly includes eval/arguments — to references, while real
//     acorn only applies those two to *bindings* (strictBind). So
//     `function f() { return arguments; }` (oracle: `'f' is defined but never
//     used.` at 1:10) and `var f = () => arguments; f();` (oracle: no messages)
//     both come back as a fatal "Parsing error: The keyword 'arguments' is
//     reserved" here. The rule's implicit-arguments carve-out is still covered
//     by `implicit-arguments-unused` below, which the parser accepts.
//  3. `/* global foo */` is implemented by ESLint's Linter (addDeclaredGlobals
//     + getDirectiveComments); the Go core only applies *configured* globals,
//     so no variable ever carries eslintExplicitGlobalComments and the rule's
//     second report branch (an unused inline global, reported at a loc inside
//     the comment) is unreachable. Oracle for `/* global foo */` alone:
//     `'foo' is defined but never used.` at 1:11. `global-comment-used` below
//     pins the half both implementations agree on.
//  4. The same applies to `/* exported foo */`, which in ESLint sets
//     variable.eslintUsed for a *global*-scope variable. It is the only
//     harness-reachable way to set that flag (no core rule calls
//     markVariableAsUsed), and the core does not implement the directive — so
//     the `eslintUsed` skip in collectUnusedVariables and the `posterior.
//     eslintUsed` check in isAfterLastUsedArg are ported faithfully but cannot
//     be exercised by a parity case here. In sourceType:module the directive is
//     a no-op on both sides anyway (the top-level binding lives in the module
//     scope), which `exported-comment-*` below documents.
func TestParity(t *testing.T) {
	ruletest.CompareWith(t, Rule, []ruletest.Case{
		// ---- default behaviour: unused declarations ------------------------
		{ID: "unused-var-assigned", Code: "var foo = 1;"},
		{ID: "unused-var-declared", Code: "var a;"},
		{ID: "unused-let-block", Code: "{ let a = 1; }"},
		{ID: "unused-const-fn", Code: "var a = function() {};"},
		{ID: "unused-function", Code: "function f() {}"},
		{ID: "fn-decl-in-block", Code: "{ function f() {} }"},
		{ID: "unused-class", Code: "class A {}"},
		{ID: "unused-arrow", Code: "var f = a => {}; f();"},
		{ID: "let-in-switch", Code: "switch (x) { case 1: let a = 1; }"},
		{ID: "try-block-let", Code: "try { let a = 1; } catch {}"},
		{ID: "empty-program", Code: ""},
		{ID: "label-not-a-variable", Code: "foo: { }"},
		{ID: "clean-file", Code: "function f(a) { return a; } f();"},
		{ID: "severity-warn", Code: "var foo = 1;", Config: map[string]any{"rules": map[string]any{Name: "warn"}}},
		{ID: "nested-block-shadow", Code: "var a = 1; { let a = 2; a; }"},

		// ---- used variables ------------------------------------------------
		{ID: "used-var", Code: "var foo = 1; foo();"},
		{ID: "used-before-definition", Code: "foo(); function foo() {}"},
		{ID: "closure-use", Code: "var a = 1; setTimeout(() => a);"},
		{ID: "shorthand-property", Code: "const a = 1; const o = { a };"},
		{ID: "template-literal", Code: "const s = `x`;"},
		{ID: "template-used", Code: "const s = `x`; s;"},
		{ID: "template-expression-use", Code: "const a = 1; const s = `${a}`; s;"},
		{ID: "member-expression-use", Code: "var a = 1; a.b = 2;"},
		{ID: "optional-chain-use", Code: "var a = 1; a?.b;"},
		{ID: "write-then-read-nested", Code: "var a; function g() { a = 1; } a; g();"},

		// ---- self-reference / function definitions -------------------------
		{ID: "self-reference-fn", Code: "function f() { f(); }"},
		{ID: "self-reference-in-nested", Code: "function f() { function g() { f(); } }"},
		{ID: "self-reference-var-fn", Code: "var f = function() { f(); };"},
		{ID: "self-reference-var-arrow", Code: "var f = () => f;"},
		{ID: "self-reference-returned-fn", Code: "function f() { return function() { f(); }; }"},
		{ID: "named-fn-expr-selfref", Code: "var f = function g() { g(); };"},
		{ID: "named-fn-expr-used", Code: "var f = function g() {}; f();"},
		{ID: "named-fn-expr-unused", Code: "var f = function g() {};"},
		{ID: "named-fn-expr-inner-var", Code: "var f = function g() { var x = 1; }; f();"},
		{ID: "recursive-arrow", Code: "var f = () => { f(); };"},
		{ID: "nested-arrow-unused-inner", Code: "var f = () => () => 1;"},
		{ID: "nested-arrow", Code: "var f = () => () => 1; f();"},

		// ---- write-only variables / last write reference -------------------
		{ID: "write-only", Code: "var foo = 1; foo = 2;"},
		{ID: "redeclaration", Code: "var foo = 1; var foo = 2;"},
		{ID: "sequence-expression", Code: "var a = 1; a = (a, 2);"},
		{ID: "logical-assign-or", Code: "var a = 1; a ||= 2;"},
		{ID: "logical-assign-and", Code: "var a = 1; a &&= 2;"},
		{ID: "logical-assign-nullish", Code: "var a = 1; a ??= 2;"},
		{ID: "update-expression", Code: "var a = 1; a++;"},
		{ID: "assign-self", Code: "var a = 1; a = a + 1;"},
		{ID: "assign-self-used", Code: "var a = 1; a = a + 1; console.log(a);"},
		{ID: "assign-a-plus-a", Code: "var a = 1; a = a + a;"},
		{ID: "assign-function-rhs", Code: "var a = 1; a = function() {};"},
		{ID: "storable-fn-in-call", Code: "var a = 1; a = foo(function() { return a; });"},
		{ID: "storable-fn-iife", Code: "var a = 1; a = (function() { return a; })();"},
		{ID: "storable-tagged-template", Code: "var a = 1; a = tag`x`;"},
		{ID: "storable-yield", Code: "function* f() { var a = 1; a = yield a; } f();"},
		{ID: "storable-new-expression", Code: "var a = 1; a = new Foo(a);"},
		{ID: "cross-scope-write", Code: "var a = 1; a = 2; function g() { a = 3; } g();"},
		{ID: "write-in-function-scope", Code: "var a = 1; function g() { var a = 2; a = 3; } g();"},
		{ID: "write-in-loop", Code: "for (;;) { var a = 1; a = 2; }"},
		{ID: "write-only-in-if", Code: "var a = 1; if (x) { a = 2; }"},
		{ID: "write-only-void", Code: "var a = 1; a = void 0;"},

		// ---- imports and exports -------------------------------------------
		{ID: "import-unused", Code: "import { a } from 'm';"},
		{ID: "import-used", Code: "import { a } from 'm'; a();"},
		{ID: "import-one-of-two-used", Code: "import { a, b } from 'm'; a();"},
		{ID: "import-default-unused", Code: "import a from 'm';"},
		{ID: "import-namespace-unused", Code: "import * as a from 'm';"},
		{ID: "import-namespace-member", Code: "import * as a from 'm'; a.b();"},
		{ID: "import-alias-unused", Code: "import { a as b } from 'm';"},
		{ID: "import-alias-used", Code: "import { a as b } from 'm'; b();"},
		{ID: "import-side-effect", Code: "import 'm';"},
		{ID: "export-fn", Code: "export function f() {}"},
		{ID: "export-var", Code: "export const a = 1;"},
		{ID: "export-default-fn", Code: "export default function f() {}"},
		{ID: "export-default-anon-fn", Code: "export default function() {}"},
		{ID: "export-default-class", Code: "export default class {}"},
		{ID: "export-default-named-class", Code: "export default class A {}"},
		{ID: "export-used", Code: "export function f() {} f();"},
		{ID: "export-list", Code: "var a = 1; export { a };"},
		{ID: "export-list-used", Code: "var a = 1; export { a }; a();"},
		{ID: "export-reexport", Code: "export { a } from 'm';"},
		{ID: "module-scope-fn-exported", Code: "function f() {} export { f };"},

		// ---- parameters: after-used (default), all, none -------------------
		{ID: "args-after-used-first", Code: "function f(a, b) { return b; }"},
		{ID: "args-after-used-last", Code: "function f(a, b) { return a; }"},
		{ID: "args-after-used-none", Code: "function f(a, b) { }"},
		{ID: "args-after-used-single", Code: "function f(a) { }"},
		{ID: "args-after-used-middle", Code: "function f(a, b, c) { return b; }"},
		{ID: "args-after-used-called", Code: "function f(a, b) { b(); }"},
		{ID: "args-destructured-after", Code: "function f(a, {b}) { return a; }"},
		{ID: "args-destructured-before", Code: "function f({b}, a) { return a; }"},
		{ID: "args-multiple-patterns", Code: "function f({a}, [b], c) {} f();"},
		{ID: "args-default-values", Code: "function f(a = 1, b = 2) { return b; }"},
		{ID: "args-default-uses-prev", Code: "function f(a, b = a) { return b; }"},
		{ID: "args-default-only", Code: "function f(a, b = a) {}"},
		{ID: "args-rest-only", Code: "function f(...args) {} f();"},
		{ID: "args-arrow-single", Code: "var f = a => {}; f();"},
		{ID: "args-arrow-parens", Code: "const f = (a) => {}; f();"},
		{ID: "args-iife", Code: "(function(a) {})();"},
		{ID: "args-async-fn", Code: "async function f(a) {} f();"},
		{ID: "args-generator-fn", Code: "function* f(a) {} f();"},
		{ID: "args-object-method", Code: "var o = { m(a) {} }; o.m();"},
		{ID: "args-class-method", Code: "class A { m(a) {} } new A();"},
		{ID: "args-class-method-after-used", Code: "class A { m(a, b) { return b; } } new A();"},
		{ID: "args-class-static-method", Code: "class A { static m(a) {} } new A();"},
		{ID: "args-setter-object", Code: "var o = { set foo(a) {} }; o.foo = 1;"},
		{ID: "args-setter-class", Code: "class A { set foo(a) {} } new A();"},
		{ID: "args-getter-no-arg", Code: "var o = { get foo() { return 1; } }; o.foo;"},
		{ID: "args-nested-function", Code: "function f(a) { function g(b) { return b; } return a; }"},
		{ID: "option-args-none", Code: "function f(a) { }", Options: opts(map[string]any{"args": "none"})},
		{ID: "option-args-none-two", Code: "function f(a, b) { return a; }", Options: opts(map[string]any{"args": "none"})},
		{ID: "option-args-all", Code: "function f(a, b) { return b; }", Options: opts(map[string]any{"args": "all"})},
		{ID: "option-args-all-rest", Code: "function f(...a) {} f();", Options: opts(map[string]any{"args": "all"})},
		{ID: "option-args-all-destructured", Code: "function f({a}) {} f();", Options: opts(map[string]any{"args": "all"})},
		{ID: "option-args-all-setter", Code: "var o = { set foo(a) {} }; o.foo = 1;", Options: opts(map[string]any{"args": "all"})},
		{ID: "option-args-ignore-pattern", Code: "function f(a) { }", Options: opts(map[string]any{"args": "all", "argsIgnorePattern": "^_"})},
		{ID: "option-args-ignore-pattern-hit", Code: "function f(_a) {} f();", Options: opts(map[string]any{"args": "all", "argsIgnorePattern": "^_"})},
		{ID: "option-args-pattern-message", Code: "function f(_a) {}", Options: opts(map[string]any{"args": "all", "argsIgnorePattern": "^_"})},

		// ---- catch parameters ----------------------------------------------
		{ID: "catch-default-none", Code: "try {} catch (e) { }"},
		{ID: "catch-used", Code: "try {} catch (e) { e(); }"},
		{ID: "catch-shadowed", Code: "var e = 1; try {} catch (e) {} e();"},
		{ID: "catch-optional-binding", Code: "try {} catch {}"},
		{ID: "catch-all", Code: "try {} catch (e) { }", Options: opts(map[string]any{"caughtErrors": "all"})},
		{ID: "catch-all-used", Code: "try {} catch (e) { e(); }", Options: opts(map[string]any{"caughtErrors": "all"})},
		{ID: "catch-all-destructured", Code: "try {} catch ({a}) {}", Options: opts(map[string]any{"caughtErrors": "all"})},
		{ID: "catch-all-nested", Code: "try {} catch (e) { try {} catch (e2) {} }", Options: opts(map[string]any{"caughtErrors": "all"})},
		{ID: "catch-ignore-pattern", Code: "try {} catch (e) { }", Options: opts(map[string]any{"caughtErrors": "all", "caughtErrorsIgnorePattern": "^_"})},
		{ID: "catch-ignore-pattern-hit", Code: "try {} catch (_e) {}", Options: opts(map[string]any{"caughtErrors": "all", "caughtErrorsIgnorePattern": "^_"})},
		{ID: "catch-ignore-pattern-message", Code: "try {} catch (e) { }", Options: opts(map[string]any{"caughtErrors": "all", "caughtErrorsIgnorePattern": "^_$"})},
		{ID: "catch-vars-ignore-pattern-applies", Code: "try {} catch (e) { }", Options: opts(map[string]any{"caughtErrors": "all", "varsIgnorePattern": "^e"})},

		// ---- rest siblings -------------------------------------------------
		{ID: "rest-siblings-off", Code: "var {a, ...rest} = obj; rest();"},
		{ID: "rest-siblings-on", Code: "var {a, ...rest} = obj; rest();", Options: opts(map[string]any{"ignoreRestSiblings": true})},
		{ID: "rest-siblings-both-unused", Code: "var {a, ...rest} = obj;", Options: opts(map[string]any{"ignoreRestSiblings": true})},
		{ID: "rest-sibling-reference", Code: "var {a, ...rest} = obj; a;", Options: opts(map[string]any{"ignoreRestSiblings": true})},
		{ID: "rest-sibling-array", Code: "var [a, ...rest] = obj; rest();", Options: opts(map[string]any{"ignoreRestSiblings": true})},
		{ID: "rest-sibling-array-both-unused", Code: "var [a, ...rest] = obj;", Options: opts(map[string]any{"ignoreRestSiblings": true})},
		{ID: "rest-sibling-param", Code: "function f({a, ...rest}) { rest(); } f();", Options: opts(map[string]any{"ignoreRestSiblings": true})},
		{ID: "rest-sibling-assignment", Code: "({a, ...rest} = obj);", Options: opts(map[string]any{"ignoreRestSiblings": true})},

		// ---- destructuring -------------------------------------------------
		{ID: "destructure-nested", Code: "var {a: {b}} = obj;"},
		{ID: "destructure-nested-used", Code: "var {a: {b}} = obj; b();"},
		{ID: "destructure-default", Code: "var {a = 1} = obj;"},
		{ID: "destructure-default-used", Code: "var {a = 1} = obj; a();"},
		{ID: "array-destructure", Code: "var [a] = xs;"},
		{ID: "array-destructure-default", Code: "var [a = 1] = xs;"},
		{ID: "array-pattern-ignore-miss", Code: "var [a, b] = xs;", Options: opts(map[string]any{"destructuredArrayIgnorePattern": "^_"})},
		{ID: "array-pattern-ignore-hit", Code: "var [_a, b] = xs;", Options: opts(map[string]any{"destructuredArrayIgnorePattern": "^_"})},
		{ID: "array-pattern-ignore-read", Code: "var [a] = xs; a;", Options: opts(map[string]any{"destructuredArrayIgnorePattern": "^_"})},
		{ID: "array-pattern-ignore-read-hit", Code: "var [_a] = xs; _a;", Options: opts(map[string]any{"destructuredArrayIgnorePattern": "^_"})},
		{ID: "array-pattern-ignore-object", Code: "var {a} = xs;", Options: opts(map[string]any{"destructuredArrayIgnorePattern": "^_"})},
		{ID: "array-pattern-ignore-nested", Code: "var [[_a], b] = xs;", Options: opts(map[string]any{"destructuredArrayIgnorePattern": "^_"})},
		{ID: "array-pattern-ignore-param", Code: "function f([_a]) {} f();", Options: opts(map[string]any{"destructuredArrayIgnorePattern": "^_"})},
		{ID: "array-pattern-ignore-param-miss", Code: "function f([a]) {} f();", Options: opts(map[string]any{"destructuredArrayIgnorePattern": "^_"})},
		{ID: "array-pattern-ignore-for-of", Code: "for (const [_a] of xs) {}", Options: opts(map[string]any{"destructuredArrayIgnorePattern": "^_"})},

		// ---- for-in / for-of -----------------------------------------------
		{ID: "for-of-const", Code: "for (const x of xs) { }"},
		{ID: "for-in-const", Code: "for (const k in obj) { }"},
		{ID: "for-of-empty-body", Code: "for (const x of xs) {}"},
		{ID: "for-of-return-block", Code: "function f() { for (const x of xs) { return; } }"},
		{ID: "for-in-return-block", Code: "function f() { for (const k in obj) { return; } }"},
		{ID: "for-of-var-return", Code: "function f() { for (var x of xs) { return; } }"},
		{ID: "for-of-return-no-block", Code: "function f() { for (const x of xs) return; }"},
		{ID: "for-of-used", Code: "for (const x of xs) { x(); }"},
		{ID: "for-of-destructured", Code: "for (const [a] of xs) { }"},
		{ID: "for-of-let-in-block", Code: "for (let i = 0; ; ) { }"},
		{ID: "for-let-used", Code: "for (let i = 0; i < 1; i++) { }"},
		{ID: "switch-let-used", Code: "switch (x) { case 1: let a = 1; a; }"},

		// ---- classes -------------------------------------------------------
		{ID: "class-expression-name", Code: "const x = class Foo {}; x();"},
		{ID: "class-expression-name-unused", Code: "const x = class Foo {};"},
		{ID: "class-field", Code: "class A { x = 1; } new A();"},
		{ID: "class-static-field", Code: "class A { static x = 1; } new A();"},
		{ID: "class-field-uses-outer", Code: "const a = 1; class A { x = a; } new A();"},
		{ID: "class-method-inner-var", Code: "class A { m() { var y = 1; } } new A();"},
		// NOTE (core gap 1): {ID: "class-static-block", Code: "class A { static { var x = 1; } } new A();"}
		// cannot run — the core's scope analysis stack-overflows on any source
		// containing `static { … }`. See the package comment above.
		{ID: "class-method-this", Code: "class A { m() {} } new A();"},

		// ---- the vars option -----------------------------------------------
		{ID: "option-vars-local", Code: "var foo = 1;", Options: opts(map[string]any{"vars": "local"})},
		{ID: "option-vars-local-fn", Code: "function g() { var foo = 1; }", Options: opts(map[string]any{"vars": "local"})},
		{ID: "option-vars-local-nested-block", Code: "{ var a = 1; }", Options: opts(map[string]any{"vars": "local"})},
		{ID: "option-vars-local-class", Code: "class A {}", Options: opts(map[string]any{"vars": "local"})},
		{ID: "option-vars-all", Code: "var foo = 1;", Options: opts(map[string]any{"vars": "all"})},
		{ID: "option-vars-string-local", Code: "var foo = 1;", Options: []any{"local"}},
		{ID: "option-vars-string-all", Code: "function g() { var foo = 1; }", Options: []any{"all"}},

		// ---- ignore patterns -----------------------------------------------
		{ID: "vars-ignore-pattern", Code: "var _foo = 1;", Options: opts(map[string]any{"varsIgnorePattern": "^_"})},
		{ID: "vars-ignore-pattern-declared", Code: "var _foo;", Options: opts(map[string]any{"varsIgnorePattern": "^_"})},
		{ID: "vars-ignore-pattern-fn", Code: "function _foo() {}", Options: opts(map[string]any{"varsIgnorePattern": "^_"})},
		{ID: "vars-ignore-pattern-write-only", Code: "var _foo = 1; _foo = 2;", Options: opts(map[string]any{"varsIgnorePattern": "^_"})},
		{ID: "vars-ignore-pattern-message", Code: "var foo = 1;", Options: opts(map[string]any{"varsIgnorePattern": "^_"})},
		{ID: "vars-ignore-pattern-message-declared", Code: "var foo;", Options: opts(map[string]any{"varsIgnorePattern": "^_"})},
		{ID: "both-patterns-message", Code: "var foo = 1;", Options: opts(map[string]any{"varsIgnorePattern": "^_", "argsIgnorePattern": "^x"})},

		// ---- arguments -----------------------------------------------------
		// NOTE (core gap 2): {ID: "implicit-arguments", Code: "function f() { return arguments; }"}
		// and {ID: "arguments-not-in-arrow", Code: "var f = () => arguments; f();"} are
		// unparseable here — acorn-go rejects `arguments` as a reference in strict mode.
		{ID: "implicit-arguments-unused", Code: "function f() { }"},
		{ID: "fn-decl-inner-var-used", Code: "function f() { var a = 1; return a; }"},

		// ---- globals -------------------------------------------------------
		{ID: "globals-used", Code: "foo();", Config: map[string]any{"globals": map[string]any{"foo": "readonly"}}},
		{ID: "globals-unused-not-reported", Code: "var bar = 1; bar();", Config: map[string]any{"globals": map[string]any{"foo": "readonly"}}},
		{ID: "global-comment-used", Code: "/* global foo */\nfoo();"},
		// NOTE (core gap 3): {ID: "global-comment-unused", Code: "/* global foo */"} would
		// need the core to implement inline global comments; see the package comment.
		{ID: "exported-comment-module-scope", Code: "/* exported foo */\nvar foo = 1;"},
		{ID: "exported-comment-exported-var", Code: "/* exported foo */\nexport var foo = 1;"},

		// ---- non-ASCII (UTF-16 column space) -------------------------------
		{ID: "non-ascii-unused", Code: "const café = 1;"},
		{ID: "non-ascii-used", Code: "const café = 1; café();"},
		{ID: "non-ascii-mixed", Code: "const é = 1; const b = 2;"},
		{ID: "non-ascii-param", Code: "function f(café) {} f();"},
	}, moduleParserOptions())
}
