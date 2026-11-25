// Package ideas keeps the break suggestions: a deck that is shuffled, drawn
// without repeating the one before, and shuffled again when it runs out.
package ideas

import (
	"embed"
	"encoding/json/v2"
	"math/rand/v2"
	"sync"

	"github.com/gythialy/fermata/internal/planner"
)

//go:embed assets/*.json
var assets embed.FS

// DefaultLanguage is the deck used when a language has none.
const DefaultLanguage = "en"

// Idea is one suggestion. A mini break's is Text; a long break's has a Title
// and a Text under it.
type Idea struct {
	Title string
	Text  string
}

// Deck draws ideas in one language.
type Deck struct {
	mu   sync.Mutex
	lang string
	mini []Idea
	long []Idea
	rng  *rand.Rand

	miniBag, longBag   []int
	lastMini, lastLong int
}

type catalog struct {
	Mini []string `json:"mini"`
	Long []struct {
		Title string `json:"title"`
		Text  string `json:"text"`
	} `json:"long"`
}

// NewDeck returns a deck in lang, falling back to English.
func NewDeck(lang string) *Deck {
	d := &Deck{rng: rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())), lastMini: -1, lastLong: -1}
	if err := d.load(lang); err != nil {
		_ = d.load(DefaultLanguage)
	}
	return d
}

// SetLanguage loads another language's ideas.
func (d *Deck) SetLanguage(lang string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.lang == normalize(lang) && len(d.mini) > 0 {
		return nil
	}
	return d.load(lang)
}

// Language returns the deck's language.
func (d *Deck) Language() string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.lang
}

// Next draws an idea for a break kind.
func (d *Deck) Next(kind planner.Kind) Idea {
	d.mu.Lock()
	defer d.mu.Unlock()
	if kind == planner.Long {
		return draw(d.long, &d.longBag, &d.lastLong, d.rng)
	}
	return draw(d.mini, &d.miniBag, &d.lastMini, d.rng)
}

func (d *Deck) load(lang string) error {
	lang = normalize(lang)
	data, err := assets.ReadFile("assets/" + lang + ".json")
	if err != nil {
		if lang == DefaultLanguage {
			d.use(builtinMini, builtinLong)
			return err
		}
		lang = DefaultLanguage
		data, err = assets.ReadFile("assets/" + lang + ".json")
		if err != nil {
			d.use(builtinMini, builtinLong)
			return err
		}
	}
	var c catalog
	if err := json.Unmarshal(data, &c); err != nil {
		d.use(builtinMini, builtinLong)
		return err
	}
	mini := make([]Idea, 0, len(c.Mini))
	for _, text := range c.Mini {
		if text != "" {
			mini = append(mini, Idea{Text: text})
		}
	}
	long := make([]Idea, 0, len(c.Long))
	for _, l := range c.Long {
		if l.Text != "" {
			long = append(long, Idea{Title: l.Title, Text: l.Text})
		}
	}
	if len(mini) == 0 || len(long) == 0 {
		d.use(builtinMini, builtinLong)
		return nil
	}
	d.lang = lang
	d.use(mini, long)
	return nil
}

// use installs loaded ideas and starts fresh bags, so that a language change
// reshuffles.
func (d *Deck) use(mini, long []Idea) {
	d.mini, d.long = mini, long
	d.miniBag, d.longBag = nil, nil
	d.lastMini, d.lastLong = -1, -1
}

// draw takes the next idea from a bag, refusing to repeat the previous one.
func draw(deck []Idea, bag *[]int, last *int, rng *rand.Rand) Idea {
	if len(deck) == 0 {
		return Idea{}
	}
	if len(*bag) == 0 {
		*bag = rng.Perm(len(deck))
		if len(deck) > 1 && (*bag)[0] == *last {
			(*bag)[0], (*bag)[1] = (*bag)[1], (*bag)[0]
		}
	}
	i := (*bag)[0]
	*bag = (*bag)[1:]
	*last = i
	return deck[i]
}

func normalize(lang string) string {
	switch lang {
	case "zh-Hans", "zh-CN", "zh":
		return "zh-Hans"
	default:
		return DefaultLanguage
	}
}

// builtinMini and builtinLong keep the break window usable if the embedded
// assets are broken.
var builtinMini = []Idea{
	{Text: "Look out a window until your eyes relax."},
	{Text: "Stand up and let your shoulders drop."},
	{Text: "Drink a glass of water."},
}

var builtinLong = []Idea{
	{Title: "Step away", Text: "Walk to a window and look at something far away."},
	{Title: "Move a little", Text: "Take a short walk, even if it is just down the hall."},
}
