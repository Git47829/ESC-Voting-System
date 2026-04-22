package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── Styles ──────────────────────────────────────────────────────────────────

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 2)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF79C6"))

	goodStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B"))
	warnStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#F1FA8C"))
	badStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555"))
	dimStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4"))

	barFull  = lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B"))
	barEmpty = lipgloss.NewStyle().Foreground(lipgloss.Color("#44475A"))

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#BD93F9")).
			Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#50FA7B"))
)

// ── Endpoint definitions ────────────────────────────────────────────────────

type Endpoint struct {
	Name   string
	Method string
	Path   string
	// GenPath returns a dynamic path for each request (for parameterized endpoints)
	GenPath func() string
}

func defaultEndpoints() []Endpoint {
	return []Endpoint{
		{Name: "Health", Method: "GET", Path: "/health"},
		{Name: "Votes", Method: "GET", Path: "/votes/"},
		{Name: "Countries", Method: "GET", Path: "/countries/"},
		{Name: "Songs", Method: "GET", Path: "/songs/"},
		{Name: "Contest", Method: "GET", Path: "/contest/current"},
		{Name: "Vote (POST)", Method: "POST", Path: "/vote/",
			GenPath: func() string {
				countries := []string{"DE", "FR", "IT", "ES", "SE", "NL", "AT", "CH"}
				phones := []string{"+4915112345678", "+33612345678", "+39312345678", "+34612345678"}
				songID := rand.Intn(10) + 1
				points := rand.Intn(5) + 1
				return fmt.Sprintf("/vote/?ownCountry=%s&phoneNum=%s&songID=%d&points=%d",
					countries[rand.Intn(len(countries))],
					phones[rand.Intn(len(phones))],
					songID, points)
			},
		},
		{Name: "ESC Points", Method: "GET", Path: "/esc-converter/api/esc-points"},
	}
}

// ── Stats collector ─────────────────────────────────────────────────────────

type RequestResult struct {
	Duration   time.Duration
	StatusCode int
	Err        error
	TimedOut   bool
}

type Stats struct {
	mu           sync.Mutex
	totalReqs    atomic.Int64
	totalErrors  atomic.Int64
	timedOut     atomic.Bool
	timeoutAt    time.Duration // latency of first timeout
	latencies    []time.Duration
	statusCodes  map[int]int64
	startTime    time.Time
	lastSnapshot time.Time
	windowReqs   int64
	currentRPS   float64
}

func NewStats() *Stats {
	return &Stats{
		statusCodes: make(map[int]int64),
		startTime:   time.Now(),
	}
}

func (s *Stats) Record(r RequestResult) {
	if r.Err != nil {
		s.totalErrors.Add(1)
	}
	if r.TimedOut {
		if s.timedOut.CompareAndSwap(false, true) {
			s.timeoutAt = r.Duration
		}
	}
	s.totalReqs.Add(1)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.latencies = append(s.latencies, r.Duration)
	if r.StatusCode > 0 {
		s.statusCodes[r.StatusCode]++
	}
	s.windowReqs++
}

func (s *Stats) Snapshot() StatsSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(s.lastSnapshot).Seconds()
	if elapsed > 0 {
		s.currentRPS = float64(s.windowReqs) / elapsed
	}
	s.lastSnapshot = now
	s.windowReqs = 0

	snap := StatsSnapshot{
		TotalRequests:  s.totalReqs.Load(),
		TotalErrors:    s.totalErrors.Load(),
		Elapsed:        now.Sub(s.startTime),
		CurrentRPS:     s.currentRPS,
		StatusCodes:    make(map[int]int64),
		TimedOut:       s.timedOut.Load(),
		TimeoutLatency: s.timeoutAt,
	}

	for k, v := range s.statusCodes {
		snap.StatusCodes[k] = v
	}

	if len(s.latencies) > 0 {
		sorted := make([]time.Duration, len(s.latencies))
		copy(sorted, s.latencies)
		sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

		snap.P50 = sorted[len(sorted)*50/100]
		snap.P90 = sorted[len(sorted)*90/100]
		snap.P95 = sorted[len(sorted)*95/100]
		snap.P99 = sorted[len(sorted)*99/100]
		snap.Min = sorted[0]
		snap.Max = sorted[len(sorted)-1]

		var total time.Duration
		for _, d := range sorted {
			total += d
		}
		snap.Avg = total / time.Duration(len(sorted))
	}

	return snap
}

type StatsSnapshot struct {
	TotalRequests int64
	TotalErrors   int64
	Elapsed       time.Duration
	CurrentRPS    float64
	P50, P90, P95, P99 time.Duration
	Min, Max, Avg      time.Duration
	StatusCodes        map[int]int64
	TimedOut           bool
	TimeoutLatency     time.Duration
}

// ── Worker pool ─────────────────────────────────────────────────────────────

type LoadTest struct {
	baseURL     string
	client      *http.Client
	stats       *Stats
	endpoints   []Endpoint
	selected    int
	concurrency int
	running     bool
	stopCh      chan struct{}
}

func NewLoadTest(baseURL string, concurrency int) *LoadTest {
	tr := &http.Transport{
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		MaxIdleConns:        concurrency * 2,
		MaxIdleConnsPerHost: concurrency * 2,
		IdleConnTimeout:     30 * time.Second,
	}
	return &LoadTest{
		baseURL: strings.TrimRight(baseURL, "/"),
		client: &http.Client{
			Transport: tr,
			Timeout:   10 * time.Second,
		},
		stats:       NewStats(),
		endpoints:   defaultEndpoints(),
		concurrency: concurrency,
		stopCh:      make(chan struct{}),
	}
}

func (lt *LoadTest) Start() {
	if lt.running {
		return
	}
	lt.running = true
	lt.stats = NewStats()
	lt.stopCh = make(chan struct{})

	ep := lt.endpoints[lt.selected]

	for i := 0; i < lt.concurrency; i++ {
		go func() {
			for {
				select {
				case <-lt.stopCh:
					return
				default:
					path := ep.Path
					if ep.GenPath != nil {
						path = ep.GenPath()
					}
					url := lt.baseURL + path

					start := time.Now()
					req, _ := http.NewRequest(ep.Method, url, nil)
					resp, err := lt.client.Do(req)
					dur := time.Since(start)

					result := RequestResult{Duration: dur, Err: err}
					if resp != nil {
						result.StatusCode = resp.StatusCode
						resp.Body.Close()
					}
					if err != nil && os.IsTimeout(err) {
						result.TimedOut = true
					}
					lt.stats.Record(result)
				}
			}
		}()
	}
}

func (lt *LoadTest) Stop() {
	if !lt.running {
		return
	}
	close(lt.stopCh)
	lt.running = false
}

// ── TUI Model ───────────────────────────────────────────────────────────────

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

type screen int

const (
	screenConfig screen = iota
	screenRunning
	screenTimeout
)

type model struct {
	lt          *LoadTest
	snap        StatsSnapshot
	screen      screen
	cursor      int // for config screen
	concurrency int
	width       int
	height      int
	maxRPS      float64
	rpsHistory  []float64

	// autoStress mode
	autoStress       bool
	autoWorkers      int
	autoStep         int
	autoLastRPS      float64
	autoHistory      []autoStressEntry
	autoWaitTicks    int // ticks to wait before ramping
}

type autoStressEntry struct {
	Workers int
	RPS     float64
	P99     time.Duration
}

func initialModel(baseURL string, concurrency int) model {
	return model{
		lt:          NewLoadTest(baseURL, concurrency),
		screen:      screenConfig,
		concurrency: concurrency,
	}
}

func (m model) Init() tea.Cmd {
	return tickCmd()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.lt.Stop()
			return m, tea.Quit

		case "up", "k":
			if m.screen == screenConfig && m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.screen == screenConfig && m.cursor < len(m.lt.endpoints)-1+2 {
				m.cursor++
			}
		case "left", "h":
			if m.screen == screenConfig {
				idx := m.cursor - len(m.lt.endpoints)
				if idx == 0 && m.concurrency > 1 {
					m.concurrency--
				}
			}
		case "right", "l":
			if m.screen == screenConfig {
				idx := m.cursor - len(m.lt.endpoints)
				if idx == 0 && m.concurrency < 1000 {
					m.concurrency++
				}
			}
		case "+", "=":
			if m.screen == screenConfig && m.concurrency < 1000 {
				m.concurrency += 10
				if m.concurrency > 1000 {
					m.concurrency = 1000
				}
			}
		case "-":
			if m.screen == screenConfig && m.concurrency > 1 {
				m.concurrency -= 10
				if m.concurrency < 1 {
					m.concurrency = 1
				}
			}

		case "enter", " ":
			if m.screen == screenConfig {
				if m.cursor < len(m.lt.endpoints) {
					m.lt.selected = m.cursor
				} else {
					// Start button
					m.lt.concurrency = m.concurrency
					m.lt.Start()
					m.screen = screenRunning
					m.maxRPS = 0
					m.rpsHistory = nil
				}
			}
		case "s":
			if m.screen == screenRunning {
				m.lt.Stop()
				m.autoStress = false
				m.screen = screenConfig
			}
		case "r":
			if m.screen == screenConfig {
				m.lt.concurrency = m.concurrency
				m.lt.Start()
				m.screen = screenRunning
				m.maxRPS = 0
				m.rpsHistory = nil
				m.autoStress = false
			}
		case "a":
			if m.screen == screenConfig {
				// Start autoStress mode
				m.autoStress = true
				m.autoWorkers = 1
				m.autoStep = max(m.concurrency/10, 1)
				m.concurrency = 1
				m.lt.concurrency = 1
				m.autoHistory = nil
				m.autoWaitTicks = 0
				m.maxRPS = 0
				m.rpsHistory = nil
				m.lt.Start()
				m.screen = screenRunning
			}
		case "b":
			if m.screen == screenTimeout {
				m.screen = screenConfig
			}
		}

	case tickMsg:
		if m.lt.running {
			m.snap = m.lt.stats.Snapshot()
			if m.snap.CurrentRPS > m.maxRPS {
				m.maxRPS = m.snap.CurrentRPS
			}
			m.rpsHistory = append(m.rpsHistory, m.snap.CurrentRPS)
			if len(m.rpsHistory) > 60 {
				m.rpsHistory = m.rpsHistory[len(m.rpsHistory)-60:]
			}

			// Stop on first timeout
			if m.snap.TimedOut {
				m.lt.Stop()
				if m.autoStress {
					m.autoHistory = append(m.autoHistory, autoStressEntry{
						Workers: m.autoWorkers,
						RPS:     m.snap.CurrentRPS,
						P99:     m.snap.P99,
					})
					m.autoStress = false
				}
				m.screen = screenTimeout
				return m, tickCmd()
			}

			// AutoStress: ramp up workers
			if m.autoStress {
				m.autoWaitTicks++
				// Wait 6 ticks (3s) per level for stabilization
				if m.autoWaitTicks >= 6 && m.snap.TotalRequests > 10 {
					m.autoHistory = append(m.autoHistory, autoStressEntry{
						Workers: m.autoWorkers,
						RPS:     m.snap.CurrentRPS,
						P99:     m.snap.P99,
					})
					m.autoLastRPS = m.snap.CurrentRPS

					// Ramp: increase workers
					m.lt.Stop()
					m.autoWorkers += m.autoStep
					m.concurrency = m.autoWorkers
					m.lt.concurrency = m.autoWorkers
					m.lt.Start()
					m.autoWaitTicks = 0
				}
			}
		}
		return m, tickCmd()
	}

	return m, nil
}

func (m model) View() string {
	var b strings.Builder

	title := titleStyle.Render(" ESC Voting System - Stress Test ")
	b.WriteString(title + "\n\n")

	switch m.screen {
	case screenConfig:
		b.WriteString(m.configView())
	case screenRunning:
		b.WriteString(m.runningView())
	case screenTimeout:
		b.WriteString(m.timeoutView())
	}

	return b.String()
}

func (m model) configView() string {
	var b strings.Builder

	b.WriteString(headerStyle.Render("Target: ") + m.lt.baseURL + "\n\n")

	b.WriteString(headerStyle.Render("Select Endpoint:") + "\n")
	for i, ep := range m.lt.endpoints {
		cursor := "  "
		style := dimStyle
		if i == m.cursor {
			cursor = "> "
			style = selectedStyle
		}
		sel := " "
		if i == m.lt.selected {
			sel = "*"
		}
		b.WriteString(style.Render(fmt.Sprintf("%s[%s] %s %s %s", cursor, sel, ep.Method, ep.Path, ep.Name)) + "\n")
	}

	b.WriteString("\n")

	// Concurrency slider
	idx := m.cursor - len(m.lt.endpoints)
	cursor := "  "
	style := dimStyle
	if idx == 0 {
		cursor = "> "
		style = selectedStyle
	}
	bar := renderSlider(m.concurrency, 1, 1000, 30)
	b.WriteString(style.Render(fmt.Sprintf("%sConcurrency: %d  %s", cursor, m.concurrency, bar)) + "\n")
	if idx == 0 {
		b.WriteString(dimStyle.Render("    [h/l] +/-1  [+/-] +/-10") + "\n")
	}

	b.WriteString("\n")

	// Start button
	startCursor := "  "
	startStyle := dimStyle
	if idx == 1 {
		startCursor = "> "
		startStyle = goodStyle
	}
	b.WriteString(startStyle.Render(startCursor+"[ START ]") + "\n")

	b.WriteString("\n" + dimStyle.Render("[enter] select  [r] run  [a] auto-stress  [q] quit"))

	return b.String()
}

func (m model) runningView() string {
	var b strings.Builder

	ep := m.lt.endpoints[m.lt.selected]
	b.WriteString(headerStyle.Render(fmt.Sprintf("Endpoint: %s %s", ep.Method, ep.Path)))
	b.WriteString(dimStyle.Render(fmt.Sprintf("  |  Workers: %d", m.lt.concurrency)))
	b.WriteString(dimStyle.Render(fmt.Sprintf("  |  Elapsed: %s", m.snap.Elapsed.Truncate(time.Second))))
	if m.autoStress {
		b.WriteString(warnStyle.Render(fmt.Sprintf("  |  AUTO-STRESS (step +%d)", m.autoStep)))
	}
	b.WriteString("\n\n")

	// RPS section
	rpsColor := goodStyle
	if m.snap.CurrentRPS < 100 {
		rpsColor = warnStyle
	}
	b.WriteString(boxStyle.Render(
		headerStyle.Render("Throughput") + "\n" +
			rpsColor.Render(fmt.Sprintf("  Current RPS: %.0f", m.snap.CurrentRPS)) + "\n" +
			goodStyle.Render(fmt.Sprintf("  Peak RPS:    %.0f", m.maxRPS)) + "\n" +
			dimStyle.Render(fmt.Sprintf("  Total Reqs:  %d", m.snap.TotalRequests)),
	))
	b.WriteString("\n\n")

	// RPS sparkline
	b.WriteString(headerStyle.Render("RPS History:") + "\n")
	b.WriteString(renderSparkline(m.rpsHistory, 40) + "\n\n")

	// Latency section
	b.WriteString(boxStyle.Render(
		headerStyle.Render("Latency") + "\n" +
			fmt.Sprintf("  Min: %s  Avg: %s  Max: %s\n",
				colorLatency(m.snap.Min), colorLatency(m.snap.Avg), colorLatency(m.snap.Max)) +
			fmt.Sprintf("  P50: %s  P90: %s  P95: %s  P99: %s",
				colorLatency(m.snap.P50), colorLatency(m.snap.P90),
				colorLatency(m.snap.P95), colorLatency(m.snap.P99)),
	))
	b.WriteString("\n\n")

	// Status codes
	b.WriteString(boxStyle.Render(m.statusCodesView()))
	b.WriteString("\n\n")

	// Error rate
	errRate := float64(0)
	if m.snap.TotalRequests > 0 {
		errRate = float64(m.snap.TotalErrors) / float64(m.snap.TotalRequests) * 100
	}
	errStyle := goodStyle
	if errRate > 1 {
		errStyle = warnStyle
	}
	if errRate > 10 {
		errStyle = badStyle
	}
	b.WriteString(errStyle.Render(fmt.Sprintf("Error Rate: %.1f%% (%d/%d)",
		errRate, m.snap.TotalErrors, m.snap.TotalRequests)))
	b.WriteString("\n\n")

	b.WriteString(dimStyle.Render("[s] stop  [q] quit"))

	return b.String()
}

func (m model) timeoutView() string {
	var b strings.Builder

	b.WriteString(badStyle.Render("⚠  TIMEOUT DETECTED — Test Stopped") + "\n\n")

	ep := m.lt.endpoints[m.lt.selected]
	b.WriteString(headerStyle.Render(fmt.Sprintf("Endpoint: %s %s", ep.Method, ep.Path)) + "\n")
	b.WriteString(fmt.Sprintf("Workers: %d  |  Total Requests: %d  |  Elapsed: %s\n\n",
		m.lt.concurrency, m.snap.TotalRequests, m.snap.Elapsed.Truncate(time.Second)))

	b.WriteString(boxStyle.Render(
		badStyle.Render(fmt.Sprintf("First timeout after: %s", m.snap.TimeoutLatency.Truncate(time.Millisecond)))+"\n"+
			dimStyle.Render(fmt.Sprintf("Client timeout limit: %s", m.lt.client.Timeout)),
	))
	b.WriteString("\n\n")

	// Final latency stats
	b.WriteString(boxStyle.Render(
		headerStyle.Render("Final Latency Stats")+"\n"+
			fmt.Sprintf("  Min: %s  Avg: %s  Max: %s\n",
				colorLatency(m.snap.Min), colorLatency(m.snap.Avg), colorLatency(m.snap.Max))+
			fmt.Sprintf("  P50: %s  P90: %s  P95: %s  P99: %s",
				colorLatency(m.snap.P50), colorLatency(m.snap.P90),
				colorLatency(m.snap.P95), colorLatency(m.snap.P99)),
	))
	b.WriteString("\n\n")

	b.WriteString(boxStyle.Render(m.statusCodesView()))
	b.WriteString("\n\n")

	// AutoStress results table
	if len(m.autoHistory) > 0 {
		b.WriteString(boxStyle.Render(m.autoStressResultsView()))
		b.WriteString("\n\n")
	}

	errRate := float64(0)
	if m.snap.TotalRequests > 0 {
		errRate = float64(m.snap.TotalErrors) / float64(m.snap.TotalRequests) * 100
	}
	errStyle := goodStyle
	if errRate > 1 {
		errStyle = warnStyle
	}
	if errRate > 10 {
		errStyle = badStyle
	}
	b.WriteString(errStyle.Render(fmt.Sprintf("Error Rate: %.1f%% (%d/%d)",
		errRate, m.snap.TotalErrors, m.snap.TotalRequests)))
	b.WriteString("\n\n")

	b.WriteString(dimStyle.Render("[b] back  [q] quit"))

	return b.String()
}

func (m model) autoStressResultsView() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("Auto-Stress Results") + "\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("  %-10s %-12s %-12s", "Workers", "RPS", "P99")) + "\n")

	for _, entry := range m.autoHistory {
		rpsStyle := goodStyle
		if entry.P99 > 200*time.Millisecond {
			rpsStyle = warnStyle
		}
		b.WriteString(fmt.Sprintf("  %-10d %s  %s\n",
			entry.Workers,
			rpsStyle.Render(fmt.Sprintf("%-12.0f", entry.RPS)),
			colorLatency(entry.P99),
		))
	}

	return b.String()
}

func (m model) statusCodesView() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("Status Codes") + "\n")

	if len(m.snap.StatusCodes) == 0 {
		b.WriteString(dimStyle.Render("  waiting..."))
		return b.String()
	}

	codes := make([]int, 0, len(m.snap.StatusCodes))
	for c := range m.snap.StatusCodes {
		codes = append(codes, c)
	}
	sort.Ints(codes)

	var maxCount int64
	for _, c := range m.snap.StatusCodes {
		if c > maxCount {
			maxCount = c
		}
	}

	for _, code := range codes {
		count := m.snap.StatusCodes[code]
		style := goodStyle
		if code >= 400 {
			style = warnStyle
		}
		if code >= 500 {
			style = badStyle
		}

		barLen := 20
		if maxCount > 0 {
			barLen = int(float64(count) / float64(maxCount) * 20)
		}
		if barLen < 1 {
			barLen = 1
		}

		bar := barFull.Render(strings.Repeat("█", barLen)) +
			barEmpty.Render(strings.Repeat("░", 20-barLen))

		b.WriteString(fmt.Sprintf("  %s %s %s\n",
			style.Render(fmt.Sprintf("%d", code)),
			bar,
			dimStyle.Render(fmt.Sprintf("%d", count)),
		))
	}

	return b.String()
}

// ── Rendering helpers ───────────────────────────────────────────────────────

func colorLatency(d time.Duration) string {
	ms := d.Milliseconds()
	s := fmt.Sprintf("%6s", d.Truncate(100*time.Microsecond))
	if ms < 50 {
		return goodStyle.Render(s)
	}
	if ms < 200 {
		return warnStyle.Render(s)
	}
	return badStyle.Render(s)
}

func renderSlider(val, min, max, width int) string {
	ratio := float64(val-min) / float64(max-min)
	filled := int(ratio * float64(width))
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	return barFull.Render(strings.Repeat("█", filled)) +
		barEmpty.Render(strings.Repeat("░", width-filled))
}

func renderSparkline(data []float64, width int) string {
	if len(data) == 0 {
		return dimStyle.Render(strings.Repeat("▁", width))
	}

	// Resample data to fit width
	display := data
	if len(data) > width {
		display = data[len(data)-width:]
	}

	maxVal := 0.0
	for _, v := range display {
		if v > maxVal {
			maxVal = v
		}
	}
	if maxVal == 0 {
		maxVal = 1
	}

	blocks := []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}
	var sb strings.Builder
	for _, v := range display {
		idx := int(math.Min(v/maxVal*7, 7))
		sb.WriteString(goodStyle.Render(blocks[idx]))
	}
	// Pad remaining
	for i := len(display); i < width; i++ {
		sb.WriteString(dimStyle.Render("▁"))
	}

	return sb.String()
}

// ── Console output ─────────────────────────────────────────────────────────

func printFinalStats(m model) {
	ep := m.lt.endpoints[m.lt.selected]

	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Println("  ESC Voting System - Stress Test Results")
	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Printf("  Endpoint:    %s %s\n", ep.Method, ep.Path)
	fmt.Printf("  Workers:     %d\n", m.lt.concurrency)
	fmt.Printf("  Duration:    %s\n", m.snap.Elapsed.Truncate(time.Second))
	fmt.Println("───────────────────────────────────────────────────────")

	fmt.Println("  Throughput")
	fmt.Printf("    Total Requests:  %d\n", m.snap.TotalRequests)
	fmt.Printf("    Peak RPS:        %.0f\n", m.maxRPS)
	fmt.Printf("    Final RPS:       %.0f\n", m.snap.CurrentRPS)
	fmt.Println("───────────────────────────────────────────────────────")

	fmt.Println("  Latency")
	fmt.Printf("    Min:  %s\n", m.snap.Min.Truncate(100*time.Microsecond))
	fmt.Printf("    Avg:  %s\n", m.snap.Avg.Truncate(100*time.Microsecond))
	fmt.Printf("    Max:  %s\n", m.snap.Max.Truncate(100*time.Microsecond))
	fmt.Printf("    P50:  %s\n", m.snap.P50.Truncate(100*time.Microsecond))
	fmt.Printf("    P90:  %s\n", m.snap.P90.Truncate(100*time.Microsecond))
	fmt.Printf("    P95:  %s\n", m.snap.P95.Truncate(100*time.Microsecond))
	fmt.Printf("    P99:  %s\n", m.snap.P99.Truncate(100*time.Microsecond))
	fmt.Println("───────────────────────────────────────────────────────")

	fmt.Println("  Status Codes")
	codes := make([]int, 0, len(m.snap.StatusCodes))
	for c := range m.snap.StatusCodes {
		codes = append(codes, c)
	}
	sort.Ints(codes)
	for _, code := range codes {
		fmt.Printf("    %d: %d\n", code, m.snap.StatusCodes[code])
	}
	fmt.Println("───────────────────────────────────────────────────────")

	errRate := float64(0)
	if m.snap.TotalRequests > 0 {
		errRate = float64(m.snap.TotalErrors) / float64(m.snap.TotalRequests) * 100
	}
	fmt.Printf("  Errors: %d/%d (%.1f%%)\n", m.snap.TotalErrors, m.snap.TotalRequests, errRate)

	if m.snap.TimedOut {
		fmt.Printf("  ⚠ First timeout after: %s\n", m.snap.TimeoutLatency.Truncate(time.Millisecond))
	}

	if len(m.autoHistory) > 0 {
		fmt.Println("───────────────────────────────────────────────────────")
		fmt.Println("  Auto-Stress Results")
		fmt.Printf("    %-10s %-12s %-12s\n", "Workers", "RPS", "P99")
		for _, entry := range m.autoHistory {
			fmt.Printf("    %-10d %-12.0f %s\n", entry.Workers, entry.RPS, entry.P99.Truncate(100*time.Microsecond))
		}
	}

	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Println()
}

// ── Main ────────────────────────────────────────────────────────────────────

func main() {
	baseURL := flag.String("url", "https://localhost", "Base URL of the ESC Voting System")
	concurrency := flag.Int("c", 50, "Initial concurrency (number of workers)")
	flag.Parse()

	p := tea.NewProgram(
		initialModel(*baseURL, *concurrency),
		tea.WithAltScreen(),
	)

	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	m := finalModel.(model)
	if m.snap.TotalRequests > 0 {
		printFinalStats(m)
	}
}
