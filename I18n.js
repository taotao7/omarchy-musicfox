var messages = {
  zh: {
    music: "网易云音乐",
    openMusicfox: "打开 Musicfox",
    serviceStarting: "播放器服务启动中…",
    serviceReady: "网易云音乐 · 已就绪",
    serviceNotReady: "播放器服务尚未就绪",
    serviceExited: "播放器服务已退出",
    parseError: "无法解析播放器服务响应",
    unknownError: "未知错误",
    qrCreating: "正在生成二维码…",
    qrCreate: "生成登录二维码",
    qrRefresh: "刷新二维码",
    qrSuccess: "登录成功",
    login: "登录网易云音乐",
    loginHelp: "扫码登录，或从浏览器复制包含 MUSIC_U / MUSIC_A 的 Cookie。凭据仅保存在本机。",
    cookieImport: "导入 Cookie 并登录",
    home: "每日推荐",
    search: "搜索",
    searchHint: "搜索歌曲、歌手…",
    searchTitle: "搜索：",
    library: "我的歌单",
    playlist: "歌单",
    unnamedPlaylist: "未命名歌单",
    toplists: "排行榜",
    personalFm: "私人 FM",
    queue: "播放队列",
    lyrics: "歌词",
    noLyrics: "暂无歌词",
    logout: "退出登录",
    recommendations: "推荐",
    songs: " 首",
    unknownSong: "未知歌曲",
    unknownArtist: "未知歌手",
    empty: "这里还没有内容",
    loading: "正在载入…",
    modeRepeat: "单曲",
    modeShuffle: "随机",
    modeList: "列表"
  },
  en: {
    music: "NetEase Cloud Music",
    openMusicfox: "Open Musicfox",
    serviceStarting: "Starting player service…",
    serviceReady: "NetEase Cloud Music · Ready",
    serviceNotReady: "Player service is not ready",
    serviceExited: "Player service exited",
    parseError: "Could not parse the player service response",
    unknownError: "Unknown error",
    qrCreating: "Creating QR code…",
    qrCreate: "Create login QR code",
    qrRefresh: "Refresh QR code",
    qrSuccess: "Signed in",
    login: "Sign in to NetEase Cloud Music",
    loginHelp: "Scan with the NetEase app, or paste a browser Cookie containing MUSIC_U / MUSIC_A. Credentials stay on this device.",
    cookieImport: "Import Cookie and sign in",
    home: "Daily recommendations",
    search: "Search",
    searchHint: "Search songs and artists…",
    searchTitle: "Search: ",
    library: "My playlists",
    playlist: "Playlist",
    unnamedPlaylist: "Untitled playlist",
    toplists: "Charts",
    personalFm: "Personal FM",
    queue: "Play queue",
    lyrics: "Lyrics",
    noLyrics: "No lyrics available",
    logout: "Sign out",
    recommendations: "For you",
    songs: " tracks",
    unknownSong: "Unknown song",
    unknownArtist: "Unknown artist",
    empty: "Nothing here yet",
    loading: "Loading…",
    modeRepeat: "One",
    modeShuffle: "Shuffle",
    modeList: "List"
  }
}

function resolve(setting, environment) {
  if (setting === "简体中文") return "zh"
  if (setting === "English") return "en"
  return String(environment || "").toLowerCase().indexOf("zh") === 0 ? "zh" : "en"
}

function text(locale, key) {
  var selected = messages[locale] || messages.en
  return selected[key] || messages.en[key] || key
}

if (typeof module !== "undefined") {
  module.exports = { resolve: resolve, text: text }
}
