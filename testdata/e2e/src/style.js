// Style + correctness problems: the rules that rewrite code are the ones
// where --fix parity is hardest, so this file is aimed at them.
const a = 1
var b = 2
let c = 3
if (a == b) {
  b = 0
}
if (a === b) c = 4;


const obj = { 'key': 'single-quoted' };
const arr = new Array(1, 2, 3);
const o2 = new Object();
const s = new String('x');
const hole = [1, , 3];
const d = .5;
const re = /a  b/;
const bad = !!true;
throw 'a string';

function shadowed(shadowed) {
  if (shadowed) { return 1 +  2; }
  else { return 3; }
}

function compare() {
  if (a === a) { return 1; }
  if (typeof a === 'number') { return 2; }
  if (Number.isNaN(a) == true) { return 3; }
  switch (a) {
    case NaN:
      return 4;
    case 1:
      return 5;
    case 1:
      return 6;
  }
  return 0;
}

export { a, b, c, obj, arr, o2, s, hole, d, re, bad, shadowed, compare };   
