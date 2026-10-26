# flatlinecc plan

What's done is in the [README](README.md). This file keeps what's next.

## Later / research
- Open format questions (fields 426–429, `SOUNDS_CFG`, …): see [docs/format.md](docs/format.md#open-questions)
- Indoor Cycling profile pages (no `DISPLAY_CFG` of its own yet)
- Delete copied rides from the device to free space
- Direct USB (gousb + own ADB protocol), no adb dependency
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
