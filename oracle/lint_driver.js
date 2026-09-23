'use strict';
// lint_driver.js — the JS oracle for rule-level and file-level parity.
//
// Reads a corpus JSON ({ cases: [{ id, code, config, ecmaVersion, sourceType,
// fix }] }), runs the REAL ESLint Linter from the vendored eslint@8.57.0 over
// each case, and prints { cases: [{ id, messages, output, fixed }] }.
//
// `fix: true` uses Linter.verifyAndFix (so --fix behaviour, including the
// 10-pass loop and fix merging, is compared); otherwise Linter.verify.
const fs = require('fs');

const eslintDir = process.env.ESLINT_ORACLE || './node_modules/eslint';
const { Linter } = require(eslintDir);

const corpus = JSON.parse(fs.readFileSync(process.argv[2], 'utf8'));
const linter = new Linter();
const out = { cases: [] };

for (const c of corpus.cases) {
    const config = Object.assign({}, c.config || {});
    config.parserOptions = Object.assign(
        { ecmaVersion: c.ecmaVersion || 2022, sourceType: c.sourceType || 'module' },
        config.parserOptions || {}
    );
    let messages, output = null, fixed = null;
    if (c.fix) {
        const result = linter.verifyAndFix(c.code, config);
        messages = result.messages;
        output = result.output;
        fixed = result.fixed;
    } else {
        messages = linter.verify(c.code, config);
    }
    out.cases.push({ id: c.id, messages, output, fixed });
}

process.stdout.write(JSON.stringify(out));
