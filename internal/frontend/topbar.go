package frontend

import (
	"fmt"

	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

type TopBar struct {
	app.Compo
	ShowLogout bool
}

func (t *TopBar) onToggleSound(ctx app.Context, e app.Event) {
	e.PreventDefault()
	State.ToggleSound()
}

func (t *TopBar) onLogout(ctx app.Context, e app.Event) {
	e.PreventDefault()
	State.Player = nil
	app.Window().Get("document").Set("cookie", "gospot_player=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;")
	State.SyncMusic()
	ctx.Navigate("/")
}

func (t *TopBar) onBannerClick(ctx app.Context, e app.Event) {
	ctx.Navigate("/")
}

func (t *TopBar) Render() app.UI {
	soundIcon := "🔊"
	if !State.SoundEnabled {
		soundIcon = "🔇"
	}

	actions := []app.UI{
		app.Li().Body(
			app.A().
				Href("#").
				OnClick(t.onToggleSound).
				Style("text-decoration", "none").
				Body(
					app.Span().
						Class("sound-icon").
						Style("font-family", "system-ui").
						Text(soundIcon),
				),
		),
		app.Li().Body(
			app.Span().Style("margin-right", "8px").Text(State.Player.Name),
			app.Img().
				Src(fmt.Sprintf("/web/images/symbol_%02d.png", State.Player.Symbol)).
				Style("width", "32px").Style("height", "32px").Style("vertical-align", "middle"),
		),
	}

	if t.ShowLogout {
		actions = append(actions, app.Li().Body(app.A().Href("#").OnClick(t.onLogout).Text("Logout")))
	}

	return app.Nav().Body(
		app.Ul().Body(
			app.Li().Body(
				app.Img().
					Src("/web/images/banner.png").
					Style("height", "2rem").
					Style("vertical-align", "middle").
					Style("cursor", "pointer").
					Style("border-radius", "8px").
					OnClick(t.onBannerClick),
			),
		),
		app.Ul().Body(actions...),
	)
}
