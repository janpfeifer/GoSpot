package lobby

import (
	"fmt"
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
			klog.Infof("Table component: State updated. Player count: %d", len(t.State.Players))
		})
	}
	State.Listeners["table"] = t.onUpdate
}

func (t *Table) OnDismount() {
	klog.Infof("Table component: OnDismount called")
	delete(State.Listeners, "table")
}

func (t *Table) OnNav(ctx app.Context) {
	klog.Infof("Table component: OnNav called")
	t.State = State.Table
	// Check auth
	if State.Player == nil || State.Player.ID == "" {
		app.Window().Get("location").Set("href", "/?return="+app.Window().URL().Path)
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

func (t *Table) Render() app.UI {
	if State.Player == nil || State.Player.ID == "" {
		return &Login{} // Or basic loading
	}

	if t.Error != "" {
		return app.Main().Class("container").Body(
			app.Article().Body(
				app.H2().Text("Error"),
				app.P().Style("color", "red").Text(t.Error),
				app.A().Href("/").Text("Return to Home"),
			),
		)
	}

	var content app.UI
	if t.State == nil {
		content = app.Div().Aria("busy", "true").Text("Connecting to table...")
	} else if t.State.Started {
		// Render Game Canvas here later
		content = app.Div().Text("Game Started! Ebiten Engine will take over here.")
	} else {
		// Render Lobby
		var playersList []app.UI
		for _, p := range t.State.Players {
			playersList = append(playersList, app.Li().Body(
				app.Img().
					Src(fmt.Sprintf("/web/images/symbol_%02d.png", p.Symbol)).
					Style("width", "32px").Style("height", "32px").Style("vertical-align", "middle").Style("margin-right", "8px"),
				app.Span().Text(p.Name),
			))
		}

		canStart := len(t.State.Players) >= 2

		var waitingMsg app.UI = app.Text("")
		if !canStart {
			waitingMsg = app.P().Class("ins").Text("Waiting for at least 2 players to start...")
		}

		content = app.Div().Body(
			app.Div().Class("grid").Body(
				app.Div().Body(
					app.H3().Text(fmt.Sprintf("Table: %s", t.TableID)),
					app.P().Text("Share this URL to invite friends:"),
					app.Div().Class("grid").Body(
						app.Input().Type("text").ReadOnly(true).Value(app.Window().URL().String()),
						app.Button().Class("secondary").Text("Copy URL").OnClick(t.onCopyURL),
					),
				),
			),
			app.Article().Body(
				app.Header().Text(fmt.Sprintf("Players (%d)", len(t.State.Players))),
				app.Ul().Body(playersList...),
				app.Footer().Body(
					app.Button().Text("Start Game").Disabled(!canStart).OnClick(t.onStart),
					waitingMsg,
				),
			),
		)
	}

	return app.Main().Class("container").Body(
		app.Nav().Body(
			app.Ul().Body(
				app.Li().Body(app.Strong().Text("GoSpot").Style("cursor", "pointer").OnClick(func(ctx app.Context, e app.Event) {
					app.Window().Get("location").Set("href", "/")
				})),
			),
			app.Ul().Body(
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
