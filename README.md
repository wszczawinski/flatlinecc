# flatlinecc

A terminal tool to manage a Wahoo ELEMNT ROAM v1 over USB, without the phone app.

## Why

I crashed with my ROAM and its Bluetooth stopped working. The device still records rides fine, but without Bluetooth the companion app can't reach it: no screen setup, no route or ride sync. flatlinecc brings that back over the cable.

## Plan

Run `flatlinecc` and pick from a menu:

- **Screens**: change data fields on each page, add or remove fields
- **Sounds**: alert and buzzer settings
- **Rides**: copy new rides to a local folder
- **Routes**: push new routes to the device
- **Backup / restore**: device config
- **Device info**
- **Settings**: local data folder (default `./data`)

Built with Go and [Charm](https://charm.sh) (Bubble Tea, Bubbles, Lip Gloss, Huh). Talks to the device through `adb`.

See [PLAN.md](PLAN.md) for details and milestones.

## Development

Requires Go and `adb` (`brew install go android-platform-tools`).

```
go mod tidy                                  # sync dependencies with imports
go run ./cmd/flatlinecc                      # compile to a temp binary and run
go build -o flatlinecc ./cmd/flatlinecc      # build ./flatlinecc
go install ./cmd/flatlinecc                  # install to ~/go/bin (add to PATH)
```

Layout:

```
cmd/flatlinecc/   # entry point (package main)
internal/adb/     # device access
internal/config/  # settings
internal/tui/     # menu and screens
```

## Status

Early work in progress.

## Credits

Builds on reverse-engineering notes by [Intyre](https://gist.github.com/Intyre).
