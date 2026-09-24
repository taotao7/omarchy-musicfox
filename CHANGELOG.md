# Changelog

All notable changes to this project are documented here. Versions follow
[Semantic Versioning](https://semver.org/).

## [0.1.2] - 2026-09-24

### Changed

- Reorganize the player toolbar into distinct mode, centered transport, and
  favorite/volume groups for clearer visual hierarchy.
- Move sign-out into the track header so playlist titles have a dedicated row.
- Update the public preview to show the refined signed-in layout.

## [0.1.1] - 2026-09-24

### Fixed

- Keep playback state correct regardless of the order in which mpv reports
  pause and idle changes, so the play/pause button responds reliably.
- Prevent the private mpv process from loading the system `mpv-mpris` script,
  which exposed the same track as a second media player.
- Make MPRIS system controls opt-in to avoid duplicating Musicfox in the
  Omarchy bar when `omarchy.media` is also enabled.

### Changed

- Replace the public screenshot with the complete signed-in player panel.

## [0.1.0] - 2026-09-24

### Added

- Independent NetEase Cloud Music client; no `musicfox` executable is needed.
- QR-code and Cookie login with a private, persistent local session.
- Daily recommendations, recommended playlists, search, personal playlists,
  charts, personal FM, playlist browsing, and queue playback.
- Album art, synced lyric source display, seek, volume, quality, like,
  previous/next, list repeat, single repeat, and shuffle controls.
- MPRIS integration for hardware media keys and desktop media controls.
- Automatic locale detection plus Simplified Chinese and English interfaces.
- Bundled static x86-64 helper and source build workflow.
