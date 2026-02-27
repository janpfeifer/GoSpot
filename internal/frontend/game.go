package frontend

import (
	"fmt"
	"math"
	"net/url"
	"slices"
	"strings"
	"time"

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

	// Click state
	actionPending bool
	clickedSymbol int
	matchedSymbol int
	glowRed       bool
	glowYellow    bool
	winnerShineID string
	lastTopCard   string
	lastRound     int

	onUpdate func()
}

func (g *Game) OnAppUpdate(ctx app.Context) {
	klog.Infof("Game component: App update available, not reloading not to interrupt the game...")
	//ctx.Reload()
}

func (g *Game) OnMount(ctx app.Context) {
	klog.Infof("Game component: OnMount called")
	g.State = State.Table
	g.clickedSymbol = -1
	g.matchedSymbol = -1
	g.lastTopCard = fmt.Sprintf("%v", State.TopCard)
	g.lastRound = State.Round
	g.onUpdate = func() {
		klog.Infof("Game component: Notify received")
		ctx.Dispatch(func(ctx app.Context) {
			g.State = State.Table
			g.Error = State.Error

			// Detect round change to unblock actionPending and play sounds
			if State.Round != g.lastRound {
				// Round changed! Check if we were waiting for a result
				if g.matchedSymbol != -1 {
					if State.WinnerID == State.Player.ID {
						if g.matchedSymbol == State.Player.Symbol {
							State.PlaySound("/web/sounds/bonus.mp3")
						} else {
							State.PlaySound("/web/sounds/matched.mp3")
						}
					} else {
						State.PlaySound("/web/sounds/lost-tie.mp3")
					}
				}

				// Shine the winner for 1 second
				if State.WinnerID != "" {
					g.winnerShineID = State.WinnerID
					klog.Infof("Game component: Winner shine set for player %s (current player: %s)", g.winnerShineID, State.Player.ID)
					time.AfterFunc(WinnerShineDuration, func() {
						ctx.Dispatch(func(ctx app.Context) {
							klog.Infof("Game component: Winner shine cleared")
							g.winnerShineID = ""
						})
					})
				}

				g.actionPending = false
				g.clickedSymbol = -1
				g.matchedSymbol = -1
				g.glowYellow = false
				g.lastRound = State.Round
				g.lastTopCard = fmt.Sprintf("%v", State.TopCard)
			} else if len(State.TopCard) > 0 {
				// Fallback: Detect top card change (e.g. initial game start or unusual state)
				topCardStr := fmt.Sprintf("%v", State.TopCard)
				if topCardStr != g.lastTopCard {
					g.actionPending = false
					g.clickedSymbol = -1
					g.matchedSymbol = -1
					g.glowYellow = false
					g.lastTopCard = topCardStr
				}
			}

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

	app.Window().Set("triggerSymbolClick", app.FuncOf(func(this app.Value, args []app.Value) any {
		if len(args) >= 1 {
			symbol := args[0].Int()
			ctx.Dispatch(func(ctx app.Context) {
				g.onSymbolClick(ctx, symbol)
			})
		}
		return nil
	}))

	State.SyncMusic()
}

func (g *Game) OnDismount() {
	klog.Infof("Game component: OnDismount called")
	delete(State.Listeners, "game")
	app.Window().Set("triggerSymbolClick", app.Undefined())
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

const PenaltyDuration = 2 * time.Second
const GlowDuration = 500 * time.Millisecond
const WinnerShineDuration = 2 * time.Second

func (g *Game) onSymbolClick(ctx app.Context, symbol int) {
	if g.actionPending {
		return
	}

	g.actionPending = true
	g.clickedSymbol = symbol
	matched := slices.Contains(State.TargetCard, symbol)

	if matched {
		g.matchedSymbol = symbol
		g.glowYellow = true
		time.AfterFunc(GlowDuration, func() {
			ctx.Dispatch(func(ctx app.Context) {
				g.glowYellow = false
			})
		})

	} else {
		// No match: set penalty time.
		State.PlaySound("/web/sounds/wrong.mp3")
		g.glowRed = true
		if State.Player != nil {
			State.Player.InPenalty = true
		}
		time.AfterFunc(PenaltyDuration, func() {
			ctx.Dispatch(func(ctx app.Context) {
				g.glowRed = false
				g.actionPending = false
				g.clickedSymbol = -1
				if State.Player != nil {
					State.Player.InPenalty = false
				}
			})
		})
	}

	State.SendClick(symbol)
}

func (g *Game) getGlowFilter(s int, isPlayerCard bool) string {
	if isPlayerCard {
		if s == g.clickedSymbol {
			if g.glowYellow {
				return "url(#glow-yellow-white)"
			}
			if g.glowRed {
				return "url(#glow-red)"
			}
		}
	} else {
		// Target card
		if s == g.matchedSymbol && g.glowYellow {
			return "url(#glow-yellow-white)"
		}
	}
	return "none"
}

func (g *Game) renderCard(symbols []int, size int, isClickable bool) app.UI {
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
	sb.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%[1]d" height="%[1]d" viewBox="0 0 %[1]d %[1]d" class="card-svg">`, size))

	sb.WriteString(`<defs>
		<filter id="glow-red" x="-50%" y="-50%" width="200%" height="200%">
			<feGaussianBlur stdDeviation="8" result="blur" />
			<feFlood flood-color="red" result="color" />
			<feComposite in="color" in2="blur" operator="in" result="glow" />
			<feComposite in="SourceGraphic" in2="glow" operator="over" />
		</filter>
		<filter id="glow-yellow-white" x="-50%" y="-50%" width="200%" height="200%">
			<feGaussianBlur stdDeviation="8" result="blur" />
			<feFlood flood-color="gold" result="color" />
			<feComposite in="color" in2="blur" operator="in" result="glow" />
			<feComposite in="SourceGraphic" in2="glow" operator="over" />
		</filter>
	</defs>`)

	if isClickable {
		sb.WriteString(`<style>
			.symbol-group { cursor: pointer; }
			.symbol-ring { opacity: 0; transition: opacity 0.2s ease-in-out; }
			.symbol-group:hover .symbol-ring { opacity: 1; }
			.symbol-group.disabled { cursor: not-allowed; }
			.symbol-group.disabled:hover .symbol-ring { opacity: 0; }
		</style>`)
	}

	// Card background image
	sb.WriteString(fmt.Sprintf(`<image href="/web/images/card_background.png" x="0" y="0" width="%d" height="%d" />`, size, size))

	for i, s := range symbols {
		angle := float64(i) * 2 * math.Pi / float64(len(symbols))
		// Position symbols along the inner radius - moved slightly closer to center
		x := center + innerRadius*0.70*math.Cos(angle) - symbolSize/2
		y := center + innerRadius*0.70*math.Sin(angle) - symbolSize/2

		if isClickable {
			disabledClass := ""
			onClickAttr := fmt.Sprintf(`onclick="triggerSymbolClick(%d)"`, s)
			if g.actionPending {
				disabledClass = " disabled"
				onClickAttr = ""
			}

			sb.WriteString(fmt.Sprintf(`<g class="symbol-group%s" %s>`, disabledClass, onClickAttr))

			// Add a blurred ring around the symbol to indicate it is clickable on hover
			cx := x + symbolSize/2
			cy := y + symbolSize/2
			ringRadius := symbolSize * 0.45 // smaller ring
			sb.WriteString(fmt.Sprintf(
				`<circle class="symbol-ring" cx="%f" cy="%f" r="%f" fill="none" stroke="var(--pico-primary-hover)" stroke-width="6" filter="blur(3px)" />`,
				cx, cy, ringRadius,
			))
		} else {
			sb.WriteString(`<g>`)
		}

		filterStyle := ""
		filter := g.getGlowFilter(s, isClickable)
		if filter != "none" {
			filterStyle = fmt.Sprintf(`style="filter: %s; transition: filter 0.1s ease-in-out;"`, filter)
		} else {
			filterStyle = `style="transition: filter 0.1s ease-in-out;"`
		}

		sb.WriteString(fmt.Sprintf(
			`<image href="/web/images/symbol_%02d.png" x="%f" y="%f" width="%f" height="%f" %s />`,
			s, x, y, symbolSize, symbolSize, filterStyle,
		))

		sb.WriteString(`</g>`)
	}

	sb.WriteString(`</svg>`)

	return app.Raw(sb.String())
}

// applyShineStyles applies golden glow inline styles to an element.
func applyShineStyles(elem app.HTMLLi) app.HTMLLi {
	return elem.
		Style("background-color", "rgba(255, 215, 0, 0.25)").
		Style("box-shadow", "inset 0 0 12px 4px rgba(255, 215, 0, 0.6), 0 0 15px 5px rgba(255, 215, 0, 0.4)")
}

func (g *Game) renderPlayerList(players []*game.Player) app.UI {
	var listItems []app.UI
	for _, p := range players {
		var text string
		if p.Score == 0 {
			text = fmt.Sprintf("%s: %s", p.Name, p.TimeTaken)
			if p.IsWinner {
				text = "👑 " + text
			}
		} else {
			text = fmt.Sprintf("%s: %d cards left ...", p.Name, p.Score)
		}
		li := app.Li().Class("player-item-game")
		if p.ID == g.winnerShineID {
			li = applyShineStyles(li)
		}
		listItems = append(listItems, li.Body(
			app.Img().
				Src(fmt.Sprintf("/web/images/symbol_%02d.png", p.Symbol)).
				Style("width", "32px").Style("height", "32px"),
			app.Span().Text(text),
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

		// Find current player in state to get up-to-date score and finished status
		var currentPlayer *game.Player
		for _, p := range g.State.Players {
			if p.ID == State.Player.ID {
				currentPlayer = p
				break
			}
		}
		if currentPlayer == nil {
			currentPlayer = State.Player
		}

		var playerCardArea app.UI
		var playerInfoText string
		if currentPlayer.Score == 0 {
			playerCardArea = app.Img().Src("/web/images/win.png").Style("width", "520px").Style("height", "520px")
			playerInfoText = fmt.Sprintf("%s (%s)", currentPlayer.Name, currentPlayer.TimeTaken)
			if currentPlayer.IsWinner {
				playerInfoText = "👑 " + playerInfoText
			}
		} else {
			playerCardArea = g.renderCard(State.TopCard, 520, true)
			playerInfoText = fmt.Sprintf("%s (%d cards left)", currentPlayer.Name, currentPlayer.Score)
		}

		// Create New Game button if finished
		var createNewGameBtn app.UI
		if currentPlayer.Score == 0 {
			createNewGameBtn = app.Button().Text("Create New Game").OnClick(func(ctx app.Context, e app.Event) {
				State.Player.Score = 0 // Reset local score logic just in case, though navigation handles it
				ctx.Navigate("/")
			}).Style("margin-top", "1rem")
		}

		content = app.Div().Class("game-grid").Body(
			// First Column (70/30)
			app.Div().Class("game-column").Body(
				// Top: Current Player's info and Card (70%)
				app.Div().Class("column-70").Class("card-container").Body(
					func() app.UI {
						div := app.Div().Style("display", "flex").Style("align-items", "center").Style("gap", "0.5rem").Style("margin-bottom", "0.5rem").Style("padding", "0.5rem").Style("border-radius", "8px")
						if currentPlayer.ID == g.winnerShineID {
							div = div.
								Style("background-color", "rgba(255, 215, 0, 0.25)").
								Style("box-shadow", "inset 0 0 12px 4px rgba(255, 215, 0, 0.6), 0 0 15px 5px rgba(255, 215, 0, 0.4)")
						}
						return div.Body(
							app.Img().
								Src(fmt.Sprintf("/web/images/symbol_%02d.png", currentPlayer.Symbol)).
								Style("width", "32px").Style("height", "32px"),
							app.Strong().Text(playerInfoText),
						)
					}(),
					playerCardArea,
				),
				// Bottom: Other players list - second half (30%)
				app.Div().Class("column-30").Body(
					g.renderPlayerList(bottomLeftPlayers),
					createNewGameBtn,
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
					g.renderCard(State.TargetCard, 520, false),
				),
			),
		)
	} else {
		content = app.Div().Aria("busy", "true").Text("Connecting to game...")
	}

	return app.Main().Class("container").Body(
		&TopBar{},
		content,
	)
}
