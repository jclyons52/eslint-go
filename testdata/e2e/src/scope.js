// Scope-sensitive problems: unused bindings, shadowing, redeclaration.
var globalVar = 1;
var globalVar = 2;

const unusedTop = 1;

function outer(usedParam, unusedParam) {
  const local = 1;
  var hoisted = 2;
  { let shadow = 3; }
  return usedParam + local;
}

function shadowsGlobal() {
  var globalVar = 9;
  return globalVar;
}

for (var i = 0; i < 3; i++) { outer(i, i); }

export { outer, shadowsGlobal };
