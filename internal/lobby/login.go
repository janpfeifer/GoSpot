package lobby

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/url"
	"time"

	"github.com/janpfeifer/GoSpot/internal/game"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

// Login is the component for user login
type Login struct {
	app.Compo
	Name         string
	SymbolID     int
	ReturnURL    string
	ErrorMessage string
}

func (l *Login) OnMount(ctx app.Context) {
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
			l.redirect()
			return
		}
	}

	// Default random symbol
	rand.Seed(time.Now().UnixNano())
	l.SymbolID = rand.Intn(57) + 1 // 1 to 57
}

func (l *Login) redirect() {
	if l.ReturnURL != "" {
		app.Window().Get("location").Set("href", l.ReturnURL)
	} else {
		app.Window().Get("location").Set("href", "/")
	}
}

func (l *Login) onNameChange(ctx app.Context, e app.Event) {
	l.Name = ctx.JSSrc().Get("value").String()
}

func (l *Login) selectSymbol(ctx app.Context, e app.Event) {
	// Let's get the id from dataset or value
	idStr := ctx.JSSrc().Get("dataset").Get("id").String()
	var id int
	if _, err := fmt.Sscanf(idStr, "%d", &id); err == nil {
		l.SymbolID = id
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

	// Save to state
	State.Player = &player

	// Save to cookie
	playerBytes, _ := json.Marshal(player)
	setCookie("gospot_player", string(playerBytes), 30) // 30 days

	l.redirect()
}

func (l *Login) Render() app.UI {
	var symbols []app.UI
	for i := range 57 {
		imgSrc := fmt.Sprintf("/web/images/symbol_%02d.png", i)
		style := "width: 48px; height: 48px; cursor: pointer; border-radius: 50%; padding: 4px;"
		if l.SymbolID == i {
			style += " background-color: var(--primary); border: 2px solid var(--primary-hover);"
		} else {
			style += " border: 2px solid transparent;"
		}

		symbols = append(symbols, app.Img().
			Src(imgSrc).
			Style("cssText", style). // go-app allows setting cssText directly to apply inline strings
			DataSet("id", fmt.Sprintf("%d", i)).
			OnClick(l.selectSymbol))
	}

	var errorUI app.UI = app.Text("")
	if l.ErrorMessage != "" {
		errorUI = app.Div().Style("color", "red").Style("margin-bottom", "1rem").Text(l.ErrorMessage)
	}

	return app.Main().Class("container").Body(
		app.Article().Body(
			app.Header().Body(
				app.H2().Text("GoSpot Login"),
			),
			errorUI,
			app.Form().OnSubmit(l.onLogin).Body(
				app.Label().For("name").Text("Player Name"),
				app.Input().
					Type("text").
					ID("name").
					Name("name").
					Placeholder("Enter your name").
					Required(true).
					Value(l.Name).
					AutoComplete(false).
					OnInput(l.onNameChange),
				app.Label().Text("Choose your symbol"),
				app.Div().Style("display", "flex").Style("flex-wrap", "wrap").Style("gap", "8px").Style("margin-bottom", "1rem").Body(
					symbols...,
				),
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
