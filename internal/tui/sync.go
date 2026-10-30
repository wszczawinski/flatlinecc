package tui

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/wszczawinski/flatlinecc/internal/wahoo"
)

// newRides returns the rides not copied yet, oldest first, so a stopped copy
// leaves no gaps. Names start with the start time, so name order is time order.
func newRides(rides []wahoo.File, synced map[string]bool) []string {
	var names []string
	for _, ride := range rides {
		if !synced[ride.Name] {
			names = append(names, ride.Name)
		}
	}
	slices.Sort(names)
	return names
}

func copyRide(serial, name, dir string) error {
	ctx, cancel := context.WithTimeout(context.Background(), rideTimeout)
	defer cancel()
	return wahoo.PullRide(ctx, serial, name, dir)
}

// readRoutes lists the routes on both sides.
func readRoutes(serial, dir string) ([]route, error) {
	local, err := localRoutes(dir)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), adbTimeout)
	defer cancel()
	remote, err := wahoo.ListRoutes(ctx, serial)
	if err != nil {
		return nil, err
	}
	routes := mergeRoutes(local, remote)
	return routes, markChanged(ctx, serial, dir, routes)
}

// markChanged compares the hashes of routes that have the same size on both
// sides, so an edit that keeps the size still syncs.
func markChanged(ctx context.Context, serial, dir string, routes []route) error {
	if !slices.ContainsFunc(routes, sameSize) {
		return nil
	}
	remote, err := wahoo.HashRoutes(ctx, serial)
	if err != nil {
		return err
	}
	for i, rt := range routes {
		if !sameSize(rt) {
			continue
		}
		local, err := hashFile(filepath.Join(dir, rt.name))
		if err != nil {
			return err
		}
		routes[i].changed = local != remote[rt.name]
	}
	return nil
}

func sameSize(rt route) bool { return rt.local >= 0 && rt.local == rt.onDev }

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// pendingRoutes returns the routes that sync would copy.
func pendingRoutes(routes []route) []route {
	var pending []route
	for _, rt := range routes {
		if rt.push() || rt.pull() {
			pending = append(pending, rt)
		}
	}
	return pending
}

func syncRoute(serial, dir string, rt route) error {
	ctx, cancel := context.WithTimeout(context.Background(), routeTimeout)
	defer cancel()
	if rt.pull() {
		return wahoo.PullRoute(ctx, serial, rt.name, dir)
	}
	return wahoo.PushRoute(ctx, serial, filepath.Join(dir, rt.name))
}

// syncStartedMsg tells an open Rides or Routes screen to wait for the sync.
type syncStartedMsg struct{ routes bool }

func syncStarted(routes bool) tea.Cmd {
	return func() tea.Msg { return syncStartedMsg{routes} }
}

// syncDoneMsg tells an open Rides or Routes screen the sync is over.
type syncDoneMsg struct{ routes bool }

// syncListedMsg carries the copies one part of the auto sync will make.
type syncListedMsg struct {
	routes bool // routes, else rides
	steps  []func() error
	err    error
}

// syncStepMsg reports one copy.
type syncStepMsg struct {
	routes bool
	err    error
}

// listSync finds what to copy, as the Rides and Routes screens do. The copies
// then run one at a time, so the status can show progress.
func listSync(serial, dataDir string, routes bool) tea.Cmd {
	return func() tea.Msg {
		if routes {
			steps, err := routeSteps(serial, routesDir(dataDir))
			return syncListedMsg{true, steps, err}
		}
		steps, err := rideSteps(serial, ridesDir(dataDir))
		return syncListedMsg{false, steps, err}
	}
}

func rideSteps(serial, dir string) ([]func() error, error) {
	ctx, cancel := context.WithTimeout(context.Background(), adbTimeout)
	rides, err := wahoo.ListRides(ctx, serial)
	cancel()
	if err != nil {
		return nil, err
	}
	var steps []func() error
	for _, name := range newRides(rides, syncedRides(dir, rides)) {
		steps = append(steps, func() error { return copyRide(serial, name, dir) })
	}
	return steps, nil
}

func routeSteps(serial, dir string) ([]func() error, error) {
	routes, err := readRoutes(serial, dir)
	if err != nil {
		return nil, err
	}
	var steps []func() error
	for _, rt := range pendingRoutes(routes) {
		steps = append(steps, func() error { return syncRoute(serial, dir, rt) })
	}
	return steps, nil
}

func runStep(routes bool, step func() error) tea.Cmd {
	return func() tea.Msg { return syncStepMsg{routes, step()} }
}

// syncState is the state of one part of the sync run on connect.
type syncState struct {
	started, listed, done bool
	steps                 []func() error // copies left
	n                     int            // copies made
	err                   error
}

func (s syncState) running() bool { return s.started && !s.done }

// status is "" when the part is off, e.g. status("rides", "new ride", "copied").
func (s syncState) status(name, noun, verb string) string {
	switch {
	case !s.started:
		return ""
	case !s.listed:
		return mutedStyle.Render("… syncing " + name)
	case !s.done:
		return mutedStyle.Render(fmt.Sprintf("… syncing %s %d/%d", name, s.n+1, s.n+len(s.steps)))
	case s.err != nil:
		return errorStyle.Render("✗ " + name + " sync failed: " + s.err.Error())
	case s.n == 0:
		return checkMark + mutedStyle.Render(" "+name+" up to date")
	}
	return checkMark + mutedStyle.Render(fmt.Sprintf(" %s %s", plural(s.n, noun), verb))
}

func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return fmt.Sprintf("%d %ss", n, word)
}
