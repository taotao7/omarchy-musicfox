function parseTimestamp(minutes, seconds) {
  return Number(minutes) * 60 + Number(seconds)
}

function parse(lrc, translated) {
  var translations = parseSingle(translated)
  var translatedByTime = {}
  for (var i = 0; i < translations.length; i++)
    translatedByTime[Math.round(translations[i].time * 100)] = translations[i].text

  var lines = parseSingle(lrc)
  for (var j = 0; j < lines.length; j++)
    lines[j].translated = translatedByTime[Math.round(lines[j].time * 100)] || ""
  return lines
}

function parseSingle(value) {
  var result = []
  var input = String(value || "").split(/\r?\n/)
  var timestamp = /\[(\d+):(\d+(?:\.\d+)?)\]/g
  for (var i = 0; i < input.length; i++) {
    var line = input[i]
    var matches = []
    var match
    timestamp.lastIndex = 0
    while ((match = timestamp.exec(line)) !== null)
      matches.push(parseTimestamp(match[1], match[2]))
    var text = line.replace(timestamp, "").trim()
    if (!text) continue
    for (var j = 0; j < matches.length; j++)
      result.push({ time: matches[j], text: text, translated: "" })
  }
  result.sort(function(a, b) { return a.time - b.time })
  return result
}

function currentIndex(lines, position) {
  var input = lines || []
  var target = Number(position) || 0
  var selected = -1
  for (var i = 0; i < input.length; i++) {
    if (input[i].time > target) break
    selected = i
  }
  return selected
}

if (typeof module !== "undefined") {
  module.exports = { parse: parse, currentIndex: currentIndex }
}
