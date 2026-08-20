'use strict';
// eslint-go oracle driver.
// Reads a corpus JSON (argv[2]) of { cases: [ {id, code, config, ecmaVersion,
// sourceType} ] }, parses each with espree, runs real Linter().verify, and
// outputs { cases: [ {id, tree, messages} ] }.
const fs = require('fs');
const { Linter } = require('eslint');
const espree = require('espree');

const linter = new Linter();
const corpus = JSON.parse(fs.readFileSync(process.argv[2], 'utf8'));
const out = { cases: [] };
for (const c of corpus.cases) {
  const parseOpts = {
    ecmaVersion: c.ecmaVersion !== undefined ? c.ecmaVersion : 2022,
    sourceType: c.sourceType || 'script',
    range: true,
    loc: true,
    comment: true
  };
  const tree = espree.parse(c.code, parseOpts);
  const config = Object.assign({}, c.config || {});
  config.parserOptions = Object.assign({}, config.parserOptions || {}, parseOpts);
  const messages = linter.verify(c.code, config);
  out.cases.push({ id: c.id, tree, messages });
}
console.log(JSON.stringify(out));
