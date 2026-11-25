package prefs

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/gythialy/fermata/internal/theme"
	"github.com/gythialy/fermata/locales"
)

// aboutPage draws the app's identity and the one reset that undoes everything.
func (w *Window) aboutPage(c *ui.Context, t *ui.Theme) {
	ui.Column(c).FillWidth().AlignItems(ui.Center).Padding(0, 0, 8, 0).Children(func() {
		bg, mark, dot := theme.AppIcon(t.Dark)
		tileBorder := ui.Hex("#DCDAD1")
		if t.Dark {
			tileBorder = ui.Hex("#2E342F")
		}
		ui.Box(c).Size(72, 72).Radius(16).Background(bg).
			Border(1, tileBorder).Draw(func(p *ui.Painter, r ui.Rect) {
			inset := r.W * 0.16
			theme.PaintMark(p, ui.Rect{X: r.X + inset, Y: r.Y + inset, W: r.W - 2*inset, H: r.H - 2*inset}, mark, dot)
		}).Margin(14, 0, 0, 0)

		ui.Text(c, locales.AppName()).FontSize(theme.Px(c, 22)).FixedLineHeight(theme.Px(c, 28)).Bold().LetterSpacing(theme.Px(c, -0.22)).
			Margin(14, 0, 0, 0)
		ui.Text(c, locales.AppTagline()).Font(theme.Serif).FontSize(theme.Px(c, 13)).FixedLineHeight(theme.Px(c, 20)).
			TextColor(t.TextMuted).Margin(3, 0, 0, 0)

		version, build := w.opts.Version()
		ui.Text(c, locales.AboutVersion(version, build)).Font(theme.Mono).FontSize(theme.Px(c, 10)).
			FixedLineHeight(15).TextColor(t.TextMuted).Margin(16, 0, 0, 0)
		ui.Text(c, locales.AboutInspired()).FontSize(theme.Px(c, 11)).FixedLineHeight(theme.Px(c, 16)).
			TextColor(t.TextMuted).Margin(3, 0, 0, 0)

		ui.Row(c).Gap(14).AlignItems(ui.Center).Margin(14, 0, 0, 0).Children(func() {
			ui.Link(c, locales.AboutWebsite(), w.opts.Website).FontSize(theme.Px(c, 11)).FixedLineHeight(theme.Px(c, 16))
			ui.Text(c, "·").FontSize(theme.Px(c, 11)).TextColor(t.Border)
			ui.Link(c, locales.AboutSource(), w.opts.Source).FontSize(theme.Px(c, 11)).FixedLineHeight(theme.Px(c, 16))
			ui.Text(c, "·").FontSize(theme.Px(c, 11)).TextColor(t.Border)
			ui.Link(c, locales.AboutReport(), w.opts.Issues).FontSize(theme.Px(c, 11)).FixedLineHeight(theme.Px(c, 16))
		})

		if version, url := w.opts.Update(); version != "" {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Margin(12, 0, 0, 0).Children(func() {
				ui.Text(c, locales.AboutUpdateAvailable(version)).FontSize(theme.Px(c, 11)).FixedLineHeight(theme.Px(c, 16)).
					TextColor(t.Warning)
				ui.Link(c, locales.AboutReleaseNotes(), url).FontSize(theme.Px(c, 11)).FixedLineHeight(theme.Px(c, 16))
			})
		}

		ui.Box(c).Width(300).Height(1).Background(t.Border).Margin(24, 0, 0, 0)

		restore := textButton(c, t, locales.SettingsRestore(), true)
		restore.Margin(16, 0, 0, 0)
		if restore.Clicked() {
			w.confirmReset()
		}

		ui.Text(c, locales.AboutFooter()).FontSize(theme.Px(c, 10)).FixedLineHeight(theme.Px(c, 15)).
			TextColor(t.TextMuted.Alpha(0.8)).Margin(32, 0, 0, 0)
	})
}

// confirmReset asks before it throws the settings away.
func (w *Window) confirmReset() {
	res, err := mygo.Dialog.Message(mygo.MessageOptions{
		Type:          mygo.MessageQuestion,
		Title:         locales.DialogRestoreTitle(),
		Message:       locales.DialogRestoreBody(),
		Buttons:       []string{locales.DialogCancel(), locales.DialogRestore()},
		DefaultButton: 0,
		CancelButton:  0,
	})
	if err != nil {
		w.opts.Log.Warn("restore defaults dialog", "error", err)
		return
	}
	if res.Button == 1 {
		w.opts.Reset()
	}
}
