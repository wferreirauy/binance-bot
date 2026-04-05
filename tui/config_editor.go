package tui

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/wferreirauy/binance-bot/config"
	yaml "gopkg.in/yaml.v2"
)

// ConfigHolder wraps a config pointer with a mutex for thread-safe runtime access.
type ConfigHolder struct {
	mu  sync.RWMutex
	cfg *config.Config
}

// NewConfigHolder creates a new thread-safe config holder.
func NewConfigHolder(cfg *config.Config) *ConfigHolder {
	return &ConfigHolder{cfg: cfg}
}

// Get returns a copy of the current config.
func (h *ConfigHolder) Get() config.Config {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return *h.cfg
}

// Ptr returns the underlying pointer (use only when you hold the lock externally).
func (h *ConfigHolder) Ptr() *config.Config {
	return h.cfg
}

// Lock locks for writing.
func (h *ConfigHolder) Lock() { h.mu.Lock() }

// Unlock unlocks writing.
func (h *ConfigHolder) Unlock() { h.mu.Unlock() }

// RLock locks for reading.
func (h *ConfigHolder) RLock() { h.mu.RLock() }

// RUnlock unlocks reading.
func (h *ConfigHolder) RUnlock() { h.mu.RUnlock() }

// showConfigViewer displays the current config as YAML in a modal.
func (d *Dashboard) showConfigViewer() {
	if d.configHolder == nil {
		d.LogInfo("No config available to display")
		return
	}

	cfg := d.configHolder.Get()
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		d.LogError(fmt.Sprintf("Config marshal: %v", err))
		return
	}

	viewer := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWordWrap(true)
	viewer.SetBorder(true).
		SetBorderColor(tcell.ColorGold).
		SetTitle(" Current Configuration (press Esc to close) ").
		SetTitleColor(tcell.ColorGold)

	// Syntax-highlight the YAML
	highlighted := highlightYAML(string(data))
	viewer.SetText(highlighted)

	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(
			tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(nil, 0, 1, false).
				AddItem(viewer, 70, 0, true).
				AddItem(nil, 0, 1, false),
			0, 4, true).
		AddItem(nil, 0, 1, false)

	overlay := tview.NewPages().
		AddPage("main", d.mainLayout, true, true).
		AddPage("config-view", modal, true, true)

	d.app.SetRoot(overlay, true)

	d.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			d.app.SetRoot(d.mainLayout, true)
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

// highlightYAML adds tview color tags to YAML text.
func highlightYAML(text string) string {
	var b strings.Builder
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == "" {
			b.WriteString("\n")
			continue
		}
		if idx := strings.Index(line, ":"); idx >= 0 {
			key := line[:idx]
			val := line[idx:]
			b.WriteString(fmt.Sprintf("[cyan]%s[white]%s[-]\n", key, val))
		} else {
			b.WriteString(fmt.Sprintf("[white]%s[-]\n", line))
		}
	}
	return b.String()
}

// showConfigEditor displays an interactive form to edit config values at runtime.
func (d *Dashboard) showConfigEditor() {
	if d.configHolder == nil {
		d.LogInfo("No config available to edit")
		return
	}

	cfg := d.configHolder.Get()

	form := tview.NewForm()
	form.SetBorder(true).
		SetBorderColor(tcell.ColorMediumPurple).
		SetTitle(" Edit Configuration (Tab/Shift-Tab to navigate, Enter to save) ").
		SetTitleColor(tcell.ColorMediumPurple)

	// Historical Prices
	form.AddInputField("Hist. Period", fmt.Sprintf("%d", cfg.HistoricalPrices.Period), 10, nil, nil)
	form.AddInputField("Hist. Interval", cfg.HistoricalPrices.Interval, 10, nil, nil)

	// Refresh Interval
	form.AddInputField("Refresh Interval (s)", fmt.Sprintf("%d", cfg.RefreshInterval), 10, nil, nil)

	// Tendency
	form.AddInputField("Tendency Interval", cfg.Tendency.Interval, 10, nil, nil)
	form.AddInputField("Tendency Direction", cfg.Tendency.Direction, 10, nil, nil)

	// RSI
	form.AddInputField("RSI Interval", cfg.Indicators.Rsi.Interval, 10, nil, nil)
	form.AddInputField("RSI Length", fmt.Sprintf("%d", cfg.Indicators.Rsi.Length), 10, nil, nil)
	form.AddInputField("RSI Upper Limit", fmt.Sprintf("%d", cfg.Indicators.Rsi.UpperLimit), 10, nil, nil)
	form.AddInputField("RSI Middle Limit", fmt.Sprintf("%d", cfg.Indicators.Rsi.MiddleLimit), 10, nil, nil)
	form.AddInputField("RSI Lower Limit", fmt.Sprintf("%d", cfg.Indicators.Rsi.LowerLimit), 10, nil, nil)

	// DEMA
	form.AddInputField("DEMA Length", fmt.Sprintf("%d", cfg.Indicators.Dema.Length), 10, nil, nil)

	// MACD
	form.AddInputField("MACD Fast Length", fmt.Sprintf("%d", cfg.Indicators.Macd.FastLength), 10, nil, nil)
	form.AddInputField("MACD Slow Length", fmt.Sprintf("%d", cfg.Indicators.Macd.SlowLength), 10, nil, nil)
	form.AddInputField("MACD Signal Length", fmt.Sprintf("%d", cfg.Indicators.Macd.SignalLength), 10, nil, nil)

	// Bollinger Bands
	form.AddInputField("BB Length", fmt.Sprintf("%d", cfg.Indicators.BollingerBands.Length), 10, nil, nil)
	form.AddInputField("BB Multiplier", fmt.Sprintf("%.1f", cfg.Indicators.BollingerBands.Multiplier), 10, nil, nil)

	// ATR
	form.AddInputField("ATR Period", fmt.Sprintf("%d", cfg.Indicators.Atr.Period), 10, nil, nil)

	// ADX
	form.AddInputField("ADX Period", fmt.Sprintf("%d", cfg.Indicators.Adx.Period), 10, nil, nil)
	form.AddInputField("ADX Threshold", fmt.Sprintf("%d", cfg.Indicators.Adx.Threshold), 10, nil, nil)

	// Volume
	form.AddInputField("Volume MA Period", fmt.Sprintf("%d", cfg.Indicators.Volume.MaPeriod), 10, nil, nil)

	// Trailing Stop
	form.AddCheckbox("Trailing Stop Enabled", cfg.TrailingStop.Enabled, nil)
	form.AddInputField("Trailing Activation %", fmt.Sprintf("%.2f", cfg.TrailingStop.ActivationPct), 10, nil, nil)
	form.AddInputField("Trailing %", fmt.Sprintf("%.2f", cfg.TrailingStop.TrailingPct), 10, nil, nil)

	// AI
	form.AddCheckbox("AI Enabled", cfg.AI.Enabled, nil)
	form.AddInputField("AI Min Confidence", fmt.Sprintf("%.2f", cfg.AI.MinConfidence), 10, nil, nil)

	// Save button
	form.AddButton("Save", func() {
		d.applyConfigForm(form)
		d.app.SetRoot(d.mainLayout, true)
		d.restoreInputCapture()
		d.LogInfo("[green]Configuration updated at runtime[-]")
	})

	// Cancel button
	form.AddButton("Cancel", func() {
		d.app.SetRoot(d.mainLayout, true)
		d.restoreInputCapture()
	})

	form.SetFieldBackgroundColor(tcell.ColorDarkSlateGray)
	form.SetButtonBackgroundColor(tcell.ColorDarkCyan)

	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(
			tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(nil, 0, 1, false).
				AddItem(form, 60, 0, true).
				AddItem(nil, 0, 1, false),
			0, 4, true).
		AddItem(nil, 0, 1, false)

	overlay := tview.NewPages().
		AddPage("main", d.mainLayout, true, true).
		AddPage("config-edit", modal, true, true)

	d.app.SetRoot(overlay, true)

	d.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			d.app.SetRoot(d.mainLayout, true)
			d.restoreInputCapture()
			return nil
		}
		return event
	})
}

// applyConfigForm reads form values and applies them to the live config.
func (d *Dashboard) applyConfigForm(form *tview.Form) {
	d.configHolder.Lock()
	defer d.configHolder.Unlock()

	cfg := d.configHolder.Ptr()

	// Helper to parse int fields safely
	parseInt := func(label string) int {
		val := form.GetFormItemByLabel(label).(*tview.InputField).GetText()
		n, err := strconv.Atoi(strings.TrimSpace(val))
		if err != nil {
			d.LogError(fmt.Sprintf("Invalid integer for %s: %s", label, val))
			return 0
		}
		return n
	}

	parseFloat := func(label string) float64 {
		val := form.GetFormItemByLabel(label).(*tview.InputField).GetText()
		f, err := strconv.ParseFloat(strings.TrimSpace(val), 64)
		if err != nil {
			d.LogError(fmt.Sprintf("Invalid float for %s: %s", label, val))
			return 0
		}
		return f
	}

	getString := func(label string) string {
		return form.GetFormItemByLabel(label).(*tview.InputField).GetText()
	}

	getBool := func(label string) bool {
		return form.GetFormItemByLabel(label).(*tview.Checkbox).IsChecked()
	}

	// Apply values
	cfg.HistoricalPrices.Period = parseInt("Hist. Period")
	cfg.HistoricalPrices.Interval = getString("Hist. Interval")
	cfg.RefreshInterval = parseInt("Refresh Interval (s)")
	cfg.Tendency.Interval = getString("Tendency Interval")
	cfg.Tendency.Direction = getString("Tendency Direction")

	cfg.Indicators.Rsi.Interval = getString("RSI Interval")
	cfg.Indicators.Rsi.Length = parseInt("RSI Length")
	cfg.Indicators.Rsi.UpperLimit = parseInt("RSI Upper Limit")
	cfg.Indicators.Rsi.MiddleLimit = parseInt("RSI Middle Limit")
	cfg.Indicators.Rsi.LowerLimit = parseInt("RSI Lower Limit")

	cfg.Indicators.Dema.Length = parseInt("DEMA Length")

	cfg.Indicators.Macd.FastLength = parseInt("MACD Fast Length")
	cfg.Indicators.Macd.SlowLength = parseInt("MACD Slow Length")
	cfg.Indicators.Macd.SignalLength = parseInt("MACD Signal Length")

	cfg.Indicators.BollingerBands.Length = parseInt("BB Length")
	cfg.Indicators.BollingerBands.Multiplier = parseFloat("BB Multiplier")

	cfg.Indicators.Atr.Period = parseInt("ATR Period")

	cfg.Indicators.Adx.Period = parseInt("ADX Period")
	cfg.Indicators.Adx.Threshold = parseInt("ADX Threshold")

	cfg.Indicators.Volume.MaPeriod = parseInt("Volume MA Period")

	cfg.TrailingStop.Enabled = getBool("Trailing Stop Enabled")
	cfg.TrailingStop.ActivationPct = parseFloat("Trailing Activation %")
	cfg.TrailingStop.TrailingPct = parseFloat("Trailing %")

	cfg.AI.Enabled = getBool("AI Enabled")
	cfg.AI.MinConfidence = parseFloat("AI Min Confidence")
}
