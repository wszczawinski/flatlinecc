## [1.0.0] - 2026-10-27

### 📚 Documentation

- Add MIT license, fold plan into README and format notes

### ⚙️ Miscellaneous Tasks

- Run tests on push
- Update checkout and setup-go actions
## [0.5.0] - 2026-10-26

### 🚀 Features

- Add -version flag

### 🐛 Bug Fixes

- *(tui)* Enable apply when the session backup finishes after a screen opened
- *(backup)* Retry the session backup when the first attempt fails
- *(wahoo)* Take the config owner from its folder, list .GPX routes

### 📚 Documentation

- Update features, session backup notes and later ideas
- Add install and data folder sections, reorder format notes, trim plan
- Add screenshots to the README

### 🚜 Refactor

- *(tui)* Drop placeholder, share list helpers, test App
- Drop unused .cfg code, share list and apply helpers, split tui files
## [0.4.0] - 2026-10-24

### 🚀 Features

- Copy new rides from the device and sync routes both ways

### 📚 Documentation

- Rides and routes, device file paths, all milestones done

### 🚜 Refactor

- *(tui)* Share list scrolling between screens
## [0.3.0] - 2026-10-24

### 🚀 Features

- *(screens)* Add, delete and reorder fields on a page
- *(sounds)* Toggle buzzers and mute from the TUI
- *(backup)* Manual backups, restore and delete from the TUI

### 🐛 Bug Fixes

- *(sounds)* Write only changed toggles

### 📚 Documentation

- Sounds, backlight and LED fields; mark milestones done
## [0.2.0] - 2026-10-24

### 🚀 Features

- *(cfg)* Read and write screen layouts in the live device config
- *(screens)* List pages and fields from the device
- *(fields)* Rename pages, hide unused fields, page-only field lists
- *(screens)* Pick fields and apply changes, with session backup

### 🐛 Bug Fixes

- *(screens)* Keep hidden current field in picker, ignore enter on empty page

### 📚 Documentation

- Update layout, apply flow, milestones and test commands
## [0.1.0] - 2026-10-21

### 🚀 Features

- Add initial TUI menu skeleton
- *(tui)* Add screen navigation and styling
- *(tui)* Add navigation, styling and settings screen
- *(adb)* Add device connection, connect screen and device info

### 📚 Documentation

- Add ROAM connection steps and project conventions

### ⚙️ Miscellaneous Tasks

- Add git-cliff config and release docs
