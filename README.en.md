# Omarchy Musicfox

English | [简体中文](README.md)

**Current development version: 0.1.3.** The project will remain in the `0.x`
series until its plugin API, account storage, and playback behavior are stable.

An independent NetEase Cloud Music client for Omarchy Quattro. It is inspired
by the capabilities of [go-musicfox](https://github.com/go-musicfox/go-musicfox),
but it does not install, launch, or call the `musicfox` executable.

![Musicfox sign-in panel](docs/screenshot-login.png)

## Features

- QR-code and Cookie sign-in with private local session persistence
- Daily recommendations, suggested playlists, search, personal playlists,
  charts, personal FM, playlist details, and a playback queue
- Album art, original and translated lyrics, seeking, volume, likes, previous,
  next, list repeat, track repeat, and shuffle
- Configurable quality from `standard` through `hires`
- Automatic locale detection with explicit English and Simplified Chinese modes
- Optional MPRIS integration for hardware media keys and desktop controllers
- Native Omarchy bar state and popup UI

## Requirements and architecture

```text
Omarchy QML UI ── line-delimited JSON ──> bundled Go helper ── HTTPS ──> NetEase
                                                |
                                                +── Unix IPC ──> mpv
```

Runtime requirements:

- Omarchy 4 / Quattro Shell
- `mpv` (included by default in Omarchy)
- x86-64 Linux for the bundled static helper; build from source with Go 1.22+
  on other architectures

The helper uses the MIT-licensed `go-musicfox/netease-music` protocol SDK.
NetEase does not publish this as a stable official API, so server changes may
break features. You remain responsible for complying with NetEase Cloud Music's
terms and applicable law.

## Install

```sh
omarchy plugin add https://github.com/taotao7/omarchy-musicfox.git --enable
```

For development:

```sh
git clone https://github.com/taotao7/omarchy-musicfox.git ~/.config/omarchy/plugins/taotao7.musicfox
cd ~/.config/omarchy/plugins/taotao7.musicfox
./scripts/build-helper.sh
omarchy plugin validate .
omarchy plugin enable taotao7.musicfox center
```

Do not install through a symlink; Omarchy intentionally rejects symlinks in
plugin directories.

## Sign in and data storage

Open the popup and scan the automatically generated code with the NetEase app.
Use **Refresh QR code** whenever it expires. Alternatively, paste a browser
Cookie containing `MUSIC_U` or `MUSIC_A`.

The session is stored at `~/.local/state/omarchy-musicfox/cookies.json` with
mode `0600`; QR images are cached under `~/.cache/omarchy-musicfox/`. Signing
out clears the local session.

## Configure

Use **Setup → Bar → Configure**, or:

```sh
omarchy bar set taotao7.musicfox displayMode "Title and artist"
omarchy bar set taotao7.musicfox hideWhenIdle false
omarchy bar set taotao7.musicfox maxLabelWidth 220
omarchy bar set taotao7.musicfox audioQuality higher
omarchy bar set taotao7.musicfox mprisEnabled false
omarchy bar set taotao7.musicfox language Auto
```

MPRIS is disabled by default because Omarchy's `omarchy.media` widget also
shows every MPRIS session, duplicating Musicfox in the bar. Enable
`mprisEnabled` when hardware media keys are preferred. The private mpv process
never registers an additional session.

## Build and verify

```sh
./scripts/build-helper.sh
./scripts/check.sh
```

The check runs the official manifest validator, localization and Go tests,
`go vet`, a static helper build, and `qmllint` against Omarchy's shell imports.

## Remove

```sh
omarchy plugin remove taotao7.musicfox
```

Removal leaves account state in place. To remove it too:

```sh
rm -rf ~/.local/state/omarchy-musicfox ~/.cache/omarchy-musicfox
```

## Security and license

Like every third-party Omarchy plugin, this code runs unsandboxed inside the
long-lived `omarchy-shell` process with your user permissions. Review it before
enabling it. The plugin never downloads executables or requests privilege.

This project is MIT licensed. See [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)
for dependency licenses and attribution.
