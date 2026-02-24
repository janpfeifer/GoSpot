package lobby

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

// Home is the landing page component
type Home struct {
	app.Compo
	TableName string
}

func (h *Home) OnNav(ctx app.Context) {
	if State.Player == nil || State.Player.ID == "" {
		// Not logged in, redirect to login via Login component logic
		// We'll actually render the login page in the Home route if not logged in
	}
}

func (h *Home) onTableNameChange(ctx app.Context, e app.Event) {
	h.TableName = ctx.JSSrc().Get("value").String()
}

func (h *Home) onCreateTable(ctx app.Context, e app.Event) {
	e.PreventDefault()
	if h.TableName == "" {
		// generate random suffix or table name
		rand.Seed(time.Now().UnixNano())
		h.TableName = fmt.Sprintf("Table-%d", rand.Intn(10000))
	}

	app.Window().Get("location").Set("href", "/table/"+h.TableName)
}

func (h *Home) Render() app.UI {
	if State.Player == nil || State.Player.ID == "" {
		// Render login instead
		return &Login{}
	}

	return app.Main().Class("container").Body(
		app.Nav().Body(
			app.Ul().Body(
				app.Li().Body(app.Strong().Text("GoSpot")),
			),
			app.Ul().Body(
				app.Li().Body(
					app.Span().Style("margin-right", "8px").Text(State.Player.Name),
					app.Img().
						Src(fmt.Sprintf("/web/images/symbol_%02d.png", State.Player.Symbol)).
						Style("width", "32px").Style("height", "32px").Style("vertical-align", "middle"),
				),
				app.Li().Body(app.A().Href("/logout").Text("Logout")),
			),
		),
		app.Article().Body(
			app.Header().Body(
				app.H2().Text("Create or Join a Table"),
			),
			app.P().Text("Enter a name for your a table to create it and invite friends."),
			app.Form().OnSubmit(h.onCreateTable).Body(
				app.Label().For("tableName").Text("Table Name"),
				app.Input().
					Type("text").
					ID("tableName").
					Name("tableName").
					Placeholder("e.g. My Awesome Table").
					Value(h.TableName).
					OnInput(h.onTableNameChange),
				app.Button().Type("submit").Text("Create Table"),
			),
		),
	)
}
