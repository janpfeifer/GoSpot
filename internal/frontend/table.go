package frontend

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/janpfeifer/GoSpot/internal/game"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"k8s.io/klog/v2"
)

import (
	"math/rand"
)

// Table represents the lobby for a specific game room
type Table struct {
	app.Compo
	TableID       string
	State         *game.Table
	Error         string
	showSoloModal bool
	randomSymbol  int

	onUpdate func()
}

func (t *Table) OnAppUpdate(ctx app.Context) {
	klog.Infof("Table component: App update available, reloading...")
	ctx.Reload()
}

func (t *Table) OnMount(ctx app.Context) {
	klog.Infof("Table component: OnMount called")
	t.State = State.Table
	t.showSoloModal = false
	t.randomSymbol = rand.Intn(57) // 0 to 56
	t.onUpdate = func() {
		klog.Infof("Table component: Notify received")
		ctx.Dispatch(func(ctx app.Context) {
			t.State = State.Table
			t.Error = State.Error
			if t.State != nil {
				klog.Infof("Table component: State updated. Player count: %d", len(t.State.Players))
				if t.State.Started {
					ctx.Navigate("/game/" + t.TableID)
				}
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
	if State.Conn == nil || State.Table == nil || State.Table.ID != t.TableID {
		// Connect to WS
		if err := State.ConnectWS(t.TableID); err != nil {
			t.Error = fmt.Sprintf("Failed to connect to table: %v", err)
			klog.Errorf("Table component: Error connecting: %v", err)
		}
	}
}

func (t *Table) onCopyURL(ctx app.Context, e app.Event) {
	url := app.Window().URL().String()
	app.Window().Get("navigator").Get("clipboard").Call("writeText", url)
	app.Window().Call("alert", "URL copied to clipboard!")
}

func (t *Table) onStart(ctx app.Context, e app.Event) {
	if t.State != nil && len(t.State.Players) == 1 {
		t.showSoloModal = true
	} else {
		State.SendStart()
	}
}

func (t *Table) onReady(ctx app.Context, e app.Event) {
	t.showSoloModal = false
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

func (t *Table) Render() app.UI {
	if State.Player == nil || State.Player.ID == "" {
		return app.Main().Class("container").Body(
			app.Div().Aria("busy", "true").Text("Redirecting to login..."),
		)
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
		content = app.Div().Aria("busy", "true").Text("Redirecting to game...")
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
		canStart := len(t.State.Players) >= 1 // allow starting with 1 player

		var footer app.UI
		if isCreator {
			var waitingMsg app.UI = app.Text("")
			if len(t.State.Players) == 1 {
				waitingMsg = app.P().Class("ins").Style("text-align", "center").Text("Waiting for more players... or play solo!")
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


		var soloModal app.UI
		if t.showSoloModal {
			soloModal = app.Dialog().Open(true).Body(
				app.Article().Body(
					app.Header().Text("Solo Game"),
					app.Div().Style("display", "flex").Style("align-items", "center").Style("gap", "1rem").Body(
						app.Img().
							Src(fmt.Sprintf("/web/images/symbol_%02d.png", t.randomSymbol)).
							Style("width", "8em").
							Style("height", "8em"),
						app.P().Text("You are playing against the clock. Try to discard 10 cards as fast as you can!"),
					),
					app.Footer().Body(
						app.Button().Text("Ready").OnClick(t.onReady),
					),
				),
			)
		} else {
			soloModal = app.Text("")
		}

		content = app.Div().Body(
			soloModal,
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
		&TopBar{},
		content,
	)
}
