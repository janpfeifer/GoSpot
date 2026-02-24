package lobby

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/url"
	"time"

	"github.com/janpfeifer/GoSpot/internal/game"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"k8s.io/klog/v2"
)

// Login is the component for user login
type Login struct {
	app.Compo
	Name         string
	SymbolID     int
	ReturnURL    string
	ErrorMessage string
	ShowSymbols  bool
}

func (l *Login) OnMount(ctx app.Context) {
	// Default random symbol only if not already set
	if l.SymbolID == 0 {
		rand.Seed(time.Now().UnixNano())
		l.SymbolID = rand.Intn(57) // 0 to 56
	}

	// Parse URL to find the return path, if any
	u, err := url.Parse(app.Window().URL().String())
	if err == nil {
		q := u.Query()
		if returnURL := q.Get("return"); returnURL != "" {
			l.ReturnURL = returnURL
		}
	}

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

func (l *Login) redirect(ctx app.Context) {
	if l.ReturnURL != "" {
		ctx.Navigate(l.ReturnURL)
	} else {
		ctx.Navigate("/")
	}
}

func (l *Login) onNameChange(ctx app.Context, e app.Event) {
	l.Name = ctx.JSSrc().Get("value").String()
}

func (l *Login) toggleSymbols(ctx app.Context, e app.Event) {
	l.ShowSymbols = !l.ShowSymbols
	klog.Infof("toggleSymbols: ShowSymbols is now %v", l.ShowSymbols)
	ctx.Update()
}

func (l *Login) selectSymbol(ctx app.Context, e app.Event) {
	// Get the value from the radio input or data-id from the image
	klog.Infof("selectSymbol: %v", ctx.JSSrc())
	idStr := ctx.JSSrc().Get("value").String()
	if idStr == "" || idStr == "undefined" {
		idStr = ctx.JSSrc().Get("dataset").Get("id").String()
	}

	var id int
	if _, err := fmt.Sscanf(idStr, "%d", &id); err == nil {
		l.SymbolID = id
		l.ShowSymbols = false
		ctx.Update()
	}
}

func (l *Login) onLogin(ctx app.Context, e app.Event) {
	e.PreventDefault()
	if l.Name == "" {
		l.ErrorMessage = "Name cannot be empty."
		return
	}

	player := game.Player{
		ID:     fmt.Sprintf("%d", time.Now().UnixNano()),
		Name:   l.Name,
		Symbol: l.SymbolID,
	}

	klog.Infof("Login registered for player: %s (ID: %s, Symbol: %d)", player.Name, player.ID, player.Symbol)

	// Save to state
	State.Player = &player

	// Save to cookie
	playerBytes, _ := json.Marshal(player)
	setCookie("gospot_player", string(playerBytes), 30) // 30 days

	l.redirect(ctx)
}

func (l *Login) Render() app.UI {
	var symbols []app.UI
	for i := range 56 {
		imgSrc := fmt.Sprintf("/web/images/symbol_%02d.png", i)
		style := "width: 48px; height: 48px; cursor: pointer; border-radius: 50%; padding: 4px;"
		if l.SymbolID == i {
			style += " background-color: var(--primary); border: 2px solid var(--primary-hover);"
		} else {
			style += " border: 2px solid transparent;"
		}

		symbols = append(symbols, app.Label().Body(
			app.Input().
				Type("radio").
				Name("symbol").
				Value(fmt.Sprintf("%d", i)).
				Checked(l.SymbolID == i).
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

	selectedSymbolImg := fmt.Sprintf("/web/images/symbol_%02d.png", l.SymbolID)

	var symbolSelection app.UI
	klog.Infof("Render: ShowSymbols=%v", l.ShowSymbols)
	if l.ShowSymbols {
		symbolSelection = app.Div().Body(
			app.Label().Text("Choose your symbol"),
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
			app.Form().Method("POST").OnSubmit(l.onLogin).Body(
				app.Div().Style("display", "flex").Style("align-items", "center").Style("gap", "1rem").Style("margin-bottom", "1rem").Body(
					app.Img().
						Src(selectedSymbolImg).
						Style("width", "64px").
						Style("height", "64px").
						Style("border-radius", "50%").
						Style("border", "2px solid var(--primary)").
						Style("cursor", "pointer").
						OnClick(l.toggleSymbols),
					app.Input().
						Type("text").
						ID("name").
						Name("name").
						Placeholder("Enter your player name").
						Required(true).
						Value(l.Name).
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
