#!/usr/bin/env node
"use strict";
// `env: node` puts ESLint in CommonJS mode: parserOptions.ecmaFeatures.globalReturn
// becomes true, which (a) lets espree accept a top-level `return` and (b) makes
// eslint-scope nest a function scope over the Program. The second effect is the
// subtle one — top-level declarations land in that function scope, so they do not
// count as redeclarations of node's globals. `var crypto` below must therefore NOT
// be reported by no-redeclare, while the genuine duplicate below must be.
var crypto = require("crypto");
var duplicated = 1;
var duplicated = 2;

function helper(flag) {
    if (flag) {
        return "yes"
    }
    return "no";
}

if (helper(true) === "yes") {
    console.log(missingVariable);
}

module.exports = { crypto, duplicated, helper };
