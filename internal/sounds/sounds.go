// Package sounds plays the app's own short sounds.
package sounds

import (
	"embed"
	"log/slog"

	"github.com/egoist/mygo/audio"
)

//go:embed assets/*.wav
var assets embed.FS

// Names are the sounds the settings accept, in menu order. Silence is the
// absence of a sound, and plays nothing.
var Names = []string{"silence", "crystal-glass", "wind-chime", "tic-toc", "reverie"}

// Player holds the decoded sounds. It may be used from any goroutine.
type Player struct {
	log    *slog.Logger
	sounds map[string]*audio.Sound
}

// New decodes the built-in sounds. A sound that fails to decode is logged once
// and plays nothing, which never blocks a break.
func New(log *slog.Logger) *Player {
	p := &Player{log: log, sounds: make(map[string]*audio.Sound, len(Names))}
	for _, name := range Names {
		if name == "silence" {
			continue
		}
		data, err := assets.ReadFile("assets/" + name + ".wav")
		if err != nil {
			log.Warn("sound missing", "sound", name, "error", err)
			continue
		}
		sound, err := audio.DecodeSound(data)
		if err != nil {
			log.Warn("sound could not be decoded", "sound", name, "error", err)
			continue
		}
		p.sounds[name] = sound
	}
	return p
}

// Play plays a sound at a volume; muting, silence and unknown names play
// nothing.
func (p *Player) Play(name string, volume float64, muted bool) {
	if muted || name == "" || name == "silence" {
		return
	}
	sound := p.sounds[name]
	if sound == nil {
		return
	}
	sound.NewPlayer().SetVolume(min(max(volume, 0), 1)).Play()
}

// Available reports whether a sound decoded, for the settings page.
func (p *Player) Available(name string) bool {
	if name == "silence" {
		return true
	}
	_, ok := p.sounds[name]
	return ok
}
