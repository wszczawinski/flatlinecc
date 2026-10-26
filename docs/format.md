# ELEMNT ROAM notes

Notes from decoding a ROAM v1. Based on [Intyre's notes](https://gist.github.com/Intyre/aefc6cced18f0610df57adcb70a11453), with differences marked.

## Live config

`adb shell` runs as root (`uid=0`, SELinux permissive), so the app's private data is readable and writable.

The Wahoo app (`com.wahoofitness.bolt`, its own app user) keeps its config in Android SharedPreferences:

- `/data/data/com.wahoofitness.bolt/shared_prefs/BCfgManager-Bolt.xml`: the `bolt-*.cfg` fields, the one flatlinecc edits
- `/data/data/com.wahoofitness.bolt/shared_prefs/BCfgManager-Comp.xml`: the `comp.cfg` fields

Each entry's key is JSON, e.g. `{"type":53,"boltId":""}` or `{"type":2,"boltId":"","cfgProfileId":-13}`:

- `type` is the cfg field ID
- `cfgProfileId` is the profile as a signed int16: `-1` = 65535, `-13` = 65523; keys without it look like the active values
- byte-array fields (like `DISPLAY_CFG`) are stored as a string of signed bytes: `[1,-59,0,7,…]`
- `DISPLAY_CFG` only exists as `{"type":53,"boltId":""}`

Field names by ID are listed in Intyre's [cfg reader](https://gist.github.com/Intyre/05957b64b3fb1306361ca9e79ca01ac1).

### Applying a change

Verified: swapped two fields on the workout page, no reboot needed. flatlinecc does:

```
adb pull /data/data/com.wahoofitness.bolt/shared_prefs/BCfgManager-Bolt.xml   # fresh copy, then edit it
adb shell stat -c %u:%g /data/data/com.wahoofitness.bolt/shared_prefs   # app owner; push creates the file as root
adb push BCfgManager-Bolt.xml /data/data/com.wahoofitness.bolt/shared_prefs/
adb shell chown <app-uid>:<app-gid> /data/data/com.wahoofitness.bolt/shared_prefs/BCfgManager-Bolt.xml
adb shell chmod 660 /data/data/com.wahoofitness.bolt/shared_prefs/BCfgManager-Bolt.xml
adb pull /data/data/com.wahoofitness.bolt/shared_prefs/BCfgManager-Bolt.xml   # compare with what was pushed
adb shell am force-stop com.wahoofitness.bolt
```

- The launcher restarts the app immediately after `force-stop`, and it reads the file on start. So push first, then stop. Stopping before the push just reloads the old config.
- Small race: if the app saves prefs between push and stop, it overwrites the file. That's what the second pull catches.
- `adb reboot` also works instead of `force-stop`.

## `DISPLAY_CFG` (field 53)

The data page layout.

### Header (6 bytes)

```
01 c5 00 07 00 07
```

**Unknown.** Intyre: `u8 0, u16 currentPage, u8 pageCount`, which doesn't fit (sample has 8 pages). Needs diffing.

### Page record

All integers little-endian.

```
u8   ?           always 01 (Intyre: 00)
u16  pageID      e.g. 0x001b
u16  ?           0,1,2,0,3,4,5,6 in sample, looks like an order index (new vs Intyre)
u8   pageType    confirmed by matching fields to page types
u8   ?           1, 4 or 5, maybe zoom
u16  ?           always 01 00, maybe enabled + padding
u16  flags?      ff ff, fe ff on elevation page (type 2)
u8   fieldCount
[fieldCount]u16 fieldType
```

Intyre's layout is `u8 0, u16 pageID, u8 type, u8 zoom, bool enabled, u16 flags, u8 fieldCount`. Ours has 2 extra bytes; the type byte and field list match.

### Sample (8 pages)

| pageID | idx | type | ? | flags | fields |
|---|---|---|---|---|---|
| 0x1b | 0 | 1 lap | 5 | ffff | 155 11 154 34 1 3 35 4 36 |
| 0x1d | 1 | 0 workout | 5 | ffff | 158 10 30 32 2 5 40 425 45 41 |
| 0x2b | 2 | 2 elevation | 4 | fffe | 158 425 40 41 47 45 |
| 0x13 | 0 | 39 climb (?) | 4 | ffff | 426 427 428 425 429 |
| 0x0d | 3 | 3 map | 1 | ffff | 158 21 |
| 0x11 | 4 | 4 KICKR | 1 | ffff | 180 32 60 158 |
| 0x15 | 5 | 15 planned workout | 4 | ffff | 294 180 297 299 298 158 |
| 0x17 | 6 | 8 segment | 4 | ffff | 238 239 240 241 242 425 201 |

Page type 39 and field types 425–429 are newer than Intyre's tables (which end at 29 and 391). Page type 39 is probably the auto-detected climb (fields 426–429 + grade), unconfirmed. flatlinecc shows type 2 as "Elevation" and type 39 as "Climb".

Confirmed on device (workout page 0x1d): speed, distance, clock, ride time, avg speed, max speed, ascent, grade, elevation, descent. So 158 shows as "speed" and 425 is grade.

## Settings

Plain values in the live config, one entry per profile plus the active one (no `cfgProfileId`). Changing a setting on the device updates only the active entry, so flatlinecc does the same. Names from Intyre's cfg reader.

### Sounds (`<boolean>`)

| ID | Key | Notes |
|---|---|---|
| 8 | `BUZZ_WORKOUT` | workout beeps, probably start / pause / lap |
| 9 | `BUZZ_NOTIF` | |
| 10 | `BUZZ_NAV` | |
| 11 | `BUZZ_MARIO` | meaning unknown |
| 13–15 | `ALERT_PHONE` / `MSG` / `EMAIL` | phone alerts, need Bluetooth |
| 18 | `MUTE` | |
| 37 | `SEGMENTS_MUTED` | |
| 45 | `PLANS_BUZZER` | |
| 79 | `SOUNDS_CFG` | byte array, `[-1]` active, `[-1,124]` per profile; not decoded |

### Display (`<int>`), mapped by changing each option on the device

| ID | Key | Values |
|---|---|---|
| 0 | `BACKLIGHT` | 0 on, 1 5 sec, 2 off, 3 auto |
| 4 | `LED_MODE` | 0 off, 1 speed, 5 power, 6 HR; 2–4 not offered in the device menu |

Top LED toggles (`<boolean>`): 5 `TOP_LED_WORKOUT`, 6 `TOP_LED_NOTIF`, 7 `TOP_LED_NAV`.

### Other

| ID | Key | Notes |
|---|---|---|
| 41 | `PAGE_DEFN` | changes on its own, probably the current page; not decoded |
| 52 | `CFG_PROFILE_NAME` | null-terminated string, e.g. `Cycling\0` |
| 74 | `BOLT_BATTERY_STATUS` | changes on its own (charging) |

## Files on the device

| Path | Contents |
|---|---|
| `/sdcard/exports/` | finished rides, `.fit`, named `YYYY-MM-DD-HHMMSS-ELEMNT ROAM <id>-<n>-0.fit` (start time first) |
| `/sdcard/routes/` | routes, plain `.gpx`. A file pushed here shows up on the device, no restart needed |
| `/sdcard/plans/` | planned workouts (empty here) |
| `/sdcard/maps/` | map tiles |

File names contain spaces, so listing uses `find … -exec stat -c '%s %n'` rather than parsing `ls`.

## Open questions

- `DISPLAY_CFG` header bytes
- Per-page `u8 ?` (zoom?), `u16 ?` (enabled?), flags
- Why the type-39 page has index 0
- Field types 425–429, page type 39: swap one onto a page and see what the device shows
- `PAGE_DEFN` (field 41)
- `SOUNDS_CFG` (field 79), `LED_MODE` values 2–4

Approach: change one thing on the device, pull the live config, diff.

## Not used by flatlinecc

### Backup / restore zip

- Backup: create `/sdcard/config_backup` on the device → `elemnt_config.zip` appears there after a few seconds. It's an export of the live config above.
- Restore: pushing a zip to `/sdcard/config_restore` (with or without reboot) was **not picked up** on current firmware. flatlinecc edits the live config instead.

Zip contents:

| File | Contents |
|---|---|
| `bolt-<profileID>.cfg` | Per-profile settings (pages, alerts, …). One file per profile, e.g. `bolt-65535.cfg` "Cycling", `bolt-65523.cfg` "Indoor Cycling". A profile without custom pages has no `DISPLAY_CFG` field |
| `comp.cfg` | User profile, units, zones |
| `poi.cfg`, `routes.cfg`, `plans.cfg`, … | 2 bytes when empty |

### `.cfg` file

All integers little-endian.

```
header
  u8   version        0 → field IDs are u16, 1 → u32
  u16  profileID      matches the file name (0xFFFF = 65535)
  u16  fieldCount
fields (fieldCount times)
  u16  fieldID        (u32 if version 1)
  u16  size
  [size]u8 data
trailer
  u16  0x0000         in every sample; maybe a count of a second section
                      (Intyre's reader has an unused `timestampFields`)
```
