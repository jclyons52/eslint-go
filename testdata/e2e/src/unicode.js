// Non-ASCII text: reported columns and fix ranges are UTF-16 code-unit based in
// ESLint, byte based inside this port (see units.go). This file is here to keep
// that boundary honest.
const greeting = "こんにちは";
const emoji = "🎉🎉";  
const café = { name: "café" };

export function describe() {
    return `${greeting} ${emoji} ${café.name}`;
}
