package tui

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// GainerRow holds one row of top-gainer data for display.
type GainerRow struct {
	Rank        int
	Symbol      string
	Price       float64
	ChangePct   float64
	Volume      float64
	QuoteVolume float64
}

// GainersDashboard is a TUI for monitoring top market gainers.
type GainersDashboard struct {
	app        *tview.Application
	header     *tview.TextView
	table      *tview.TextView
	logPanel   *tview.TextView
	mainLayout *tview.Flex

	quoteAsset   string
	limit        int
	pollInterval time.Duration

	mu            sync.Mutex
	lastUpdate    time.Time
	countdownStop chan struct{}

	fileLogger *FileLogger
}

// NewGainersDashboard creates a new TUI for the top gainers monitor.
func NewGainersDashboard(quoteAsset string, limit int, pollInterval time.Duration) *GainersDashboard {
	d := &GainersDashboard{
		app:          tview.NewApplication(),
		quoteAsset:   quoteAsset,
		limit:        limit,
		pollInterval: pollInterval,
	}

	// Header
	d.header = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	d.header.SetBorder(true).
		SetBorderColor(tcell.ColorDodgerBlue).
		SetTitle(" Binance Top Gainers Monitor ").
		SetTitleColor(tcell.ColorGold)
	d.header.SetText(fmt.Sprintf("[yellow::b]Top %d Gainers[-] [white]|[-] [cyan]%s[-] [white]|[-] [aqua]Poll: %ds[-] [white]| [red]q[-] quit",
		limit, quoteAsset, int(pollInterval.Seconds())))

	// Table panel (text-based table for gainers)
	d.table = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)
	d.table.SetBorder(true).
		SetBorderColor(tcell.ColorTeal).
		SetTitle(" Top Gainers (24h) ").
		SetTitleColor(tcell.ColorTeal)

	// Log panel
	d.logPanel = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetMaxLines(100)
	d.logPanel.SetBorder(true).
		SetBorderColor(tcell.ColorOrangeRed).
		SetTitle(" Log ").
		SetTitleColor(tcell.ColorOrangeRed)

	mainLayout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(d.header, 3, 0, false).
		AddItem(d.table, 0, 4, false).
		AddItem(d.logPanel, 8, 0, false)

	d.mainLayout = mainLayout
	d.app.SetRoot(mainLayout, true)

	d.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 'q' || event.Key() == tcell.KeyCtrlC {
			d.app.Stop()
			return nil
		}
		return event
	})

	return d
}

// Run starts the TUI event loop.
func (d *GainersDashboard) Run() error {
	return d.app.Run()
}

// Stop gracefully stops the TUI.
func (d *GainersDashboard) Stop() {
	d.app.Stop()
}

// UpdateGainers refreshes the table with new gainer data.
func (d *GainersDashboard) UpdateGainers(rows []GainerRow) {
	var b strings.Builder

	// Header row
	b.WriteString(fmt.Sprintf(" [yellow::b]%-4s %-12s %12s %10s %14s %14s[-]\n",
		"#", "SYMBOL", "PRICE", "CHG %", "VOLUME", "QUOTE VOL"))
	b.WriteString(fmt.Sprintf(" [gray]%s[-]\n", strings.Repeat("─", 70)))

	for _, r := range rows {
		chgColor := "green"
		if r.ChangePct < 0 {
			chgColor = "red"
		}

		b.WriteString(fmt.Sprintf(" [white]%-4d[-] [cyan::b]%-12s[-] [white]%12.4f[-] [%s::b]%+9.2f%%[-] [gray]%14.0f %14.0f[-]\n",
			r.Rank, r.Symbol, r.Price, chgColor, r.ChangePct, r.Volume, r.QuoteVolume))
	}

	now := time.Now().Format("15:04:05")
	b.WriteString(fmt.Sprintf("\n [dimgray]Last update: %s[-]", now))

	d.app.QueueUpdateDraw(func() {
		d.table.SetText(b.String())
	})
}

// SetFileLogger attaches a file logger so log messages are also written to disk.
func (d *GainersDashboard) SetFileLogger(fl *FileLogger) {
	d.fileLogger = fl
}

// LogInfo appends an info message to the log panel.
func (d *GainersDashboard) LogInfo(msg string) {
	now := time.Now().Format("15:04:05")
	line := fmt.Sprintf("[gray]%s[-] [aqua]%s[-]\n", now, msg)
	d.app.QueueUpdateDraw(func() {
		fmt.Fprint(d.logPanel, line)
		d.logPanel.ScrollToEnd()
	})
	if d.fileLogger != nil {
		d.fileLogger.Log("INFO", msg)
	}
}

// LogError appends an error message to the log panel.
func (d *GainersDashboard) LogError(msg string) {
	now := time.Now().Format("15:04:05")
	line := fmt.Sprintf("[gray]%s[-] [red]ERROR: %s[-]\n", now, msg)
	d.app.QueueUpdateDraw(func() {
		fmt.Fprint(d.logPanel, line)
		d.logPanel.ScrollToEnd()
	})
	if d.fileLogger != nil {
		d.fileLogger.Log("ERROR", msg)
	}
}
