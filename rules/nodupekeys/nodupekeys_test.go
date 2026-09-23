package nodupekeys

import (
	"testing"

	"github.com/jclyons52/eslint-go/internal/ruletest"
)

func TestParity(t *testing.T) {
	ruletest.Compare(t, Rule, []ruletest.Case{
		{ID: "basic", Code: "var o = { a: 1, a: 2 };"},
		{ID: "string-keys", Code: "var o = { 'b': 1, b: 2 };"},
		{ID: "get-set", Code: "var o = { get x() {}, set x(v) {} };"},
		{ID: "get-get", Code: "var o = { get x() {}, get x() {} };"},
		{ID: "init-get", Code: "var o = { x: 1, get x() {} };"},
		{ID: "computed", Code: "var k = 'a'; var o = { [k]: 1, [k]: 2 };"},
		{ID: "numbers", Code: "var o = { 1: 'a', 1: 'b' };"},
		{ID: "numeric-string-same", Code: "var o = { 1: 'a', '1': 'b' };"},
		{ID: "nested", Code: "var o = { a: { x: 1, x: 2 }, a: 3 };"},
		{ID: "nested-shadow", Code: "var o = { a: { x: 1 }, b: { x: 2 } };"},
		{ID: "destructuring", Code: "function f({ a, a: b }) { return a + b; }"},
		{ID: "methods", Code: "var o = { foo() {}, foo() {} };"},
		{ID: "bigint-keys", Code: "var o = { 1n: 1, '1': 2 };"},
		{ID: "template-key", Code: "var o = { [`a`]: 1, a: 2 };"},
		{ID: "clean", Code: "var o = { a: 1, b: 2, c: 3 };"},
		{ID: "spread-no-crash", Code: "var extra = { z: 1 }; var o = { ...extra, a: 1, a: 2 };"},
		{ID: "class-body-unaffected", Code: "class C { m() {} m() {} }"},
	})
}
