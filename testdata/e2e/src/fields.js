// Duplicate object keys, a debugger statement, an undeclared read and
// trailing whitespace — one file that trips every enabled rule.
const settings = {   
    mode: "fast",
    mode: "slow",
    get level() {
        return 3;
    },
    set level(value) {
        this._level = value;
    },
    level: 4
};

function run() {
    debugger;   
    return missingGlobal(settings);
}

console.log(run());
