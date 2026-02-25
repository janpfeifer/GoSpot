package frontend

import (
	"fmt"
	"math"
	"net/url"
	"strings"

	"github.com/janpfeifer/GoSpot/internal/game"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"k8s.io/klog/v2"
)

// Game represents the running game room
type Game struct {
	app.Compo
	GameID string
	State  *game.Table
	Error  string

	onUpdate func()
}

func (g *Game) OnMount(ctx app.Context) {
	klog.Infof("Game component: OnMount called")
	g.State = State.Table
	g.onUpdate = func() {
		klog.Infof("Game component: Notify received")
		ctx.Dispatch(func(ctx app.Context) {
			g.State = State.Table
			g.Error = State.Error
			if g.State != nil {
				klog.Infof("Game component: State updated. Player count: %d", len(g.State.Players))
				if !g.State.Started {
					ctx.Navigate("/table/" + g.GameID)
				}
			} else if g.Error != "" {
				klog.Infof("Game component: Error received. Error: %s", g.Error)
			}
		})
	}
	State.Listeners["game"] = g.onUpdate
	State.SyncMusic()
}

func (g *Game) OnDismount() {
	klog.Infof("Game component: OnDismount called")
	delete(State.Listeners, "game")
}

func (g *Game) OnNav(ctx app.Context) {
	klog.Infof("Game component: OnNav called")
	g.State = State.Table
	State.SyncMusic()
	// Check auth
	if State.Player == nil || State.Player.ID == "" {
		ctx.Navigate("/?return=" + url.QueryEscape(app.Window().URL().Path))
		return
	}

	path := app.Window().URL().Path
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	klog.Infof("Game component: Navigated to %s, parts: %v", path, parts)
	if len(parts) >= 2 && parts[0] == "game" {
		g.GameID = parts[1]
	}

	if g.GameID == "" {
		g.Error = "No Game ID provided"
		klog.Errorf("Game component: Error: %s", g.Error)
		return
	}

	klog.Infof("Game component: Connecting to game ID: %s", g.GameID)
	if State.Conn == nil || State.Table == nil || State.Table.ID != g.GameID {
		// Connect to WS
		if err := State.ConnectWS(g.GameID); err != nil {
			g.Error = fmt.Sprintf("Failed to connect to game: %v", err)
			klog.Errorf("Game component: Error connecting: %v", err)
		}
	}
}

func (g *Game) onToggleSound(ctx app.Context, e app.Event) {
	e.PreventDefault()
	State.ToggleSound()
}

func (g *Game) renderCard(symbols []int, size int) app.UI {
	if len(symbols) == 0 {
		return app.Div().Class("card-svg").Style("width", fmt.Sprintf("%dpx", size)).Style("height", fmt.Sprintf("%dpx", size)).Body(
			app.P().Style("text-align", "center").Text("No card"),
		)
	}

	center := float64(size) / 2
	radius := float64(size) / 2
	// Symbols further out from center
	innerRadius := radius * 0.85
	symbolSize := float64(size) / 5.0

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<svg width="%d" height="%d" viewBox="0 0 %d %d" class="card-svg">`, size, size, size, size))

	// Card background image
	sb.WriteString(fmt.Sprintf(`<image href="/web/images/card_background.png" x="0" y="0" width="%d" height="%d" />`, size, size))

	for i, s := range symbols {
		angle := float64(i) * 2 * math.Pi / float64(len(symbols))
		// Position symbols along the inner radius - moved slightly closer to center
		x := center + innerRadius*0.70*math.Cos(angle) - symbolSize/2
		y := center + innerRadius*0.70*math.Sin(angle) - symbolSize/2

		sb.WriteString(fmt.Sprintf(
			`<image href="/web/images/symbol_%02d.png" x="%f" y="%f" width="%f" height="%f" />`,
			s, x, y, symbolSize, symbolSize,
		))
	}
	sb.WriteString(`</svg>`)

	return app.Raw(sb.String())
}

func (g *Game) renderPlayerList(players []*game.Player) app.UI {
	var listItems []app.UI
	for _, p := range players {
		listItems = append(listItems, app.Li().Class("player-item-game").Body(
			app.Img().
				Src(fmt.Sprintf("/web/images/symbol_%02d.png", p.Symbol)).
				Style("width", "32px").Style("height", "32px"),
			app.Span().Text(fmt.Sprintf("%s: %d cards left", p.Name, p.Score)),
		))
	}
	return app.Ul().Class("player-list-game").Body(listItems...)
}

func (g *Game) Render() app.UI {
	if State.Player == nil || State.Player.ID == "" {
		return app.Main().Class("container").Body(
			app.Div().Aria("busy", "true").Text("Redirecting to login..."),
		)
	}

	soundIcon := "🔊"
	if !State.SoundEnabled {
		soundIcon = "🔇"
	}

	if g.Error != "" {
		return app.Main().Class("container").Body(
			app.Article().Body(
				app.H2().Text("Game Error"),
				app.P().Style("color", "red").Text(g.Error),
				app.A().Href("#").OnClick(func(ctx app.Context, e app.Event) {
					State.Error = ""
					ctx.Navigate("/")
				}).Text("Return to Home"),
			),
		)
	}

	var content app.UI
	if g.State == nil {
		content = app.Div().Aria("busy", "true").Text("Connecting to game...")
	} else if g.State.Started {
		// Render Game Page using SVG and HTML
		otherPlayers := make([]*game.Player, 0)
		for _, p := range g.State.Players {
			if p.ID != State.Player.ID {
				otherPlayers = append(otherPlayers, p)
			}
		}

		topRightPlayers := make([]*game.Player, 0)
		bottomLeftPlayers := make([]*game.Player, 0)
		for i, p := range otherPlayers {
			if i%2 == 0 {
				topRightPlayers = append(topRightPlayers, p)
			} else {
				bottomLeftPlayers = append(bottomLeftPlayers, p)
			}
		}

		content = app.Div().Class("game-grid").Body(
			// First Column (70/30)
			app.Div().Class("game-column").Body(
				// Top: Current Player's info and Card (70%)
				app.Div().Class("column-70").Class("card-container").Body(
					app.Div().Style("display", "flex").Style("align-items", "center").Style("gap", "0.5rem").Style("margin-bottom", "0.5rem").Body(
						app.Img().
							Src(fmt.Sprintf("/web/images/symbol_%02d.png", State.Player.Symbol)).
							Style("width", "32px").Style("height", "32px"),
						app.Strong().Text(fmt.Sprintf("%s (%d cards left)", State.Player.Name, State.Player.Score)),
					),
					g.renderCard(State.TopCard, 520),
				),
				// Bottom: Other players list - second half (30%)
				app.Div().Class("column-30").Body(
					g.renderPlayerList(bottomLeftPlayers),
				),
			),
			// Second Column (30/70)
			app.Div().Class("game-column").Body(
				// Top: Other players list - first half (30%)
				app.Div().Class("column-30").Body(
					g.renderPlayerList(topRightPlayers),
				),
				// Bottom: Target Card (70%)
				app.Div().Class("column-70").Class("card-container").Body(
					g.renderCard(State.TargetCard, 520),
				),
			),
		)
	} else {
		content = app.Div().Aria("busy", "true").Text("Connecting to game...")
	}

	return app.Main().Class("container").Body(
		app.Nav().Body(
			app.Ul().Body(
				app.Li().Body(
					app.Img().
						Src("/web/images/banner.png").
						Style("height", "2rem").
						Style("vertical-align", "middle").
						Style("cursor", "pointer").
						Style("border-radius", "8px").
						OnClick(func(ctx app.Context, e app.Event) {
							ctx.Navigate("/")
						}),
				),
			),
			app.Ul().Body(
				app.Li().Body(
					app.A().
						Href("#").
						OnClick(g.onToggleSound).
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
			),
		),
		content,
	)
}
