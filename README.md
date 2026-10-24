# flatlinecc

A terminal tool to manage a Wahoo ELEMNT ROAM v1 over USB, without the phone app.

## Why

I crashed with my ROAM and its Bluetooth stopped working. The device still records rides fine, but without Bluetooth the companion app can't reach it: no screen setup, no route or ride sync. flatlinecc brings that back over the cable.

## Features

Run `flatlinecc` and pick from a menu:

- **Screens**: change, add, remove and reorder data fields on each page
- **Sounds**: mute and buzzer settings
- **Rides**: copy new rides to a local folder
- **Routes**: sync `.gpx` routes both ways between a local folder and the device
- **Backup / restore**: save and restore the device config. It is also backed up automatically when a session starts
- **Device info**
- **Settings**: local data folder (default `./data`, relative to where you run it; it holds your rides and config backups, so don't run it inside another git repo, or set an absolute path)

Built with Go and [Charm](https://charm.sh) (Bubble Tea, Bubbles, Lip Gloss). Talks to the device through `adb`.

`flatlinecc -version` prints the version. See [PLAN.md](PLAN.md) for details and milestones.

## Connecting the ROAM

The ROAM must authorize your computer for `adb`. This has to be repeated every time you plug it in:

1. Start the adb server once (`adb devices`) so your key exists in `~/.android/adbkey`
2. Unplug the ROAM (keep it on)
3. Press **power** (opens settings), then **power** again (back)
4. Plug in the USB cable

`adb devices -l` should now show `device` instead of `unauthorized`.

## Development

Requires Go and `adb` (`brew install go android-platform-tools`).

```
go mod tidy                                  # sync dependencies with imports
go run ./cmd/flatlinecc                      # compile to a temp binary and run
go test -race ./...                          # run tests
go test ./internal/cfg -fuzz FuzzParseDisplay # fuzz a parser (ctrl+c to stop)
go build -o flatlinecc ./cmd/flatlinecc      # build ./flatlinecc
go install ./cmd/flatlinecc                  # install to ~/go/bin (add to PATH)
```

Layout:

```
cmd/flatlinecc/   # entry point (package main)
internal/adb/     # device access
internal/cfg/     # .cfg and DISPLAY_CFG binary formats
internal/config/  # settings
internal/fields/  # field and page names
internal/prefs/   # values in the live config XML
internal/tui/     # menu and screens
internal/wahoo/   # live config, rides and routes on the device
docs/format.md    # reverse-engineered format notes
```

## Releasing

[Conventional Commits](https://www.conventionalcommits.org) + [SemVer](https://semver.org). [git-cliff](https://git-cliff.org) generates `CHANGELOG.md` and picks the next version from commits since the last tag.

```
git cliff --bump -o CHANGELOG.md
git add CHANGELOG.md
git commit -m "chore(release): prepare for vX.Y.Z"
git tag vX.Y.Z
git push && git push --tags
```

## Status

All menu items work. Open questions about the config format are in [docs/format.md](docs/format.md).

## Credits

Builds on reverse-engineering notes by [Intyre](https://gist.github.com/Intyre).
