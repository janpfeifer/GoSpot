package lobby

import (
	"fmt"
	"math"
	"net/url"
	"strings"

	"github.com/janpfeifer/GoSpot/internal/game"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"k8s.io/klog/v2"
)

// Table represents the lobby for a specific game room
type Table struct {
	app.Compo
	TableID string
	State   *game.Table
	Error   string

	onUpdate func()
}

func (t *Table) OnMount(ctx app.Context) {
	klog.Infof("Table component: OnMount called")
	t.State = State.Table
	t.onUpdate = func() {
		klog.Infof("Table component: Notify received")
		ctx.Dispatch(func(ctx app.Context) {
			t.State = State.Table
			t.Error = State.Error
			if t.State != nil {
				klog.Infof("Table component: State updated. Player count: %d", len(t.State.Players))
			} else if t.Error != "" {
				klog.Infof("Table component: Error received. Error: %s", t.Error)
			}
		})
	}
	State.Listeners["table"] = t.onUpdate
	State.SyncMusic()
}

func (t *Table) OnDismount() {
	klog.Infof("Table component: OnDismount called")
	delete(State.Listeners, "table")
}

func (t *Table) OnNav(ctx app.Context) {
	klog.Infof("Table component: OnNav called")
	t.State = State.Table
	State.SyncMusic()
	// Check auth
	if State.Player == nil || State.Player.ID == "" {
		ctx.Navigate("/?return=" + url.QueryEscape(app.Window().URL().Path))
		return
	}

	path := app.Window().URL().Path
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	klog.Infof("Table component: Navigated to %s, parts: %v", path, parts)
	if len(parts) >= 2 && parts[0] == "table" {
		t.TableID = parts[1]
	}

	if t.TableID == "" {
		t.Error = "No Table ID provided"
		klog.Errorf("Table component: Error: %s", t.Error)
		return
	}

	klog.Infof("Table component: Connecting to table ID: %s", t.TableID)
	// Connect to WS
	if err := State.ConnectWS(t.TableID); err != nil {
		t.Error = fmt.Sprintf("Failed to connect to table: %v", err)
		klog.Errorf("Table component: Error connecting: %v", err)
	}
}

func (t *Table) onCopyURL(ctx app.Context, e app.Event) {
	url := app.Window().URL().String()
	app.Window().Get("navigator").Get("clipboard").Call("writeText", url)
	app.Window().Call("alert", "URL copied to clipboard!")
}

func (t *Table) onStart(ctx app.Context, e app.Event) {
	State.SendStart()
}

func (t *Table) onCancel(ctx app.Context, e app.Event) {
	State.SendCancel()
	State.SyncMusic()
	ctx.Navigate("/")
}

func (t *Table) onToggleSound(ctx app.Context, e app.Event) {
	e.PreventDefault()
	State.ToggleSound()
}

func (t *Table) renderCard(symbols []int, size int) app.UI {
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

func (t *Table) renderPlayerList(players []*game.Player) app.UI {
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

func (t *Table) Render() app.UI {
	if State.Player == nil || State.Player.ID == "" {
		return app.Main().Class("container").Body(
			app.Div().Aria("busy", "true").Text("Redirecting to login..."),
		)
	}

	soundIcon := "🔊"
	if !State.SoundEnabled {
		soundIcon = "🔇"
	}

	if t.Error != "" {
		return app.Main().Class("container").Body(
			app.Article().Body(
				app.H2().Text("Table Closed"),
				app.P().Style("color", "red").Text(t.Error),
				app.A().Href("#").OnClick(func(ctx app.Context, e app.Event) {
					State.Error = ""
					ctx.Navigate("/")
				}).Text("Return to Home"),
			),
		)
	}

	var content app.UI
	if t.State == nil {
		content = app.Div().Aria("busy", "true").Text("Connecting to table...")
	} else if t.State.Started {
		// Render Game Page using SVG and HTML
		otherPlayers := make([]*game.Player, 0)
		for _, p := range t.State.Players {
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
			// Top-Left: Current Player's info and Card
			app.Div().Class("card-container").Style("grid-column", "1").Style("grid-row", "1").Body(
				app.Div().Style("display", "flex").Style("align-items", "center").Style("gap", "0.5rem").Style("margin-bottom", "0.5rem").Body(
					app.Img().
						Src(fmt.Sprintf("/web/images/symbol_%02d.png", State.Player.Symbol)).
						Style("width", "32px").Style("height", "32px"),
					app.Strong().Text(fmt.Sprintf("%s (%d cards left)", State.Player.Name, State.Player.Score)),
				),
				t.renderCard(State.TopCard, 400),
			),
			// Top-Right: Half of other players
			app.Div().Style("grid-column", "2").Style("grid-row", "1").Body(
				t.renderPlayerList(topRightPlayers),
			),
			// Bottom-Left: Other half of other players
			app.Div().Style("grid-column", "1").Style("grid-row", "2").Body(
				t.renderPlayerList(bottomLeftPlayers),
			),
			// Bottom-Right: Target Card
			app.Div().Class("card-container").Style("grid-column", "2").Style("grid-row", "2").Body(
				t.renderCard(State.TargetCard, 400),
			),
		)
	} else {
		// Render Lobby
		var playersList []app.UI
		for i, p := range t.State.Players {
			name := p.Name
			if i == 0 {
				name += " (Creator)"
			}
			playersList = append(playersList, app.Li().Body(
				app.Img().
					Src(fmt.Sprintf("/web/images/symbol_%02d.png", p.Symbol)).
					Style("width", "32px").Style("height", "32px").Style("vertical-align", "middle").Style("margin-right", "8px"),
				app.Span().Text(name),
			))
		}

		isCreator := len(t.State.Players) > 0 && t.State.Players[0].ID == State.Player.ID
		canStart := len(t.State.Players) >= 2

		var footer app.UI
		if isCreator {
			var waitingMsg app.UI = app.Text("")
			if !canStart {
				waitingMsg = app.P().Class("ins").Style("text-align", "center").Text("Waiting for at least 2 players to start...")
			}
			footer = app.Footer().Body(
				waitingMsg,
				app.Div().Style("display", "flex").Style("gap", "1rem").Style("justify-content", "center").Body(
					app.Button().
						Text("Start Game").
						Disabled(!canStart).
						OnClick(t.onStart).
						Style("flex", "1").
						Style("margin-bottom", "0"),
					app.Button().
						Class("outline contrast").
						Text("Cancel Table").
						OnClick(t.onCancel).
						Style("flex", "1").
						Style("margin-bottom", "0"),
				),
			)
		} else {
			footer = app.Footer().Body(
				app.P().Text("Waiting for the creator to start the game..."),
			)
		}

		content = app.Div().Body(
			app.Div().Class("grid").Body(
				app.Div().Body(
					app.H3().Text(fmt.Sprintf("Table: %s", t.TableID)),
					app.P().Text("Share this URL to invite friends:"),
					app.Div().Style("display", "flex").Style("gap", "0.5rem").Style("align-items", "center").Style("margin-bottom", "var(--pico-spacing)").Body(
						app.Input().
							Type("text").
							ReadOnly(true).
							Value(app.Window().URL().String()).
							Style("margin-bottom", "0").
							Style("flex", "1"),
						app.Button().
							Class("secondary").
							Text("Copy URL").
							OnClick(t.onCopyURL).
							Style("margin-bottom", "0").
							Style("width", "auto").
							Style("padding", "0.5rem 1rem"),
					),
				),
			),
			app.Article().Body(
				app.Header().Text(fmt.Sprintf("Players (%d)", len(t.State.Players))),
				app.Ul().Body(playersList...),
				footer,
			),
		)
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
			),
		),
		content,
	)
}
