import QtQuick
import Quickshell
import Quickshell.Io
import "I18n.js" as I18n
import "Lyrics.js" as Lyrics

Item {
  id: root

  property var shell: null
  property bool ready: false
  property bool loggedIn: false
  property bool busy: false
  property bool mprisEnabled: false
  property string lastError: ""
  property string uiLocale: I18n.resolve("Auto", Quickshell.env("LANG"))
  property string activeView: "home"
  property string viewTitle: I18n.text(uiLocale, "home")
  property string currentQuery: ""
  property string currentPlaylistName: ""
  property var profile: ({})
  property var tracks: []
  property var playlists: []
  property var playback: ({
    track: {}, queue: [], index: -1, playing: false, position: 0,
    duration: 0, volume: 70, mode: "list", liked: false, idle: true
  })
  property string lyrics: ""
  property string translatedLyrics: ""
  property var lyricLines: []
  property string qrImage: ""
  property int qrCode: 0
  property string qrMessage: ""
  property int requestSequence: 0

  readonly property string helperPath: (Quickshell.env("XDG_CONFIG_HOME") ||
    (Quickshell.env("HOME") + "/.config")) +
    "/omarchy/plugins/taotao7.musicfox/bin/omarchy-musicfox-helper"
  readonly property var currentTrack: playback && playback.track ? playback.track : ({})
  readonly property bool hasTrack: currentTrack && String(currentTrack.id || "") !== ""
  readonly property string title: hasTrack ? String(currentTrack.name || "") : "Musicfox"
  readonly property string artist: hasTrack ? String(currentTrack.artists || "") : t("music")

  signal dataChanged()

  function t(key) { return I18n.text(uiLocale, key) }
  onUiLocaleChanged: refreshViewTitle()

  function refreshViewTitle() {
    if (activeView === "search") viewTitle = t("searchTitle") + currentQuery
    else if (activeView === "library") viewTitle = t("library")
    else if (activeView === "toplists") viewTitle = t("toplists")
    else if (activeView === "playlist") viewTitle = currentPlaylistName || t("playlist")
    else if (activeView === "queue") viewTitle = t("queue")
    else viewTitle = t("home")
  }

  function send(command, args) {
    if (!helper.running || !helper.stdinEnabled) {
      lastError = t("serviceNotReady")
      return ""
    }
    requestSequence += 1
    var id = "qml-" + requestSequence
    busy = true
    helper.write(JSON.stringify({id: id, command: command, args: args || {}}) + "\n")
    return id
  }

  function loginWithCookie(cookie) { return send("loginCookie", {cookie: cookie}) }
  function startQrLogin() {
    qrCode = 0
    qrMessage = t("qrCreating")
    return send("loginQrStart", {})
  }
  function logout() { return send("logout", {}) }
  function loadHome() {
    activeView = "home"
    viewTitle = t("home")
    return send("home", {})
  }
  function search(query) {
    if (!String(query).trim()) return
    activeView = "search"
    currentQuery = String(query).trim()
    viewTitle = t("searchTitle") + currentQuery
    return send("search", {query: String(query).trim()})
  }
  function loadLibrary() {
    activeView = "library"
    viewTitle = t("library")
    return send("library", {})
  }
  function loadToplists() {
    activeView = "toplists"
    viewTitle = t("toplists")
    return send("toplists", {})
  }
  function loadPlaylist(playlist) {
    activeView = "playlist"
    currentPlaylistName = playlist && playlist.name ? playlist.name : ""
    viewTitle = currentPlaylistName || t("playlist")
    return send("playlist", {id: String(playlist.id)})
  }
  function playPersonalFm() { return send("personalFm", {}) }
  function showQueue() {
    activeView = "queue"
    viewTitle = t("queue")
    tracks = playback && playback.queue ? playback.queue : []
    playlists = []
    dataChanged()
  }
  function playTracks(items, index) { return send("play", {tracks: items, index: index}) }
  function playQueueIndex(index) { return send("playIndex", {index: index}) }
  function toggle() { return send("toggle", {}) }
  function next() { return send("next", {}) }
  function previous() { return send("previous", {}) }
  function seek(position) { return send("seek", {position: position}) }
  function setVolume(volume) { return send("volume", {volume: volume}) }
  function cycleMode() {
    var nextMode = playback.mode === "list" ? "repeat" :
      (playback.mode === "repeat" ? "shuffle" : "list")
    return send("mode", {mode: nextMode})
  }
  function toggleLike() { return send("like", {liked: !Boolean(playback.liked)}) }
  function setQuality(quality) { return send("quality", {quality: quality}) }
  function setMpris(enabled) {
    mprisEnabled = Boolean(enabled)
    return ready ? send("mpris", {enabled: mprisEnabled}) : ""
  }

  function formatDuration(seconds) {
    var value = Math.max(0, Math.floor(Number(seconds) || 0))
    var minutes = Math.floor(value / 60)
    var remainder = value % 60
    return minutes + ":" + (remainder < 10 ? "0" : "") + remainder
  }

  function currentLyricIndex() {
    return Lyrics.currentIndex(lyricLines, playback.position)
  }

  function handleLine(line) {
    if (!String(line).trim()) return
    var message
    try {
      message = JSON.parse(line)
    } catch (error) {
      lastError = t("parseError")
      return
    }
    var data = message.data || {}
    if (message.event === "ready") {
      ready = true
      busy = false
      loggedIn = Boolean(data.loggedIn)
      profile = data.profile || {}
      applyPlayback(data.playback)
      send("mpris", {enabled: mprisEnabled})
      if (loggedIn) loadHome()
    } else if (message.event === "response") {
      busy = false
      applyResponse(data.kind, data.data)
    } else if (message.event === "playback") {
      applyPlayback(data)
    } else if (message.event === "lyrics") {
      if (!hasTrack || String(data.trackId || "") === String(currentTrack.id || "")) {
        lyrics = String(data.lrc || "")
        translatedLyrics = String(data.translated || "")
        lyricLines = Lyrics.parse(lyrics, translatedLyrics)
      }
    } else if (message.event === "qr") {
      qrCode = Number(data.code || 0)
      qrMessage = String(data.message || "")
      if (data.image) qrImage = String(data.image)
    } else if (message.event === "login") {
      profile = data || {}
      loggedIn = Boolean(profile.userId)
      qrCode = 803
      qrMessage = t("qrSuccess")
      loadHome()
    } else if (message.event === "error" || message.event === "fatal") {
      busy = false
      lastError = String(data.message || t("unknownError"))
    }
  }

  function applyResponse(kind, data) {
    if (kind === "qr") {
      qrCode = Number(data.code || 0)
      qrMessage = String(data.message || "")
      qrImage = String(data.image || "")
    } else if (kind === "login") {
      profile = data || {}
      loggedIn = Boolean(profile.userId)
      if (loggedIn) loadHome()
    } else if (kind === "logout") {
      loggedIn = false
      profile = ({})
      tracks = []
      playlists = []
      activeView = "home"
    } else if (kind === "home") {
      tracks = data && data.songs ? data.songs : []
      playlists = data && data.playlists ? data.playlists : []
      dataChanged()
    } else if (kind === "search" || kind === "playlist") {
      tracks = data || []
      playlists = []
      dataChanged()
    } else if (kind === "library" || kind === "toplists") {
      playlists = data || []
      tracks = []
      dataChanged()
    } else if (kind === "personalFm") {
      applyPlayback(playback)
    } else if (kind === "play" || kind === "playback") {
      applyPlayback(data)
    } else if (kind === "like") {
      var next = Object.assign({}, playback)
      next.liked = Boolean(data.liked)
      playback = next
    }
  }

  function applyPlayback(data) {
    if (!data) return
    var previousTrackId = currentTrack ? String(currentTrack.id || "") : ""
    playback = data
    var nextTrackId = data.track ? String(data.track.id || "") : ""
    if (nextTrackId !== previousTrackId) {
      lyrics = ""
      translatedLyrics = ""
      lyricLines = []
    }
    if (activeView === "queue") tracks = data.queue || []
  }

  Process {
    id: helper
    command: [root.helperPath]
    running: true
    stdinEnabled: true
    stdout: SplitParser {
      splitMarker: "\n"
      onRead: data => root.handleLine(data)
    }
    stderr: SplitParser {
      splitMarker: "\n"
      onRead: data => {
        if (String(data).trim()) root.lastError = String(data).trim()
      }
    }
    onExited: function(exitCode, exitStatus) {
      root.ready = false
      root.busy = false
      if (exitCode !== 0) root.lastError = root.t("serviceExited") + " (" + exitCode + ")"
      restartTimer.start()
    }
  }

  Timer {
    id: restartTimer
    interval: 2000
    onTriggered: if (!helper.running) helper.running = true
  }
}
