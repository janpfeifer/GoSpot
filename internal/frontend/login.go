package frontend

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/janpfeifer/GoSpot/internal/game"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"k8s.io/klog/v2"
)

// Login is the component for user login
type Login struct {
	app.Compo
	ReturnURL    string
	ErrorMessage string
}

func (l *Login) OnMount(ctx app.Context) {
	klog.V(1).Infof("Login: OnMount called")
	l.parseReturnURL()

	// Read cookie to see if already logged in
	playerStr := getCookie("gospot_player")
	if playerStr != "" {
		var p game.Player
		if err := json.Unmarshal([]byte(playerStr), &p); err == nil {
			State.Player = &p
			l.redirect(ctx)
			return
		}
	}
}

func (l *Login) OnNav(ctx app.Context) {
	klog.V(1).Infof("Login: OnNav called")
	l.parseReturnURL()
}

func (l *Login) parseReturnURL() {
	// Parse URL to find the return path, if any
	u := app.Window().URL()
	q := u.Query()
	l.ReturnURL = q.Get("return")
	klog.V(1).Infof("Login: parseReturnURL URL=%s, ReturnURL=%s", u.String(), l.ReturnURL)
}

func (l *Login) redirect(ctx app.Context) {
	if l.ReturnURL != "" {
		ctx.Navigate(l.ReturnURL)
	} else {
		ctx.Navigate("/")
	}
}

func (l *Login) onNameChange(ctx app.Context, e app.Event) {
	State.PendingName = ctx.JSSrc().Get("value").String()
}

func (l *Login) toggleSymbols(ctx app.Context, e app.Event) {
	State.ShowSymbols = !State.ShowSymbols
	klog.V(1).Infof("toggleSymbols: ShowSymbols is now %v", State.ShowSymbols)
	ctx.Update()
}

func (l *Login) selectSymbol(ctx app.Context, e app.Event) {
	// Get the value from the radio input or data-id from the image
	idStr := ctx.JSSrc().Get("value").String()
	if idStr == "" || idStr == "undefined" {
		idStr = ctx.JSSrc().Get("dataset").Get("id").String()
	}

	var id int
	if _, err := fmt.Sscanf(idStr, "%d", &id); err == nil {
		State.SymbolID = id
		State.ShowSymbols = false
		ctx.Update()
	}
}

func (l *Login) onLogin(ctx app.Context, e app.Event) {
	e.PreventDefault()
	if State.PendingName == "" {
		l.ErrorMessage = "Name cannot be empty."
		return
	}

	player := game.Player{
		ID:     fmt.Sprintf("%d", time.Now().UnixNano()),
		Name:   State.PendingName,
		Symbol: State.SymbolID,
	}

	klog.V(1).Infof("Login registered for player: %s (ID: %s, Symbol: %d)", player.Name, player.ID, player.Symbol)

	// Save to state
	State.Player = &player

	// Save to cookie
	playerBytes, _ := json.Marshal(player)
	setCookie("gospot_player", string(playerBytes), 30) // 30 days

	State.SyncMusic()
	l.redirect(ctx)
}

func (l *Login) Render() app.UI {
	var symbols []app.UI
	for i := 0; i < 57; i++ {
		imgSrc := fmt.Sprintf("/web/images/symbol_%02d.png", i)
		style := "width: 48px; height: 48px; cursor: pointer; border-radius: 50%; padding: 4px;"
		if State.SymbolID == i {
			style += " background-color: var(--pico-primary); border: 2px solid var(--pico-primary-hover);"
		} else {
			style += " border: 2px solid transparent;"
		}

		symbols = append(symbols, app.Label().Body(
			app.Input().
				Type("radio").
				Name("symbol").
				Value(fmt.Sprintf("%d", i)).
				Checked(State.SymbolID == i).
				Style("display", "none").
				OnChange(l.selectSymbol),
			app.Img().
				Src(imgSrc).
				Style("cssText", style).
				DataSet("id", fmt.Sprintf("%d", i)).
				OnClick(l.selectSymbol),
		))
	}

	var errorUI app.UI = app.Text("")
	if l.ErrorMessage != "" {
		errorUI = app.Div().Style("color", "red").Style("margin-bottom", "1rem").Text(l.ErrorMessage)
	}

	selectedSymbolImg := fmt.Sprintf("/web/images/symbol_%02d.png", State.SymbolID)

	var symbolSelection app.UI
	klog.V(1).Infof("Render: ShowSymbols=%v", State.ShowSymbols)
	if State.ShowSymbols {
		symbolSelection = app.Div().Body(
			app.Label().Text("Choose your player symbol:").
				Attr("data-tooltip", "You discard extra cards if you match your own symbol during the game"),
			app.Div().Style("display", "flex").Style("flex-wrap", "wrap").Style("gap", "8px").Style("margin-bottom", "1rem").Body(
				symbols...,
			),
		)
	}

	return app.Main().Class("container").Body(
		app.Article().Body(
			app.Header().Body(
				app.Div().Style("text-align", "center").Body(
					app.Img().
						Src("/web/images/banner.png").
						Style("max-width", "100%").
						Style("max-height", "20vh").
						Style("border-radius", "12px"),
				),
			),
			errorUI,
			app.Form().OnSubmit(l.onLogin).Body(
				app.Div().Style("display", "flex").Style("align-items", "center").Style("gap", "1rem").Style("margin-bottom", "1rem").Body(
					app.Div().
						Style("position", "relative").
						Style("cursor", "pointer").
						OnClick(l.toggleSymbols).
						Body(
							app.Img().
								Src(selectedSymbolImg).
								Style("width", "64px").
								Style("height", "64px").
								Style("border-radius", "50%").
								Style("border", "2px solid var(--pico-primary)"),
							// Affordance triangle (bottom-left)
							app.Div().
								Style("position", "absolute").
								Style("bottom", "0").
								Style("left", "0").
								Style("width", "8px").
								Style("height", "8px").
								Style("background-color", "var(--pico-primary)").
								Style("border", "1px solid var(--pico-primary)").
								Style("clip-path", "polygon(0 0, 0 100%, 100% 100%)").
								Style("z-index", "10"),
						),
					app.Input().
						Type("text").
						ID("name").
						Name("name").
						Placeholder("Enter your player name").
						Required(true).
						Value(State.PendingName).
						AutoComplete(false).
						OnInput(l.onNameChange).
						Style("margin-bottom", "0"), // Remove pico.css default bottom margin
				),
				symbolSelection,
				app.Button().Type("submit").Text("Play"),
			),
		),
	)
}

func getCookie(name string) string {
	document := app.Window().Get("document")
	if !document.Truthy() {
		return ""
	}
	cookie := document.Get("cookie").String()
	// Parse the cookie string
	// A simple manual parser for exactly the key
	nameLen := len(name)
	for i := 0; i < len(cookie); i++ {
		if i+nameLen <= len(cookie) && cookie[i:i+nameLen] == name {
			// Found name, check if next char is '='
			if i+nameLen < len(cookie) && cookie[i+nameLen] == '=' {
				start := i + nameLen + 1
				end := start
				for end < len(cookie) && cookie[end] != ';' {
					end++
				}
				v, _ := url.QueryUnescape(cookie[start:end])
				return v
			}
		}
	}
	return ""
}

func setCookie(name, value string, days int) {
	document := app.Window().Get("document")
	if !document.Truthy() {
		return
	}
	expires := ""
	if days > 0 {
		t := time.Now().AddDate(0, 0, days)
		expires = "; expires=" + t.UTC().Format(time.RFC1123)
	}
	encodedValue := url.QueryEscape(value)
	document.Set("cookie", name+"="+encodedValue+expires+"; path=/")
}
