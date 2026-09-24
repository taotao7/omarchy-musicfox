import QtQuick
import Quickshell
import qs.Commons
import qs.Ui
import "I18n.js" as I18n

BarWidget {
  id: root
  moduleName: "taotao7.musicfox"

  readonly property var music: bar?.shell?.serviceFor(root.moduleName)
  readonly property color barText: bar ? bar.barForeground : Color.foreground
  readonly property color textColor: bar ? bar.foreground : Color.foreground
  readonly property color mutedColor: Util.alpha(textColor, 0.8)
  readonly property color accentColor: Color.accent
  readonly property string displayMode: String(setting("displayMode", "Title and artist"))
  readonly property bool hideWhenIdle: Boolean(setting("hideWhenIdle", false))
  readonly property string audioQuality: String(setting("audioQuality", "higher"))
  readonly property bool mprisEnabled: Boolean(setting("mprisEnabled", false))
  readonly property string languageSetting: String(setting("language", "Auto"))
  readonly property string uiLocale: I18n.resolve(languageSetting, Quickshell.env("LANG"))
  readonly property real maxLabelWidth: Math.max(80, Math.min(500,
    Number(setting("maxLabelWidth", 220)) || 220))
  readonly property bool opened: popupOpen
  readonly property bool popoutSwitchClosing: false
  readonly property string barLabel: {
    if (displayMode === "Icon only") return ""
    if (!music || !music.hasTrack) return "Musicfox"
    if (displayMode === "Title") return music.title
    return music.title + (music.artist ? "  ·  " + music.artist : "")
  }

  property bool popupOpen: false
  property bool showLyrics: false
  property string cookieText: ""

  onAudioQualityChanged: if (music && music.ready) music.setQuality(audioQuality)
  onMprisEnabledChanged: if (music && music.ready) music.setMpris(mprisEnabled)
  onUiLocaleChanged: if (music) music.uiLocale = uiLocale
  Component.onCompleted: if (music) music.setMpris(mprisEnabled)

  function t(key) { return I18n.text(uiLocale, key) }

  function open() {
    popupOpen = true
    if (music) music.uiLocale = uiLocale
    if (music) music.setMpris(mprisEnabled)
    if (music && music.ready) music.setQuality(audioQuality)
    if (music && music.ready && !music.loggedIn &&
        (music.qrImage === "" || music.qrCode === 800)) music.startQrLogin()
  }
  function close() { popupOpen = false }
  function toggle() { if (popupOpen) close(); else open() }
  function closeForPopoutSwitch() { close() }

  visible: !hideWhenIdle || (music && music.hasTrack)
  implicitWidth: visible ? barRow.implicitWidth + Style.space(14) : 0
  implicitHeight: barSize

  component MiniButton: Rectangle {
    id: button
    property string label: ""
    property bool active: false
    signal clicked()
    implicitWidth: Math.max(Style.space(34), buttonText.implicitWidth + Style.space(16))
    implicitHeight: Style.space(32)
    radius: Style.space(8)
    color: active ? Util.alpha(root.accentColor, 0.3) :
      (buttonMouse.containsMouse ? Util.alpha(root.textColor, 0.2) : Util.alpha(root.textColor, 0.12))
    border.width: 1
    border.color: active ? Util.alpha(root.accentColor, 0.7) : Util.alpha(root.textColor, 0.18)
    opacity: enabled ? 1 : 0.6

    Text {
      id: buttonText
      anchors.centerIn: parent
      text: button.label
      color: button.active ? root.accentColor : root.textColor
      font.family: root.bar ? root.bar.fontFamily : Style.font.family
      font.pixelSize: Style.font.bodySmall
      font.bold: button.active
    }
    MouseArea {
      id: buttonMouse
      anchors.fill: parent
      hoverEnabled: true
      cursorShape: button.enabled ? Qt.PointingHandCursor : Qt.ArrowCursor
      onClicked: if (button.enabled) button.clicked()
    }
  }

  Row {
    id: barRow
    anchors.centerIn: parent
    spacing: Style.space(6)

    Text {
      anchors.verticalCenter: parent.verticalCenter
      text: music && music.playback.playing ? "󰏤" : "󰐊"
      color: music && music.playback.playing ? root.accentColor : root.barText
      font.family: root.bar ? root.bar.fontFamily : Style.font.family
      font.pixelSize: Style.font.body
    }
    Item {
      width: Math.min(root.maxLabelWidth, barTitle.implicitWidth)
      height: barTitle.implicitHeight
      visible: !root.vertical && root.barLabel !== ""
      clip: true
      Text {
        id: barTitle
        text: root.barLabel
        color: root.barText
        font.family: root.bar ? root.bar.fontFamily : Style.font.family
        font.pixelSize: Style.font.body
      }
    }
  }

  MouseArea {
    anchors.fill: parent
    acceptedButtons: Qt.LeftButton | Qt.RightButton | Qt.MiddleButton
    hoverEnabled: true
    cursorShape: Qt.PointingHandCursor
    onClicked: function(mouse) {
      if (mouse.button === Qt.MiddleButton && root.music) root.music.next()
      else if (mouse.button === Qt.LeftButton && root.music && root.music.hasTrack) root.music.toggle()
      else root.toggle()
    }
    onWheel: function(wheel) {
      if (!root.music || !root.music.hasTrack) return
      if (wheel.angleDelta.y > 0) root.music.previous()
      else root.music.next()
    }
    onEntered: if (root.bar) root.bar.showTooltip(root,
      root.music && root.music.hasTrack ? root.music.title + " — " + root.music.artist : root.t("openMusicfox"))
    onExited: if (root.bar) root.bar.hideTooltip(root)
  }

  KeyboardPanel {
    id: popup
    anchorItem: root
    bar: root.bar
    owner: root
    open: root.popupOpen
    focusTarget: root.music && root.music.loggedIn ? searchInput : cookieInput
    contentWidth: popup.fittedContentWidth(Style.space(470))
    contentHeight: popup.fittedContentHeight(Style.space(
      root.music && root.music.loggedIn ? 650 :
      (root.music && root.music.qrImage ? 650 : 370)))

    Column {
      id: panel
      anchors.fill: parent
      spacing: Style.space(10)

      Row {
        width: parent.width
        height: Style.space(82)
        spacing: Style.space(12)

        Rectangle {
          id: coverArt
          width: Style.space(82)
          height: width
          radius: Style.space(10)
          color: Util.alpha(root.textColor, 0.08)
          clip: true
          Image {
            anchors.fill: parent
            source: root.music && root.music.hasTrack ? String(root.music.currentTrack.coverUrl || "") : ""
            fillMode: Image.PreserveAspectCrop
            asynchronous: true
            visible: status === Image.Ready
          }
          Text {
            anchors.centerIn: parent
            text: "󰝚"
            color: root.mutedColor
            font.family: root.bar ? root.bar.fontFamily : Style.font.family
            font.pixelSize: Style.font.displayLarge
          }
        }

        Column {
          width: parent.width - coverArt.width - Style.space(12) -
            (headerLogout.visible ? headerLogout.width + Style.space(12) : 0)
          anchors.verticalCenter: parent.verticalCenter
          spacing: Style.space(4)
          Text {
            width: parent.width
            text: root.music ? root.music.title : "Musicfox"
            color: root.textColor
            elide: Text.ElideRight
            font.family: root.bar ? root.bar.fontFamily : Style.font.family
            font.pixelSize: Style.font.subtitle
            font.bold: true
          }
          Text {
            width: parent.width
            text: root.music ? root.music.artist : root.t("music")
            color: root.mutedColor
            elide: Text.ElideRight
            font.family: root.bar ? root.bar.fontFamily : Style.font.family
            font.pixelSize: Style.font.bodySmall
          }
          Text {
            width: parent.width
            text: root.music && root.music.hasTrack ? String(root.music.currentTrack.album || "") :
              (root.music && root.music.ready ? root.t("serviceReady") : root.t("serviceStarting"))
            color: root.mutedColor
            elide: Text.ElideRight
            font.family: root.bar ? root.bar.fontFamily : Style.font.family
            font.pixelSize: Style.font.caption
          }
        }

        MiniButton {
          id: headerLogout
          anchors.verticalCenter: parent.verticalCenter
          visible: root.music && root.music.loggedIn
          label: root.t("logout")
          onClicked: root.music.logout()
        }
      }

      Column {
        width: parent.width
        spacing: Style.space(5)
        visible: root.music && root.music.hasTrack
        Rectangle {
          width: parent.width
          height: Style.space(6)
          radius: height / 2
          color: Util.alpha(root.textColor, 0.16)
          Rectangle {
            height: parent.height
            radius: height / 2
            color: root.accentColor
            width: parent.width * (root.music ? Math.max(0, Math.min(1,
              Number(root.music.playback.position || 0) /
              Math.max(1, Number(root.music.playback.duration || root.music.currentTrack.duration / 1000 || 1)))) : 0)
          }
          MouseArea {
            anchors.fill: parent
            cursorShape: Qt.PointingHandCursor
            onClicked: function(mouse) {
              var duration = Number(root.music.playback.duration || root.music.currentTrack.duration / 1000 || 0)
              root.music.seek(duration * mouse.x / width)
            }
          }
        }
        Row {
          width: parent.width
          Text {
            text: root.music ? root.music.formatDuration(root.music.playback.position) : "0:00"
            color: root.mutedColor
            font.family: root.bar ? root.bar.fontFamily : Style.font.family
            font.pixelSize: Style.font.caption
          }
          Item { width: parent.width - Style.space(70); height: 1 }
          Text {
            text: root.music ? root.music.formatDuration(root.music.playback.duration || root.music.currentTrack.duration / 1000) : "0:00"
            color: root.mutedColor
            font.family: root.bar ? root.bar.fontFamily : Style.font.family
            font.pixelSize: Style.font.caption
          }
        }
      }

      Item {
        width: parent.width
        height: Style.space(42)

        MiniButton {
          id: playbackModeButton
          anchors.left: parent.left
          anchors.verticalCenter: parent.verticalCenter
          label: root.music && root.music.playback.mode === "repeat" ? "󰑘  " + root.t("modeRepeat") :
            (root.music && root.music.playback.mode === "shuffle" ? "󰒝  " + root.t("modeShuffle") : "󰑖  " + root.t("modeList"))
          enabled: root.music && root.music.hasTrack
          onClicked: root.music.cycleMode()
        }

        Row {
          anchors.centerIn: parent
          spacing: Style.space(7)
          MiniButton {
            implicitWidth: Style.space(36)
            implicitHeight: Style.space(36)
            label: "󰒮"
            enabled: root.music && root.music.hasTrack
            onClicked: root.music.previous()
          }
          MiniButton {
            implicitWidth: Style.space(42)
            implicitHeight: Style.space(42)
            label: root.music && root.music.playback.playing ? "󰏤" : "󰐊"
            active: root.music && root.music.playback.playing
            enabled: root.music && root.music.hasTrack
            onClicked: root.music.toggle()
          }
          MiniButton {
            implicitWidth: Style.space(36)
            implicitHeight: Style.space(36)
            label: "󰒭"
            enabled: root.music && root.music.hasTrack
            onClicked: root.music.next()
          }
        }

        Row {
          anchors.right: parent.right
          anchors.verticalCenter: parent.verticalCenter
          spacing: Style.space(7)
          MiniButton {
            label: root.music && root.music.playback.liked ? "󰋑" : "󰋕"
            active: root.music && root.music.playback.liked
            enabled: root.music && root.music.loggedIn && root.music.hasTrack
            onClicked: root.music.toggleLike()
          }
          Rectangle {
            anchors.verticalCenter: parent.verticalCenter
            width: 1
            height: Style.space(22)
            color: Util.alpha(root.textColor, 0.16)
          }
          Text {
            anchors.verticalCenter: parent.verticalCenter
            text: "󰕾"
            color: root.mutedColor
            font.family: root.bar ? root.bar.fontFamily : Style.font.family
            font.pixelSize: Style.font.body
          }
          Rectangle {
            anchors.verticalCenter: parent.verticalCenter
            width: Style.space(52)
            height: Style.space(5)
            radius: height / 2
            color: Util.alpha(root.textColor, 0.16)
            Rectangle {
              width: parent.width * Math.max(0, Math.min(1,
                Number(root.music ? root.music.playback.volume : 70) / 100))
              height: parent.height
              radius: height / 2
              color: root.accentColor
            }
            MouseArea {
              anchors.fill: parent
              cursorShape: Qt.PointingHandCursor
              onClicked: function(mouse) { if (root.music) root.music.setVolume(100 * mouse.x / width) }
            }
          }
        }
      }

      Rectangle { width: parent.width; height: 1; color: Util.alpha(root.textColor, 0.12) }

      Column {
        width: parent.width
        spacing: Style.space(12)
        visible: !root.music || !root.music.loggedIn

        Text {
          width: parent.width
          text: root.t("login")
          horizontalAlignment: Text.AlignHCenter
          color: root.textColor
          font.family: root.bar ? root.bar.fontFamily : Style.font.family
          font.pixelSize: Style.font.subtitle
          font.bold: true
        }
        Text {
          width: parent.width
          text: root.t("loginHelp")
          wrapMode: Text.WordWrap
          horizontalAlignment: Text.AlignHCenter
          color: root.mutedColor
          font.family: root.bar ? root.bar.fontFamily : Style.font.family
          font.pixelSize: Style.font.bodySmall
        }
        Item {
          width: parent.width
          height: root.music && root.music.qrImage ? Style.space(218) : Style.space(40)
          Image {
            anchors.centerIn: parent
            width: Style.space(210)
            height: width
            source: root.music && root.music.qrImage ? "file://" + root.music.qrImage : ""
            fillMode: Image.PreserveAspectFit
            visible: source !== ""
          }
          MiniButton {
            anchors.centerIn: parent
            visible: !root.music || !root.music.qrImage
            label: root.t("qrCreate")
            enabled: root.music && root.music.ready
            onClicked: root.music.startQrLogin()
          }
        }
        Text {
          width: parent.width
          visible: root.music && root.music.qrMessage !== ""
          text: root.music ? root.music.qrMessage : ""
          horizontalAlignment: Text.AlignHCenter
          color: root.mutedColor
          font.family: root.bar ? root.bar.fontFamily : Style.font.family
          font.pixelSize: Style.font.bodySmall
        }
        MiniButton {
          anchors.horizontalCenter: parent.horizontalCenter
          visible: root.music && root.music.qrImage !== ""
          label: root.t("qrRefresh")
          enabled: root.music && root.music.ready && !root.music.busy
          onClicked: root.music.startQrLogin()
        }
        Rectangle {
          width: parent.width
          height: Style.space(38)
          radius: Style.space(8)
          color: Util.alpha(root.textColor, 0.12)
          border.width: 1
          border.color: cookieInput.activeFocus ? root.accentColor : Util.alpha(root.textColor, 0.14)
          TextInput {
            id: cookieInput
            anchors.fill: parent
            anchors.margins: Style.space(10)
            verticalAlignment: TextInput.AlignVCenter
            color: root.textColor
            selectionColor: root.accentColor
            clip: true
            echoMode: TextInput.Password
            text: root.cookieText
            onTextChanged: root.cookieText = text
            font.family: root.bar ? root.bar.fontFamily : Style.font.family
            font.pixelSize: Style.font.bodySmall
          }
          Text {
            anchors.verticalCenter: parent.verticalCenter
            anchors.left: parent.left
            anchors.leftMargin: Style.space(10)
            visible: cookieInput.text === ""
            text: "MUSIC_U=…; MUSIC_A=…"
            color: root.mutedColor
            font.family: root.bar ? root.bar.fontFamily : Style.font.family
            font.pixelSize: Style.font.bodySmall
          }
        }
        MiniButton {
          anchors.horizontalCenter: parent.horizontalCenter
          label: root.t("cookieImport")
          enabled: root.music && root.cookieText.trim() !== ""
          onClicked: root.music.loginWithCookie(root.cookieText)
        }
      }

      Column {
        width: parent.width
        spacing: Style.space(9)
        visible: root.music && root.music.loggedIn

        Row {
          width: parent.width
          spacing: Style.space(6)
          Rectangle {
            width: parent.width - searchButton.width - Style.space(6)
            height: Style.space(36)
            radius: Style.space(8)
            color: Util.alpha(root.textColor, 0.12)
            border.width: 1
            border.color: searchInput.activeFocus ? root.accentColor : Util.alpha(root.textColor, 0.14)
            TextInput {
              id: searchInput
              anchors.fill: parent
              anchors.margins: Style.space(9)
              verticalAlignment: TextInput.AlignVCenter
              color: root.textColor
              selectionColor: root.accentColor
              clip: true
              font.family: root.bar ? root.bar.fontFamily : Style.font.family
              font.pixelSize: Style.font.bodySmall
              Keys.onReturnPressed: if (root.music) root.music.search(text)
            }
            Text {
              anchors.verticalCenter: parent.verticalCenter
              anchors.left: parent.left
              anchors.leftMargin: Style.space(9)
              visible: searchInput.text === ""
              text: root.t("searchHint")
              color: root.mutedColor
              font.family: root.bar ? root.bar.fontFamily : Style.font.family
              font.pixelSize: Style.font.bodySmall
            }
          }
          MiniButton {
            id: searchButton
            label: root.t("search")
            enabled: searchInput.text.trim() !== ""
            onClicked: root.music.search(searchInput.text)
          }
        }

        Row {
          width: parent.width
          spacing: Style.space(5)
          MiniButton { label: root.t("recommendations"); active: !root.showLyrics && root.music.activeView === "home"; onClicked: { root.showLyrics = false; root.music.loadHome() } }
          MiniButton { label: root.t("playlist"); active: !root.showLyrics && root.music.activeView === "library"; onClicked: { root.showLyrics = false; root.music.loadLibrary() } }
          MiniButton { label: root.t("toplists"); active: !root.showLyrics && root.music.activeView === "toplists"; onClicked: { root.showLyrics = false; root.music.loadToplists() } }
          MiniButton { label: root.t("personalFm"); onClicked: root.music.playPersonalFm() }
          MiniButton { label: root.t("queue"); active: !root.showLyrics && root.music.activeView === "queue"; onClicked: { root.showLyrics = false; root.music.showQueue() } }
          MiniButton { label: root.t("lyrics"); active: root.showLyrics; enabled: root.music.hasTrack; onClicked: root.showLyrics = true }
        }

        Text {
          width: parent.width
          text: root.showLyrics ? root.t("lyrics") : root.music.viewTitle
          color: root.textColor
          elide: Text.ElideRight
          font.family: root.bar ? root.bar.fontFamily : Style.font.family
          font.pixelSize: Style.font.body
          font.bold: true
        }

        Flickable {
          id: contentFlick
          width: parent.width
          height: Style.space(310)
          contentWidth: width
          contentHeight: contentColumn.implicitHeight
          clip: true
          boundsBehavior: Flickable.StopAtBounds

          Column {
            id: contentColumn
            width: parent.width
            spacing: Style.space(5)

            Text {
              width: parent.width
              visible: root.showLyrics && root.music && root.music.lyricLines.length === 0
              text: root.t("noLyrics")
              horizontalAlignment: Text.AlignHCenter
              color: root.mutedColor
              font.family: root.bar ? root.bar.fontFamily : Style.font.family
              font.pixelSize: Style.font.bodySmall
            }

            Repeater {
              id: lyricRepeater
              model: root.showLyrics && root.music ? root.music.lyricLines : []
              delegate: Column {
                required property var modelData
                required property int index
                width: contentColumn.width
                spacing: Style.space(2)
                readonly property bool active: root.music && index === root.music.currentLyricIndex()
                Text {
                  width: parent.width
                  text: String(modelData.text || "")
                  wrapMode: Text.Wrap
                  horizontalAlignment: Text.AlignHCenter
                  color: parent.active ? root.accentColor : root.textColor
                  font.bold: parent.active
                  font.family: root.bar ? root.bar.fontFamily : Style.font.family
                  font.pixelSize: Style.font.bodySmall
                }
                Text {
                  width: parent.width
                  visible: text !== ""
                  text: String(modelData.translated || "")
                  wrapMode: Text.Wrap
                  horizontalAlignment: Text.AlignHCenter
                  color: parent.active ? root.accentColor : root.mutedColor
                  font.family: root.bar ? root.bar.fontFamily : Style.font.family
                  font.pixelSize: Style.font.caption
                }
              }
            }

            Repeater {
              model: !root.showLyrics && root.music ? root.music.playlists : []
              delegate: Rectangle {
                required property var modelData
                required property int index
                width: contentColumn.width
                height: Style.space(54)
                radius: Style.space(8)
                color: playlistMouse.containsMouse ? Util.alpha(root.textColor, 0.1) : "transparent"
                Image {
                  id: playlistCover
                  anchors.left: parent.left
                  anchors.verticalCenter: parent.verticalCenter
                  width: Style.space(44)
                  height: width
                  source: String(modelData.coverUrl || "")
                  fillMode: Image.PreserveAspectCrop
                  asynchronous: true
                }
                Column {
                  anchors.left: playlistCover.right
                  anchors.leftMargin: Style.space(10)
                  anchors.right: parent.right
                  anchors.verticalCenter: parent.verticalCenter
                  Text {
                    width: parent.width
                    text: String(modelData.name || root.t("unnamedPlaylist"))
                    color: root.textColor
                    elide: Text.ElideRight
                    font.family: root.bar ? root.bar.fontFamily : Style.font.family
                    font.pixelSize: Style.font.bodySmall
                  }
                  Text {
                    width: parent.width
                    text: (modelData.trackCount ? modelData.trackCount + root.t("songs") : "") +
                      (modelData.creator ? "  ·  " + modelData.creator : "")
                    color: root.mutedColor
                    elide: Text.ElideRight
                    font.family: root.bar ? root.bar.fontFamily : Style.font.family
                    font.pixelSize: Style.font.caption
                  }
                }
                MouseArea {
                  id: playlistMouse
                  anchors.fill: parent
                  hoverEnabled: true
                  cursorShape: Qt.PointingHandCursor
                  onClicked: root.music.loadPlaylist(modelData)
                }
              }
            }

            Repeater {
              model: !root.showLyrics && root.music ? root.music.tracks : []
              delegate: Rectangle {
                required property var modelData
                required property int index
                width: contentColumn.width
                height: Style.space(48)
                radius: Style.space(8)
                color: root.music && String(root.music.currentTrack.id || "") === String(modelData.id || "")
                  ? Util.alpha(root.accentColor, 0.16)
                  : (trackMouse.containsMouse ? Util.alpha(root.textColor, 0.1) : "transparent")
                Text {
                  id: trackNumber
                  anchors.left: parent.left
                  anchors.verticalCenter: parent.verticalCenter
                  width: Style.space(26)
                  text: index + 1
                  horizontalAlignment: Text.AlignHCenter
                  color: root.mutedColor
                  font.family: root.bar ? root.bar.fontFamily : Style.font.family
                  font.pixelSize: Style.font.caption
                }
                Column {
                  anchors.left: trackNumber.right
                  anchors.right: trackDuration.left
                  anchors.rightMargin: Style.space(8)
                  anchors.verticalCenter: parent.verticalCenter
                  Text {
                    width: parent.width
                    text: String(modelData.name || root.t("unknownSong"))
                    color: root.textColor
                    elide: Text.ElideRight
                    font.family: root.bar ? root.bar.fontFamily : Style.font.family
                    font.pixelSize: Style.font.bodySmall
                  }
                  Text {
                    width: parent.width
                    text: String(modelData.artists || root.t("unknownArtist")) +
                      (modelData.album ? "  ·  " + modelData.album : "")
                    color: root.mutedColor
                    elide: Text.ElideRight
                    font.family: root.bar ? root.bar.fontFamily : Style.font.family
                    font.pixelSize: Style.font.caption
                  }
                }
                Text {
                  id: trackDuration
                  anchors.right: parent.right
                  anchors.verticalCenter: parent.verticalCenter
                  width: Style.space(42)
                  text: root.music ? root.music.formatDuration(Number(modelData.duration || 0) / 1000) : ""
                  color: root.mutedColor
                  horizontalAlignment: Text.AlignRight
                  font.family: root.bar ? root.bar.fontFamily : Style.font.family
                  font.pixelSize: Style.font.caption
                }
                MouseArea {
                  id: trackMouse
                  anchors.fill: parent
                  hoverEnabled: true
                  cursorShape: Qt.PointingHandCursor
                  onClicked: {
                    if (root.music.activeView === "queue") root.music.playQueueIndex(index)
                    else root.music.playTracks(root.music.tracks, index)
                  }
                }
              }
            }

            Text {
              width: parent.width
              height: Style.space(90)
              visible: !root.showLyrics && root.music && !root.music.busy &&
                root.music.tracks.length === 0 && root.music.playlists.length === 0
              text: root.t("empty")
              horizontalAlignment: Text.AlignHCenter
              verticalAlignment: Text.AlignVCenter
              color: root.mutedColor
              font.family: root.bar ? root.bar.fontFamily : Style.font.family
              font.pixelSize: Style.font.bodySmall
            }
          }
        }
      }

      Timer {
        interval: 500
        repeat: true
        running: root.popupOpen && root.showLyrics && root.music && root.music.playback.playing
        onTriggered: {
          var item = lyricRepeater.itemAt(root.music.currentLyricIndex())
          if (item) contentFlick.contentY = Math.max(0,
            Math.min(contentFlick.contentHeight - contentFlick.height,
              item.y - contentFlick.height / 2 + item.height / 2))
        }
      }

      Text {
        width: parent.width
        visible: root.music && (root.music.busy || root.music.lastError !== "")
        text: root.music && root.music.lastError !== "" ? "⚠ " + root.music.lastError : root.t("loading")
        color: root.music && root.music.lastError !== "" ? "#ef4444" : root.mutedColor
        wrapMode: Text.WordWrap
        maximumLineCount: 2
        elide: Text.ElideRight
        font.family: root.bar ? root.bar.fontFamily : Style.font.family
        font.pixelSize: Style.font.caption
      }
    }
  }
}
