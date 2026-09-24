// CommonJS-style source: with globalReturn in effect a top-level `return` is
// legal, and the top-level declarations below live in the function scope
// eslint-scope nests over the Program.
var path = require("path");

var first = 1;
var first = 2;

function unusedHelper(name) {
    return path.basename(name);
}

return { first: first };
