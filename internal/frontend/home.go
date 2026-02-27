package frontend

import (
	"fmt"
	"math/rand"

	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"k8s.io/klog/v2"
)

// Home is the landing page component
type Home struct {
	app.Compo
	TableName string
	login     *Login
}

func (h *Home) OnMount(ctx app.Context) {
	klog.V(1).Infof("Home: OnMount called")
	h.login = &Login{}
	State.Listeners["home"] = func() {
		ctx.Dispatch(func(ctx app.Context) {})
	}
	State.SyncMusic()
}

func (h *Home) OnDismount() {
	delete(State.Listeners, "home")
}

func (h *Home) OnNav(ctx app.Context) {
	klog.V(1).Infof("Home: OnNav called, Path=%s", app.Window().URL().Path)
	State.SyncMusic()
	if h.login != nil {
		h.login.OnNav(ctx)
	}
}

func (h *Home) onTableNameChange(ctx app.Context, e app.Event) {
	h.TableName = ctx.JSSrc().Get("value").String()
}

func (h *Home) onCreateTable(ctx app.Context, e app.Event) {
	e.PreventDefault()
	if h.TableName == "" {
		// generate random suffix or table name
		h.TableName = fmt.Sprintf("Table-%d", rand.Intn(10000))
	}

	ctx.Navigate("/table/" + h.TableName)
}

func (h *Home) onLogout(ctx app.Context, e app.Event) {
	e.PreventDefault()
	State.Player = nil
	// Clear cookie in JS as well
	app.Window().Get("document").Set("cookie", "gospot_player=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;")
	State.SyncMusic()
	ctx.Navigate("/")
}

func (h *Home) onToggleSound(ctx app.Context, e app.Event) {
	e.PreventDefault()
	State.ToggleSound()
}

func (h *Home) OnAppUpdate(ctx app.Context) {
	klog.Infof("Home component: App update available, reloading...")
	ctx.Reload()
}

func (h *Home) Render() app.UI {
	if State.Player == nil || State.Player.ID == "" {
		// Render login instead
		if h.login == nil {
			h.login = &Login{}
		}
		return h.login
	}

	return app.Main().Class("container").Body(
		&TopBar{ShowLogout: true},
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
