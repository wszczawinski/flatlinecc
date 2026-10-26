package wahoo

// Setting is a value in the live config, by its entry number (the "type" in
// its key, see docs/format.md). These are not data field IDs.
type Setting struct {
	ID   int
	Name string
}

// Sounds are the sound toggles, all <boolean> entries. Phone, message and
// email alerts are left out: they need Bluetooth.
var Sounds = []Setting{
	{18, "Mute all sounds"},
	{8, "Workout buzzer"},
	{9, "Notification buzzer"},
	{10, "Navigation buzzer"},
	{45, "Planned workout buzzer"},
	{37, "Mute segments"},
	{11, "Mario tune"},
}
