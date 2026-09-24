const assert = require("node:assert/strict")
const lyrics = require("../Lyrics.js")

const parsed = lyrics.parse(
  "[00:10.50]First line\n[00:20.00][00:30.00]Repeated\n[offset:0]ignored",
  "[00:10.50]第一行\n[00:30.00]重复"
)

assert.deepEqual(parsed, [
  { time: 10.5, text: "First line", translated: "第一行" },
  { time: 20, text: "Repeated", translated: "" },
  { time: 30, text: "Repeated", translated: "重复" }
])
assert.equal(lyrics.currentIndex(parsed, 9.9), -1)
assert.equal(lyrics.currentIndex(parsed, 10.5), 0)
assert.equal(lyrics.currentIndex(parsed, 29.9), 1)
assert.equal(lyrics.currentIndex(parsed, 99), 2)

console.log("lyrics tests passed")
