// Package settings is Fermata's configuration: its JSON form, defaults,
// validation and an atomic, debounced writer.
//
// The package imports only the standard library. The app hands the planner a
// snapshot converted from a Store, so the planner never sees this package.
package settings

import (
	"encoding/json/jsontext"
)

// CurrentVersion is the configuration version this build writes.
const CurrentVersion = 2

// The theme sources.
const (
	ThemeSystem = "system"
	ThemeLight  = "light"
	ThemeDark   = "dark"
)

// The languages the app is written in.
const (
	LanguageSystem  = "system"
	LanguageEnglish = "en"
	LanguageChinese = "zh-Hans"
)

// The tray icon styles.
const (
	StyleMark     = "mark"
	StyleTime     = "time"
	StyleProgress = "progress"
)

// The break field presets, and the custom one.
const (
	FieldPine    = "pine"
	FieldInk     = "ink"
	FieldOxblood = "oxblood"
	FieldSlate   = "slate"
	FieldSand    = "sand"
	FieldPaper   = "paper"
	FieldCustom  = "custom"
)

// DefaultCustomField is the custom break field's color when the setting is
// absent or invalid.
const DefaultCustomField = "#24413A"

// The sounds an app can play, by name.
const (
	SoundSilence      = "silence"
	SoundCrystalGlass = "crystal-glass"
	SoundWindChime    = "wind-chime"
	SoundTicToc       = "tic-toc"
	SoundReverie      = "reverie"
)

// The app exclusion rule kinds.
const (
	RulePause  = "pause"
	RuleResume = "resume"
)

// The displays a break window may target, besides an index.
const (
	ScreenPrimary = "primary"
	ScreenCursor  = "cursor"
	ScreenAll     = "all"
)

// FieldPresets are the break field colors the settings accept, in the order
// the appearance page shows them.
var FieldPresets = []string{FieldPine, FieldInk, FieldOxblood, FieldSlate, FieldSand, FieldPaper}

// SoundNames are the sounds the settings accept.
var SoundNames = []string{SoundSilence, SoundCrystalGlass, SoundWindChime, SoundTicToc, SoundReverie}

// Languages are the values of general.language.
var Languages = []string{LanguageSystem, LanguageEnglish, LanguageChinese}

// Settings is the whole configuration, as config.json holds it.
type Settings struct {
	Version     int         `json:"version"`
	General     General     `json:"general"`
	Schedule    Schedule    `json:"schedule"`
	Presence    Presence    `json:"presence"`
	BreakWindow BreakWindow `json:"breakWindow"`
	Sound       Sound       `json:"sound"`
	Shortcuts   Shortcuts   `json:"shortcuts"`
	Health      Health      `json:"health"`
	FirstRun    bool        `json:"isFirstRun"`

	// Extra keeps the members of the file this build does not know, so that
	// a newer version's settings survive a round trip.
	Extra map[string]jsontext.Value `json:"-"`
}

// General is the settings of the general page.
type General struct {
	OpenAtLogin  bool   `json:"openAtLogin"`
	Language     string `json:"language"`
	Theme        string `json:"theme"`
	MenuBarStyle string `json:"menuBarStyle"`
	// ShowTrayMenuInStrictMode keeps settings and about in the menu of a
	// strict break, where the menu otherwise shows its status alone.
	ShowTrayMenuInStrictMode bool `json:"showTrayMenuInStrictMode"`
}

// Shortcuts are keyboard shortcuts.
type Shortcuts struct {
	EndBreak string `json:"endBreak"`
}

// Schedule is the rhythm of the two kinds of breaks.
type Schedule struct {
	Mini           MiniBreaks `json:"mini"`
	Long           LongBreaks `json:"long"`
	PauseDurations []int      `json:"pauseDurationsMs"`
	MorningHour    int        `json:"morningHour"`
}

// Postpone is how one kind of break may be postponed.
type Postpone struct {
	Enabled bool `json:"enabled"`
	TimeMs  int  `json:"timeMs"`
	Limit   int  `json:"limit"`
	Percent int  `json:"percent"`
}

// MiniBreaks is the short break settings.
type MiniBreaks struct {
	Enabled      bool     `json:"enabled"`
	DurationMs   int      `json:"durationMs"`
	IntervalMs   int      `json:"intervalMs"`
	Notify       bool     `json:"notify"`
	NotifyLeadMs int      `json:"notifyLeadMs"`
	Postpone     Postpone `json:"postpone"`
	Strict       bool     `json:"strict"`
	ManualFinish bool     `json:"manualFinish"`
}

// LongBreaks is the long break settings.
type LongBreaks struct {
	Enabled    bool `json:"enabled"`
	DurationMs int  `json:"durationMs"`
	// EveryNBreaks is how many short breaks come before a long one; unused
	// when short breaks are off.
	EveryNBreaks int      `json:"everyNBreaks"`
	IntervalMs   int      `json:"intervalMs"`
	Notify       bool     `json:"notify"`
	NotifyLeadMs int      `json:"notifyLeadMs"`
	Postpone     Postpone `json:"postpone"`
	Strict       bool     `json:"strict"`
	ManualFinish bool     `json:"manualFinish"`
}

// Presence is what pauses breaks besides the menu.
type Presence struct {
	Idle        Idle `json:"idle"`
	DND         DND  `json:"dnd"`
	PauseOnLock bool `json:"pauseOnLock"`
	// Fullscreen holds a due break while an app is fullscreen.
	Fullscreen bool `json:"fullscreen"`
	// WaitWhileTyping holds a due break until the user pauses typing.
	WaitWhileTyping bool           `json:"waitWhileTyping"`
	AppExclusions   []AppExclusion `json:"appExclusions"`
}

// Idle pauses breaks while the user is away.
type Idle struct {
	Enabled         bool `json:"enabled"`
	ResetAfterMs    int  `json:"resetAfterMs"`
	CheckIntervalMs int  `json:"checkIntervalMs"`
}

// DND pauses breaks while the system's Focus or Do Not Disturb is on.
type DND struct {
	Enabled         bool `json:"enabled"`
	CheckIntervalMs int  `json:"checkIntervalMs"`
}

// AppExclusion pauses, or allows, breaks while an application runs.
type AppExclusion struct {
	Name     string   `json:"name"`
	Commands []string `json:"commands"`
	Rule     string   `json:"rule"`
	Active   bool     `json:"active"`
}

// BreakWindow is the look and placement of the break windows.
type BreakWindow struct {
	AllScreens      bool    `json:"allScreens"`
	Screen          string  `json:"screen"`
	ContentScreen   string  `json:"contentScreen"`
	FillScreen      bool    `json:"fillScreen"`
	WidthRatio      float64 `json:"widthRatio"`
	HeightRatio     float64 `json:"heightRatio"`
	Field           string  `json:"field"`
	CustomField     string  `json:"customField"`
	ShowIdeas       bool    `json:"showIdeas"`
	ShowCurrentTime bool    `json:"showCurrentTime"`
	// Transparent, Opacity and BackgroundImage are only read from the file.
	Transparent     bool    `json:"transparent"`
	Opacity         float64 `json:"opacity"`
	BackgroundImage bool    `json:"backgroundImage"`
}

// Sound is the app's own sounds.
type Sound struct {
	Muted     bool    `json:"muted"`
	Volume    float64 `json:"volume"`
	MiniStart string  `json:"miniStart"`
	LongStart string  `json:"longStart"`
	MiniEnd   string  `json:"miniEnd"`
	LongEnd   string  `json:"longEnd"`
}

// Health is the break health counter's setting.
type Health struct {
	Mode bool `json:"mode"`
}

// Default returns the settings of a first run.
func Default() Settings {
	return Settings{
		Version: CurrentVersion,
		General: General{
			OpenAtLogin:  false,
			Language:     LanguageSystem,
			Theme:        ThemeSystem,
			MenuBarStyle: StyleMark,
		},
		Schedule: Schedule{
			Mini: MiniBreaks{
				Enabled:      true,
				DurationMs:   300_000,
				IntervalMs:   1_500_000,
				Notify:       true,
				NotifyLeadMs: 10_000,
				Postpone:     Postpone{Enabled: true, TimeMs: 120_000, Limit: 1, Percent: 30},
			},
			Long: LongBreaks{
				Enabled:      true,
				DurationMs:   900_000,
				EveryNBreaks: 4,
				IntervalMs:   6_000_000,
				Notify:       true,
				NotifyLeadMs: 30_000,
				Postpone:     Postpone{Enabled: true, TimeMs: 300_000, Limit: 1, Percent: 30},
			},
			PauseDurations: []int{1_800_000, 3_600_000, 7_200_000, 18_000_000},
			MorningHour:    6,
		},
		Presence: Presence{
			Idle:            Idle{Enabled: true, ResetAfterMs: 300_000, CheckIntervalMs: 2_000},
			DND:             DND{Enabled: true, CheckIntervalMs: 2_000},
			PauseOnLock:     true,
			Fullscreen:      true,
			WaitWhileTyping: false,
		},
		BreakWindow: BreakWindow{
			AllScreens:    true,
			Screen:        ScreenPrimary,
			ContentScreen: ScreenAll,
			FillScreen:    false,
			WidthRatio:    0.85,
			HeightRatio:   0.85,
			Field:         FieldPine,
			CustomField:   DefaultCustomField,
			ShowIdeas:     true,
			Opacity:       0.9,
		},
		Sound: Sound{
			Muted:     false,
			Volume:    1,
			MiniStart: SoundSilence,
			LongStart: SoundSilence,
			MiniEnd:   SoundCrystalGlass,
			LongEnd:   SoundCrystalGlass,
		},
		Shortcuts: Shortcuts{EndBreak: "CmdOrCtrl+X"},
		Health:    Health{Mode: false},
		FirstRun:  true,
	}
}
