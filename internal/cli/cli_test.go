package cli

import (
	"errors"
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want Command
	}{
		{"nothing", nil, Command{Name: CmdStart}},
		{"help", []string{"help"}, Command{Name: CmdHelp}},
		{"help flag", []string{"-h"}, Command{Name: CmdHelp}},
		{"version", []string{"version"}, Command{Name: CmdVersion}},
		{"resume", []string{"resume"}, Command{Name: CmdResume}},
		{"toggle", []string{"toggle"}, Command{Name: CmdToggle}},
		{"reset", []string{"reset"}, Command{Name: CmdReset}},
		{"skip", []string{"skip"}, Command{Name: CmdSkip}},
		{"postpone", []string{"postpone"}, Command{Name: CmdPostpone}},
		{"finish", []string{"finish"}, Command{Name: CmdFinish}},
		{"last", []string{"last"}, Command{Name: CmdLast}},
		{"preferences", []string{"preferences"}, Command{Name: CmdPreferences}},
		{"pause forever", []string{"pause"}, Command{Name: CmdPause, Indefinite: true}},
		{"pause explicit", []string{"pause", "-d", "indefinitely"}, Command{Name: CmdPause, Indefinite: true}},
		{"pause 30", []string{"pause", "-d", "30"}, Command{Name: CmdPause, Duration: 30 * time.Minute}},
		{"pause positional", []string{"pause", "30"}, Command{Name: CmdPause, Duration: 30 * time.Minute}},
		{"pause positional units", []string{"pause", "1h20m"}, Command{Name: CmdPause, Duration: 80 * time.Minute}},
		{"pause hours", []string{"pause", "--duration", "1h20m"}, Command{Name: CmdPause, Duration: 80 * time.Minute}},
		{"pause equals", []string{"pause", "-d=2h"}, Command{Name: CmdPause, Duration: 2 * time.Hour}},
		{"pause morning", []string{"pause", "-d", "until-morning"}, Command{Name: CmdPause, UntilMorning: true}},
		{"mini", []string{"mini"}, Command{Name: CmdMini}},
		{"mini title", []string{"mini", "-T", "Look away"}, Command{Name: CmdMini, Title: "Look away"}},
		{"mini no skip", []string{"mini", "-n"}, Command{Name: CmdMini, NoSkip: true}},
		{"mini wait", []string{"mini", "-w", "20m"}, Command{Name: CmdMini, Wait: 20 * time.Minute}},
		{"long", []string{"long", "-T", "Step away", "-t", "Walk to a window.", "-w", "1h"}, Command{Name: CmdLong, Title: "Step away", Text: "Walk to a window.", Wait: time.Hour}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse(tc.args)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"unknown", []string{"dance"}},
		{"resume args", []string{"resume", "now"}},
		{"skip args", []string{"skip", "-d", "30"}},
		{"pause missing value", []string{"pause", "-d"}},
		{"pause bad duration", []string{"pause", "-d", "soon"}},
		{"pause too many", []string{"pause", "30", "60"}},
		{"mini text", []string{"mini", "-t", "no text"}},
		{"mini both", []string{"mini", "-n", "-w", "5m"}},
		{"mini bad wait", []string{"mini", "-w", "later"}},
		{"long missing title", []string{"long", "-T"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Parse(tc.args); !errors.Is(err, ErrUsage) {
				t.Fatalf("error %v is not a usage error", err)
			}
		})
	}
}

func TestDuration(t *testing.T) {
	tests := []struct {
		in   string
		want time.Duration
	}{
		{"30", 30 * time.Minute},
		{"1", time.Minute},
		{"90m", 90 * time.Minute},
		{"1h", time.Hour},
		{"1h20m", 80 * time.Minute},
		{"1h 20m", 80 * time.Minute},
		{"2h30m15s", 2*time.Hour + 30*time.Minute + 15*time.Second},
		{"45s", 45 * time.Second},
	}
	for _, tc := range tests {
		got, err := Duration(tc.in)
		if err != nil {
			t.Errorf("%q: %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("%q: %v, want %v", tc.in, got, tc.want)
		}
	}
	for _, in := range []string{"", "0", "abc", "1h20", "-5", "1x"} {
		if _, err := Duration(in); err == nil {
			t.Errorf("%q parsed", in)
		}
	}
}

func TestUntilMorning(t *testing.T) {
	day := time.Date(2026, 11, 25, 0, 0, 0, 0, time.Local)
	tests := []struct {
		now  time.Time
		want time.Time
	}{
		{day.Add(5 * time.Hour), day.Add(6 * time.Hour)},
		{day.Add(7 * time.Hour), day.Add(30 * time.Hour)},
		{day.Add(6 * time.Hour), day.Add(30 * time.Hour)},
	}
	for _, tc := range tests {
		if got := UntilMorning(tc.now, 6); !got.Equal(tc.want) {
			t.Errorf("%v: %v, want %v", tc.now, got, tc.want)
		}
	}
}

func TestString(t *testing.T) {
	tests := []struct {
		in   time.Duration
		want string
	}{
		{0, "0:00"},
		{14 * time.Second, "0:14"},
		{-3 * time.Second, "0:00"},
		{time.Hour + 2*time.Minute + 3*time.Second, "1:02:03"},
		{5 * time.Minute, "5:00"},
	}
	for _, tc := range tests {
		if got := String(tc.in); got != tc.want {
			t.Errorf("%v: %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestVersionLine(t *testing.T) {
	if got := Version("0.1.0", ""); got != "Fermata 0.1.0" {
		t.Errorf("got %q", got)
	}
	if got := Version("0.1.0", "abc1234"); got != "Fermata 0.1.0 (abc1234)" {
		t.Errorf("got %q", got)
	}
}
