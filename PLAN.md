# flatlinecc — CLI/TUI for Wahoo ELEMNT ROAM v1

Bluetooth on the device is broken; everything goes over USB.

## Stack
- Go, module `github.com/wszczawinski/flatlinecc`
- Charm: Bubble Tea, Bubbles, Lip Gloss, Huh
- Device access: `adb` server (v1). Direct USB (gousb + own ADB protocol) later.

## UX
- Run `flatlinecc` → full-screen menu, everything happens inside the TUI.

## v1 features
1. **Edit screens**: drill down: pages → page fields → searchable field list
   - change the field in a slot
   - add / remove fields on a page
2. **Sounds**: MUTE / BUZZ_* / ALERT_* (format undocumented, decode from backup diffs)
3. **Backup / restore config**: `elemnt_config.zip`
4. **Pull rides**: copy new rides only, device untouched
5. **Push routes**: push routes not yet on the device
6. **Device info**
7. **Settings**: data folder path, saved to a config file

## Apply flow
Stage edits → review diff → Apply (auto backup → push to `/sdcard/config_restore` → reboot).

## Local data
One configurable folder (default `./data` in the repo). Each feature creates its subfolder on first write.
```
data/
  rides/
  routes/
  backups/
```

## Milestones
1. **Skeleton**: menu, settings screen, adb connection
2. Config backup + parser/writer for `bolt-65535.cfg` (`DISPLAY_CFG`), round-trip byte-identical
3. Screen editor + apply flow
4. Backup / restore UI
5. Rides pull, routes push
6. Sounds
7. Device info

## Later / research
- Instant apply without reboot (send app config messages via adb)
- Direct USB, no adb dependency
- Polish translations (out of scope for now)

## Out of scope
- Structured workouts (.plan)

## References
- ADB enable + config backup/restore: https://gist.github.com/Intyre/2c0a8e337671ed6f523950ef08e3ca3f
  - ADB auth (verified on ROAM v1, `product:roam1p1 model:ELEMNT_ROAM1_1`): unplug → power (settings) → power (back) → plug in. The adb key (`~/.android/adbkey`) must exist first, so start the adb server once before doing this.
  - Backup: `mkdir /sdcard/config_backup` → `elemnt_config.zip` appears
  - Restore: push zip to `/sdcard/config_restore`, reboot
- cfg file reader (Python) + key names: https://gist.github.com/Intyre/05957b64b3fb1306361ca9e79ca01ac1
- DISPLAY_CFG format + page/field type tables: https://gist.github.com/Intyre/aefc6cced18f0610df57adcb70a11453#displaycfg
- elemntary (Electron, GPL-3.0): https://github.com/vti/elemntary
- WaoStats (ADB ride sync): https://github.com/masc4ii/WaoStats
