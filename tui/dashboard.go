package tui

import (
	"fmt"
	"strings"
	"sync"
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

	mainLayout   *tview.Flex

	tradeMode    string
	symbol       string
	operation    int
	phase        string // "SCANNING", "BUYING", "SELLING", etc.

	// Countdown state
	mu            sync.Mutex
	priceText     string
	refreshSecs   int
	countdown     int
	countdownStop chan struct{}
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
		SetDynamicColors(true).
		SetScrollable(true).
		SetWordWrap(true)
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

	// Layout: header on top, then a row with [price | ai agents], then [indicators | orders]
	topRow := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(d.pricePanel, 0, 1, false).
		AddItem(d.aiPanel, 0, 2, false)

	bottomRow := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(d.indPanel, 0, 1, false).
		AddItem(d.ordersPanel, 0, 2, false)

	mainLayout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(d.header, 3, 0, false).
		AddItem(topRow, 0, 3, false).
		AddItem(bottomRow, 0, 2, false)

	d.mainLayout = mainLayout
	d.app.SetRoot(mainLayout, true)

	// Keyboard shortcuts
	d.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 'q' || event.Key() == tcell.KeyCtrlC {
			d.app.Stop()
			return nil
		}
		if event.Rune() == 'h' {
			d.showHelp(mainLayout)
			return nil
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
	if d.countdownStop != nil {
		select {
		case <-d.countdownStop:
		default:
			close(d.countdownStop)
		}
	}
	d.app.Stop()
}

func (d *Dashboard) headerText() string {
	modeColor := "green"
	if d.tradeMode == "BEAR" {
		modeColor = "red"
	}
	return fmt.Sprintf("[%s::b]%s MODE[-] [white]|[-] [yellow::b]%s[-] [white]|[-] [cyan]Op #%d[-] [white]|[-] [aqua]%s[-] [white]| [red]q[-] quit [white]|[-] [blue]h[-] help",
		modeColor, d.tradeMode, d.symbol, d.operation, d.phase)
}

func (d *Dashboard) updateHeader() {
	d.app.QueueUpdateDraw(func() {
		d.header.SetText(d.headerText())
	})
}

func (d *Dashboard) showHelp(mainLayout *tview.Flex) {
	help := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	help.SetBorder(true).
		SetBorderColor(tcell.ColorGold).
		SetTitle(" Keyboard Shortcuts ").
		SetTitleColor(tcell.ColorGold)

	help.SetText(
		"[yellow::b]Key          Action[-]\n" +
			"[white::b]q[-]            Quit the application\n" +
			"[white::b]h[-]            Toggle this help popup\n" +
			"[white::b]Ctrl+C[-]       Force quit\n" +
			"\n" +
			"[yellow::b]Panels[-]\n" +
			"[cyan]Price[-]         Current price with change indicator\n" +
			"[teal]Indicators[-]    RSI, MACD, Bollinger Bands, DEMA, ADX\n" +
			"[mediumpurple]AI Agents[-]     Consensus from OpenAI, DeepSeek, Claude\n" +
			"[orangered]Orders Log[-]    Trade execution and system messages\n" +
			"\n" +
			"[dimgray]Press [white::b]h[-][dimgray] or [white::b]Esc[-][dimgray] to close[-]")

	// Center the help modal
	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(
			tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(nil, 0, 1, false).
				AddItem(help, 50, 0, true).
				AddItem(nil, 0, 1, false),
			15, 0, true).
		AddItem(nil, 0, 1, false)

	// Overlay modal on top of main layout
	overlay := tview.NewPages().
		AddPage("main", mainLayout, true, true).
		AddPage("help", modal, true, true)

	d.app.SetRoot(overlay, true)

	// Override input to close help on h or Esc
	d.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 'h' || event.Key() == tcell.KeyEscape {
			d.app.SetRoot(mainLayout, true)
			d.restoreInputCapture()
			return nil
		}
		if event.Rune() == 'q' || event.Key() == tcell.KeyCtrlC {
			d.app.Stop()
			return nil
		}
		return event
	})
}

func (d *Dashboard) restoreInputCapture() {
	d.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 'q' || event.Key() == tcell.KeyCtrlC {
			d.app.Stop()
			return nil
		}
		if event.Rune() == 'h' {
			d.showHelp(d.mainLayout)
			return nil
		}
		return event
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

// SetRefreshInterval stores the polling interval and starts a 1-second
// countdown ticker that updates the price panel between polls.
func (d *Dashboard) SetRefreshInterval(interval time.Duration) {
	d.mu.Lock()
	d.refreshSecs = int(interval.Seconds())
	d.countdown = d.refreshSecs
	d.mu.Unlock()

	d.countdownStop = make(chan struct{})
	go d.runCountdown()
}

func (d *Dashboard) runCountdown() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-d.countdownStop:
			return
		case <-ticker.C:
			d.mu.Lock()
			if d.countdown > 0 {
				d.countdown--
			}
			text := d.priceText
			secs := d.countdown
			d.mu.Unlock()

			if text == "" {
				continue
			}
			full := text + fmt.Sprintf("\n[dimgray]Next poll in [white::b]%ds[-]", secs)
			d.app.QueueUpdateDraw(func() {
				d.pricePanel.SetText(full)
			})
		}
	}
}

// UpdatePrice updates the price panel with color-coded current price.
func (d *Dashboard) UpdatePrice(price, prevPrice float64, round uint) {
	now := time.Now().Format("15:04:05")
	priceStr := fmt.Sprintf("%g", price)

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

	d.mu.Lock()
	d.priceText = text
	d.countdown = d.refreshSecs
	d.mu.Unlock()

	full := text
	if d.refreshSecs > 0 {
		full += fmt.Sprintf("\n[dimgray]Next poll in [white::b]%ds[-]", d.refreshSecs)
	}
	d.app.QueueUpdateDraw(func() {
		d.pricePanel.SetText(full)
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
			b.WriteString(fmt.Sprintf("   [gray::i]%s[-]\n", agent.Reasoning))
		}
	}

	d.app.QueueUpdateDraw(func() {
		d.aiPanel.SetText(b.String())
		d.aiPanel.ScrollToBeginning()
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
