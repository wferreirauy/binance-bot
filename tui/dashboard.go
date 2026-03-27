package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Dashboard manages the multi-panel terminal UI for the trading bot.
type Dashboard struct {
	app          *tview.Application
	header       *tview.TextView
	pricePanel   *tview.TextView
	indPanel     *tview.TextView
	aiPanel      *tview.TextView
	ordersPanel  *tview.TextView

	tradeMode    string
	symbol       string
	operation    int
	phase        string // "SCANNING", "BUYING", "SELLING", etc.
}

// NewDashboard creates a new TUI dashboard with multi-panel layout.
func NewDashboard(tradeMode, symbol string) *Dashboard {
	d := &Dashboard{
		app:       tview.NewApplication(),
		tradeMode: tradeMode,
		symbol:    symbol,
		operation: 1,
		phase:     "SCANNING",
	}

	// Header panel
	d.header = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	d.header.SetBorder(true).
		SetBorderColor(tcell.ColorDodgerBlue).
		SetTitle(" Binance Trading Bot ").
		SetTitleColor(tcell.ColorGold)

	// Price panel
	d.pricePanel = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	d.pricePanel.SetBorder(true).
		SetBorderColor(tcell.ColorWhite).
		SetTitle(" Price ").
		SetTitleColor(tcell.ColorWhite)

	// Indicators panel
	d.indPanel = tview.NewTextView().
		SetDynamicColors(true)
	d.indPanel.SetBorder(true).
		SetBorderColor(tcell.ColorTeal).
		SetTitle(" Indicators ").
		SetTitleColor(tcell.ColorTeal)

	// AI Agents panel
	d.aiPanel = tview.NewTextView().
		SetDynamicColors(true)
	d.aiPanel.SetBorder(true).
		SetBorderColor(tcell.ColorMediumPurple).
		SetTitle(" AI Agents ").
		SetTitleColor(tcell.ColorMediumPurple)

	// Orders log panel
	d.ordersPanel = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetMaxLines(200)
	d.ordersPanel.SetBorder(true).
		SetBorderColor(tcell.ColorOrangeRed).
		SetTitle(" Orders Log ").
		SetTitleColor(tcell.ColorOrangeRed)

	// Layout: header on top, then a row with [price | indicators], then [ai | orders]
	topRow := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(d.pricePanel, 0, 1, false).
		AddItem(d.indPanel, 0, 2, false)

	bottomRow := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(d.aiPanel, 0, 1, false).
		AddItem(d.ordersPanel, 0, 1, false)

	mainLayout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(d.header, 3, 0, false).
		AddItem(topRow, 0, 2, false).
		AddItem(bottomRow, 0, 3, false)

	d.app.SetRoot(mainLayout, true)

	// Allow 'q' to quit
	d.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 'q' || event.Key() == tcell.KeyCtrlC {
			d.app.Stop()
		}
		return event
	})

	d.header.SetText(d.headerText())

	return d
}

// Run starts the TUI event loop (blocking). Call from a goroutine.
func (d *Dashboard) Run() error {
	return d.app.Run()
}

// Stop gracefully stops the TUI application.
func (d *Dashboard) Stop() {
	d.app.Stop()
}

func (d *Dashboard) headerText() string {
	modeColor := "green"
	if d.tradeMode == "BEAR" {
		modeColor = "red"
	}
	return fmt.Sprintf("[%s::b]%s MODE[-] [white]|[-] [yellow::b]%s[-] [white]|[-] [cyan]Op #%d[-] [white]|[-] [aqua]%s[-] [white]| Press [red]q[-] to quit",
		modeColor, d.tradeMode, d.symbol, d.operation, d.phase)
}

func (d *Dashboard) updateHeader() {
	d.app.QueueUpdateDraw(func() {
		d.header.SetText(d.headerText())
	})
}

// SetOperation updates the current operation number.
func (d *Dashboard) SetOperation(n int) {
	d.operation = n
	d.updateHeader()
}

// SetPhase updates the current trading phase displayed in the header.
func (d *Dashboard) SetPhase(phase string) {
	d.phase = phase
	d.updateHeader()
}

// IndicatorData holds indicator values for display.
type IndicatorData struct {
	RSI           float64
	RSIUpperLimit int
	RSILowerLimit int
	MACDLine      float64
	SignalLine    float64
	MACDCross     string // "BULLISH" or "BEARISH"
	DEMA          float64
	UpperBand     float64
	LowerBand     float64
	Tendency      string
	ADX           float64
	ADXThreshold  int
	Volume        float64
	AvgVolume     float64
}

// UpdatePrice updates the price panel with color-coded current price.
func (d *Dashboard) UpdatePrice(price, prevPrice float64, round uint) {
	now := time.Now().Format("15:04:05")
	priceStr := fmt.Sprintf("%.*f", round, price)

	var colorTag string
	var arrow string
	change := 0.0
	if prevPrice > 0 {
		change = (price - prevPrice) / prevPrice * 100
	}

	switch {
	case price > prevPrice:
		colorTag = "green"
		arrow = "▲"
	case price < prevPrice:
		colorTag = "red"
		arrow = "▼"
	default:
		colorTag = "white"
		arrow = "▬"
	}

	text := fmt.Sprintf("\n[%s::b]%s %s[-]\n[gray]%s | %+.2f%%[-]",
		colorTag, arrow, priceStr, now, change)

	d.app.QueueUpdateDraw(func() {
		d.pricePanel.SetText(text)
	})
}

// UpdateIndicators refreshes the indicators panel.
func (d *Dashboard) UpdateIndicators(ind *IndicatorData) {
	var b strings.Builder

	// RSI with color coding
	rsiColor := "yellow"
	if ind.RSI >= float64(ind.RSIUpperLimit) {
		rsiColor = "red"
	} else if ind.RSI <= float64(ind.RSILowerLimit) {
		rsiColor = "green"
	}
	b.WriteString(fmt.Sprintf(" [white::b]RSI:[:-]         [%s]%.1f[-]", rsiColor, ind.RSI))
	if ind.RSIUpperLimit > 0 {
		b.WriteString(fmt.Sprintf(" [gray](<%d >%d)[-]", ind.RSILowerLimit, ind.RSIUpperLimit))
	}
	b.WriteString("\n")

	// MACD
	macdColor := "green"
	if ind.MACDCross == "BEARISH" {
		macdColor = "red"
	}
	b.WriteString(fmt.Sprintf(" [white::b]MACD:[:-]        [%s]%s[-]\n", macdColor, ind.MACDCross))
	b.WriteString(fmt.Sprintf("                [gray]Line: %.6f | Signal: %.6f[-]\n", ind.MACDLine, ind.SignalLine))

	// Bollinger Bands
	b.WriteString(fmt.Sprintf(" [white::b]Bollinger:[:-]   [aqua]%.4f[-] - [aqua]%.4f[-]\n", ind.LowerBand, ind.UpperBand))

	// DEMA
	b.WriteString(fmt.Sprintf(" [white::b]DEMA:[:-]        [white]%.4f[-]\n", ind.DEMA))

	// Tendency
	tendColor := "green"
	if ind.Tendency == "down" {
		tendColor = "red"
	}
	b.WriteString(fmt.Sprintf(" [white::b]Tendency:[:-]    [%s::b]%s[-]\n", tendColor, strings.ToUpper(ind.Tendency)))

	// ADX
	if ind.ADX > 0 {
		adxColor := "yellow"
		if ind.ADX > float64(ind.ADXThreshold) {
			adxColor = "green"
		}
		b.WriteString(fmt.Sprintf(" [white::b]ADX:[:-]         [%s]%.1f[-]", adxColor, ind.ADX))
		if ind.ADXThreshold > 0 {
			b.WriteString(fmt.Sprintf(" [gray](threshold: %d)[-]", ind.ADXThreshold))
		}
		b.WriteString("\n")
	}

	// Volume
	if ind.AvgVolume > 0 {
		volRatio := ind.Volume / ind.AvgVolume
		volColor := "yellow"
		if volRatio > 1.0 {
			volColor = "green"
		}
		b.WriteString(fmt.Sprintf(" [white::b]Volume:[:-]      [%s]%.0f[-] [gray](avg: %.0f, ratio: %.2fx)[-]\n", volColor, ind.Volume, ind.AvgVolume, volRatio))
	}

	d.app.QueueUpdateDraw(func() {
		d.indPanel.SetText(b.String())
	})
}

// AgentResult holds a single AI agent's decision for display.
type AgentResult struct {
	Provider   string
	Signal     string
	Confidence float64
	Reasoning  string
}

// AIConsensusData holds the full AI consensus for display.
type AIConsensusData struct {
	FinalSignal   string
	AvgConfidence float64
	BuyScore      float64
	SellScore     float64
	HoldScore     float64
	Agents        []AgentResult
	FearGreed     int
	FearGreedLabel string
}

// providerColor returns the tview color tag for each AI provider.
func providerColor(provider string) string {
	switch strings.ToLower(provider) {
	case "openai":
		return "green"
	case "deepseek":
		return "dodgerblue"
	case "claude":
		return "darkorange"
	default:
		return "white"
	}
}

// signalColor returns a color tag for a trading signal.
func signalColor(signal string) string {
	switch strings.ToUpper(signal) {
	case "BUY":
		return "green"
	case "SELL":
		return "red"
	case "HOLD":
		return "yellow"
	default:
		return "gray"
	}
}

// UpdateAI refreshes the AI agents panel with consensus and per-agent results.
func (d *Dashboard) UpdateAI(data *AIConsensusData) {
	var b strings.Builder

	if data == nil {
		b.WriteString(" [gray]AI analysis disabled[-]\n")
		d.app.QueueUpdateDraw(func() {
			d.aiPanel.SetText(b.String())
		})
		return
	}

	// Consensus header
	sigCol := signalColor(data.FinalSignal)
	b.WriteString(fmt.Sprintf(" [white::b]Consensus:[:-]  [%s::b]%s[-] [gray](%.0f%% confidence)[-]\n",
		sigCol, data.FinalSignal, data.AvgConfidence*100))
	b.WriteString(fmt.Sprintf(" [white::b]Scores:[:-]     [green]Buy %.0f%%[-] [red]Sell %.0f%%[-] [yellow]Hold %.0f%%[-]\n",
		data.BuyScore*100, data.SellScore*100, data.HoldScore*100))

	// Fear & Greed
	if data.FearGreed >= 0 {
		fgColor := "yellow"
		if data.FearGreed <= 25 {
			fgColor = "red"
		} else if data.FearGreed >= 75 {
			fgColor = "green"
		}
		b.WriteString(fmt.Sprintf(" [white::b]Fear/Greed:[:-] [%s]%d[-] [gray](%s)[-]\n", fgColor, data.FearGreed, data.FearGreedLabel))
	}

	b.WriteString(" [gray]───────────────────────────────[-]\n")

	// Per-agent results
	for _, agent := range data.Agents {
		pColor := providerColor(agent.Provider)
		sColor := signalColor(agent.Signal)
		b.WriteString(fmt.Sprintf(" [%s::b]%-10s[-] [%s]%-4s[-] [white]%.0f%%[-]",
			pColor, strings.ToUpper(agent.Provider), sColor, agent.Signal, agent.Confidence*100))
		b.WriteString("\n")
		if agent.Reasoning != "" {
			// Truncate reasoning to fit panel
			reasoning := agent.Reasoning
			if len(reasoning) > 120 {
				reasoning = reasoning[:117] + "..."
			}
			b.WriteString(fmt.Sprintf("   [gray::i]%s[-]\n", reasoning))
		}
	}

	d.app.QueueUpdateDraw(func() {
		d.aiPanel.SetText(b.String())
	})
}

// LogOrder appends an order event to the orders log panel.
func (d *Dashboard) LogOrder(text string) {
	now := time.Now().Format("15:04:05")
	line := fmt.Sprintf("[gray]%s[-] %s\n", now, text)
	d.app.QueueUpdateDraw(func() {
		fmt.Fprint(d.ordersPanel, line)
		d.ordersPanel.ScrollToEnd()
	})
}

// LogInfo appends an informational message to the orders log.
func (d *Dashboard) LogInfo(msg string) {
	now := time.Now().Format("15:04:05")
	line := fmt.Sprintf("[gray]%s[-] [aqua]%s[-]\n", now, msg)
	d.app.QueueUpdateDraw(func() {
		fmt.Fprint(d.ordersPanel, line)
		d.ordersPanel.ScrollToEnd()
	})
}

// LogError appends an error message to the orders log.
func (d *Dashboard) LogError(msg string) {
	now := time.Now().Format("15:04:05")
	line := fmt.Sprintf("[gray]%s[-] [red]ERROR: %s[-]\n", now, msg)
	d.app.QueueUpdateDraw(func() {
		fmt.Fprint(d.ordersPanel, line)
		d.ordersPanel.ScrollToEnd()
	})
}
