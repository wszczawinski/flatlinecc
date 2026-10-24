# flatlinecc — CLI/TUI for Wahoo ELEMNT ROAM v1

Bluetooth on the device is broken; everything goes over USB.

## Stack
- Go, module `github.com/wszczawinski/flatlinecc`
- Charm: Bubble Tea, Bubbles, Lip Gloss
- Device access: `adb` server (v1). Direct USB (gousb + own ADB protocol) later.

## UX
- Run `flatlinecc` → full-screen menu, everything happens inside the TUI.
- `flatlinecc -version` prints the version Go stamps from the git tag.

## v1 features
1. **Edit screens**: drill down: pages → page fields → searchable field list
   - change, add, remove and reorder fields
   - min / max fields per page type (workout 3–10, elevation 3–6, climb 3–5, map 2–3)
   - lap, KICKR, segment and planned workout fields are offered only on their own page
2. **Sounds**: mute and buzzer toggles (`MUTE`, `BUZZ_*`). Phone alerts skipped (need Bluetooth); LEDs and backlight are set on the device
3. **Backup / restore config**: manual copies of `BCfgManager-Bolt.xml`, kept in `backups/manual/` (restore via `elemnt_config.zip` doesn't work on current firmware)
4. **Pull rides**: copy new `.fit` rides from `/sdcard/exports` only, device untouched
5. **Sync routes**: `.gpx` files in `/sdcard/routes`, both ways. Local wins when sizes differ; nothing is deleted
6. **Device info**
7. **Settings**: data folder path, saved to a config file

## Session backup
On first connect, `BCfgManager-Bolt.xml` is saved to `backups/session/`. Removed on a clean quit, kept after a crash or a failed apply. Apply and restore are refused until it exists. It can be restored from Backup / restore, but not deleted there.

## Apply flow
Stage edits → review diff → Apply (pull a fresh config, refuse if its pages changed → put edits in → push → fix owner/perms → pull and verify → `force-stop` app, it restarts with the new config in seconds). Quit is blocked while applying. Details in `docs/format.md`.

## Local data
One configurable folder (default `./data`, relative to where flatlinecc runs). Each feature creates its subfolder on first write.
```
data/
  rides/
  routes/
  backups/
    session/
    manual/
```

## Milestones
1. ✓ **Skeleton**: menu, settings screen, adb connection, device info
2. ✓ Parser/writer for `DISPLAY_CFG` + live config (`BCfgManager-Bolt.xml`), round-trip byte-identical
3. ✓ Screen editor + apply flow: change, add, remove and reorder fields
4. ✓ Backup / restore UI
5. ✓ Sounds: toggles (`SOUNDS_CFG` bitmask not decoded)
6. ✓ Rides pull, routes sync

## Later / research
- Decode fields 426–429 (on the climb page): swap one in, see what the device shows
- Indoor Cycling profile pages (no `DISPLAY_CFG` of its own yet)
- Delete copied rides from the device to free space
- `SOUNDS_CFG` bitmask, `LED_MODE` values 2–4 (see `docs/format.md`)
- Direct USB, no adb dependency
- Polish translations (out of scope for now)

## Before publishing
- Add a LICENSE
- README: `go install github.com/wszczawinski/flatlinecc/cmd/flatlinecc@latest`
- CI: `go vet` + `go test` on push

## Out of scope
- Structured workouts (.plan)

## References
- ADB enable + config backup/restore: https://gist.github.com/Intyre/2c0a8e337671ed6f523950ef08e3ca3f
  - ADB auth (verified on ROAM v1, `product:roam1p1 model:ELEMNT_ROAM1_1`): unplug → power (settings) → power (back) → plug in. The adb key (`~/.android/adbkey`) must exist first, so start the adb server once before doing this.
  - Backup: `mkdir /sdcard/config_backup` → `elemnt_config.zip` appears
  - Restore: push zip to `/sdcard/config_restore`, reboot (not picked up on current firmware)
- cfg file reader (Python) + key names: https://gist.github.com/Intyre/05957b64b3fb1306361ca9e79ca01ac1
- DISPLAY_CFG format + page/field type tables: https://gist.github.com/Intyre/aefc6cced18f0610df57adcb70a11453#displaycfg
- elemntary (Electron, GPL-3.0): https://github.com/vti/elemntary
- WaoStats (ADB ride sync): https://github.com/masc4ii/WaoStats
