package frontend

import (
	"fmt"
	"time"

	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

type TopBar struct {
	app.Compo
	ShowLogout bool

	// Timer state
	timeDisplay string
	ticker      *time.Ticker
	closeTicker chan struct{}
}

func (t *TopBar) OnMount(ctx app.Context) {
	State.Listeners["topbar"] = func() {
		ctx.Dispatch(func(ctx app.Context) {
			t.checkTimer(ctx)
		})
	}
	t.checkTimer(ctx)
}

func (t *TopBar) OnDismount() {
	delete(State.Listeners, "topbar")
	if t.ticker != nil {
		t.ticker.Stop()
		close(t.closeTicker)
		t.ticker = nil
	}
}

func (t *TopBar) checkTimer(ctx app.Context) {
	if State.Table != nil && State.Table.Started && t.ticker == nil {
		t.updateTime()
		t.closeTicker = make(chan struct{})
		t.ticker = time.NewTicker(1 * time.Second)
		go func() {
			for {
				select {
				case <-t.ticker.C:
					ctx.Dispatch(func(ctx app.Context) {
						t.updateTime()
					})
				case <-t.closeTicker:
					return
				}
			}
		}()
	} else if (State.Table == nil || !State.Table.Started) && t.ticker != nil {
		t.ticker.Stop()
		close(t.closeTicker)
		t.ticker = nil
		t.timeDisplay = ""
	}
}

func (t *TopBar) updateTime() {
	if State.Table != nil && !State.Table.StartTime.IsZero() {
		duration := time.Since(State.Table.StartTime)
		minutes := int(duration.Minutes())
		seconds := int(duration.Seconds()) % 60
		t.timeDisplay = fmt.Sprintf("%02d:%02d", minutes, seconds)
	} else {
		t.timeDisplay = "00:00"
	}
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
		app.Ul().Body(
			app.Li().Style("font-weight", "bold").Style("font-size", "1.2rem").Text(t.timeDisplay),
		),
		app.Ul().Body(actions...),
	)
}
