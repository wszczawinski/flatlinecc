package wahoo

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/wszczawinski/flatlinecc/internal/adb"
)

const (
	// RidesDir is where the device keeps finished rides as .fit files.
	RidesDir = "/sdcard/exports"
	// RoutesDir is where the device reads routes from, as .gpx files.
	RoutesDir = "/sdcard/routes"
)

// File is a file on the device.
type File struct {
	Name string
	Size int64
}

// ListRides returns the .fit files in RidesDir, in no particular order.
// Names start with the ride's start time.
func ListRides(ctx context.Context, serial string) ([]File, error) {
	return listFiles(ctx, serial, RidesDir, "*.fit")
}

// ListRoutes returns the .gpx files in RoutesDir, in no particular order.
func ListRoutes(ctx context.Context, serial string) ([]File, error) {
	return listFiles(ctx, serial, RoutesDir, "*.gpx")
}

// PullRide copies a ride into dir.
func PullRide(ctx context.Context, serial, name, dir string) error {
	return pullInto(ctx, serial, path.Join(RidesDir, name), dir)
}

// PullRoute copies a route from the device into dir.
func PullRoute(ctx context.Context, serial, name, dir string) error {
	return pullInto(ctx, serial, path.Join(RoutesDir, name), dir)
}

// pullInto copies a device file into dir under its own name. It writes a temp
// file and renames it, so an interrupted copy never looks complete.
func pullInto(ctx context.Context, serial, remote, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	name := path.Base(remote)
	tmp := filepath.Join(dir, "."+name+".part")
	if err := adb.PullFile(ctx, serial, remote, tmp); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, name))
}

// PushRoute copies a local .gpx file into RoutesDir under its own name.
func PushRoute(ctx context.Context, serial, local string) error {
	return adb.PushFile(ctx, serial, local, path.Join(RoutesDir, filepath.Base(local)))
}

// HashRoutes returns the SHA-256 of each .gpx file in RoutesDir, by name.
func HashRoutes(ctx context.Context, serial string) (map[string]string, error) {
	out, err := adb.Shell(ctx, serial, "find", RoutesDir, "-maxdepth", "1", "-iname", "'*.gpx'", "-exec", "sha256sum", "{}", "+")
	if err != nil {
		return nil, err
	}
	return parseHashes(out), nil
}

func listFiles(ctx context.Context, serial, dir, pattern string) ([]File, error) {
	// find prints nothing when there are no matches, where a glob would fail.
	// -iname: a route pushed as .GPX must be listed too.
	out, err := adb.Shell(ctx, serial, "find", dir, "-maxdepth", "1", "-iname", "'"+pattern+"'", "-exec", "stat", "-c", "'%s %n'", "{}", "+")
	if err != nil {
		return nil, err
	}
	return parseFiles(out), nil
}

// parseHashes reads sha256sum lines: the hash, two spaces, then the path.
func parseHashes(out string) map[string]string {
	hashes := map[string]string{}
	for line := range strings.SplitSeq(out, "\n") {
		hash, p, ok := strings.Cut(strings.TrimRight(line, "\r"), "  ")
		if ok && len(hash) == 64 {
			hashes[path.Base(p)] = hash
		}
	}
	return hashes
}

// parseFiles reads `stat -c '%s %n'` lines: size, a space, then the path.
func parseFiles(out string) []File {
	var files []File
	for line := range strings.SplitSeq(out, "\n") {
		size, p, ok := strings.Cut(strings.TrimRight(line, "\r"), " ")
		if !ok {
			continue
		}
		n, err := strconv.ParseInt(size, 10, 64)
		if err != nil {
			continue
		}
		files = append(files, File{Name: path.Base(p), Size: n})
	}
	return files
}
