// Sloppy-mode script: this file only parses with sourceType "script" — a module
// is strict, so duplicate parameter names, `with`, legacy octal literals and
// `arguments.callee` would all be rejected there. It exists to keep the parser's
// script mode honest end to end.
function dup(a, a) {
  return a;
}

var octal = 0755;

function withStatement(obj) {
  with (obj) {
    return obj.value;
  }
}

function callSelf() {
  return arguments.callee;
}

if (octal == dup(1, 2)) {
  debugger;
}

function empty() {}

console.log(octal, withStatement, callSelf, empty);
