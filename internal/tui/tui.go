// Package tui hosts the interactive charmbracelet/huh forms.
package tui

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"

	"github.com/zachicecreamcohn/soute/internal/config"
	"github.com/zachicecreamcohn/soute/internal/manifest"
	"github.com/zachicecreamcohn/soute/internal/paths"
	"github.com/zachicecreamcohn/soute/internal/render"
)

// ErrAborted is returned when the user cancels a form before submitting.
var ErrAborted = errors.New("aborted")

// Init runs the setup wizard and returns the resulting configuration. A
// non-nil prefill reopens the wizard for editing existing settings.
func Init(prefill *config.Config) (*config.Config, error) {
	cfg := config.Default()
	useDefaults := true
	if prefill != nil {
		cfg = *prefill
		useDefaults = false
	}

	target := cfg.TargetPath
	if err := run(huh.NewForm(huh.NewGroup(
		targetField(&target),
		defaultsField(&useDefaults),
	))); err != nil {
		return nil, err
	}
	cfg.TargetPath = paths.CleanInput(target)

	if !useDefaults {
		if err := numberFields(&cfg); err != nil {
			return nil, err
		}
	}
	return &cfg, nil
}

// Settings runs the settings-only wizard (defaults and thresholds), skipping
// the target path prompt.
func Settings(prefill *config.Config) (*config.Config, error) {
	cfg := config.Default()
	useDefaults := true
	if prefill != nil {
		cfg = *prefill
	}

	if err := run(huh.NewForm(huh.NewGroup(defaultsField(&useDefaults)))); err != nil {
		return nil, err
	}
	if !useDefaults {
		if err := numberFields(&cfg); err != nil {
			return nil, err
		}
	}
	return &cfg, nil
}

// SelectSnapshot presents the history (newest first) for selection.
func SelectSnapshot(snapshots []manifest.Snapshot) (*manifest.Snapshot, error) {
	if len(snapshots) == 0 {
		return nil, errors.New("no snapshots available")
	}

	var selected string
	opts := make([]huh.Option[string], 0, len(snapshots))
	for i := len(snapshots) - 1; i >= 0; i-- {
		s := snapshots[i]
		opts = append(opts, huh.NewOption(snapshotLabel(s), s.ID))
	}

	if err := run(huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().
			Title("Select snapshot").
			Options(opts...).
			Value(&selected),
	))); err != nil {
		return nil, err
	}

	for i := range snapshots {
		if snapshots[i].ID == selected {
			return &snapshots[i], nil
		}
	}
	return nil, fmt.Errorf("snapshot %q not found", selected)
}

// InputPath prompts for a filesystem path.
func InputPath(title string) (string, error) {
	var v string
	if err := run(huh.NewForm(huh.NewGroup(
		huh.NewInput().
			Title(title).
			Value(&v).
			Validate(func(s string) error {
				if strings.TrimSpace(s) == "" {
					return errors.New("path is required")
				}
				return nil
			}),
	))); err != nil {
		return "", err
	}
	return paths.CleanInput(v), nil
}

func snapshotLabel(s manifest.Snapshot) string {
	return fmt.Sprintf("%s  %s  %s  %+.1f%%  %s",
		s.ID, render.ShortTime(s.Timestamp), render.HumanBytes(s.SizeBytes), s.DeltaPct, s.Tag)
}

func run(form *huh.Form) error {
	if err := form.WithTheme(huh.ThemeBase()).Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return ErrAborted
		}
		return err
	}
	return nil
}

func targetField(v *string) *huh.Input {
	return huh.NewInput().
		Title("Target file").
		Description("Path to the media/bundle file to watch").
		Placeholder("./main_show.qlab5").
		Value(v).
		Validate(func(s string) error {
			c := paths.CleanInput(s)
			if c == "" {
				return errors.New("target path is required")
			}
			if strings.Contains(c, "\"") {
				return errors.New("target path must not contain double quotes")
			}
			return nil
		})
}

func defaultsField(v *bool) *huh.Confirm {
	return huh.NewConfirm().
		Title("Use recommended defaults?").
		Description("1 MiB min delta · 2.5% min delta · 5s cooldown · 50 snapshots").
		Value(v)
}

func numberFields(cfg *config.Config) error {
	minBytes := strconv.FormatInt(cfg.MinDeltaBytes, 10)
	minPct := strconv.FormatFloat(cfg.MinDeltaPct, 'f', 2, 64)
	cooldown := strconv.Itoa(cfg.CooldownSeconds)
	maxSnaps := strconv.Itoa(cfg.MaxSnapshots)

	if err := run(huh.NewForm(huh.NewGroup(
		huh.NewInput().Title("Minimum delta (bytes)").Value(&minBytes).Validate(intValidator("bytes")),
		huh.NewInput().Title("Minimum delta (%)").Value(&minPct).Validate(floatValidator("%")),
		huh.NewInput().Title("Cooldown (seconds)").Value(&cooldown).Validate(intValidator("seconds")),
		huh.NewInput().Title("Max snapshots").Value(&maxSnaps).Validate(intValidator("snapshots")),
	))); err != nil {
		return err
	}

	cfg.MinDeltaBytes, _ = strconv.ParseInt(strings.TrimSpace(minBytes), 10, 64)
	cfg.MinDeltaPct, _ = strconv.ParseFloat(strings.TrimSpace(minPct), 64)
	cfg.CooldownSeconds, _ = strconv.Atoi(strings.TrimSpace(cooldown))
	cfg.MaxSnapshots, _ = strconv.Atoi(strings.TrimSpace(maxSnaps))
	return nil
}

func intValidator(unit string) func(string) error {
	return func(s string) error {
		if _, err := strconv.Atoi(strings.TrimSpace(s)); err != nil {
			return fmt.Errorf("must be an integer (%s)", unit)
		}
		return nil
	}
}

func floatValidator(unit string) func(string) error {
	return func(s string) error {
		if _, err := strconv.ParseFloat(strings.TrimSpace(s), 64); err != nil {
			return fmt.Errorf("must be a number (%s)", unit)
		}
		return nil
	}
}
