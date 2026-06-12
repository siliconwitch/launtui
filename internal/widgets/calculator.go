package widgets

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type CalculatorConfig struct {
	Enabled    bool `toml:"enabled"`
	Precision  int  `toml:"precision"`
	MaxHistory int  `toml:"max_history"`
}

func (CalculatorConfig) SectionName() string { return "calculator" }

func DefaultCalculatorConfig() CalculatorConfig {
	return CalculatorConfig{Enabled: true, Precision: 6, MaxHistory: 50}
}

const (
	calculatorHistoryFile = "calculator-history.json"
	currencyCacheFile     = "currency-rates.json"
	currencyRatesURL      = "https://api.frankfurter.dev/v1/latest"
	currencyCacheMaxAge   = 24 * time.Hour
)

var (
	calculatorResultStyle   = lipgloss.NewStyle().Foreground(clockColor).Bold(true)
	calculatorBarStyle      = lipgloss.NewStyle().Foreground(clockColor)
	calculatorSelectedStyle = lipgloss.NewStyle().Foreground(clockColor).Bold(true)
)

type calculation struct {
	Expression string `json:"expression"`
	Answer     string `json:"answer"`
	Time       int64  `json:"time"`
}

type calculatorHistoryMsg []calculation

type currencyRatesMsg struct {
	rates  map[string]float64
	failed bool
}

type Calculator struct {
	cfg         CalculatorConfig
	query       string
	answer      string
	valid       bool
	note        string
	history     []calculation
	rates       map[string]float64
	ratesFailed bool
	cursor      int
}

func NewCalculator(cfg CalculatorConfig) Calculator {
	return Calculator{cfg: cfg}
}

func (Calculator) Name() string    { return "Calc" }
func (Calculator) Hotkey() string  { return "ctrl+c" }
func (c Calculator) Enabled() bool { return c.cfg.Enabled }

func (c Calculator) Init() tea.Cmd {
	if !c.cfg.Enabled {
		return nil
	}

	return tea.Batch(loadCalculatorHistoryCmd(), loadCurrencyRatesCmd())
}

func (c Calculator) Update(msg tea.Msg) (Mode, tea.Cmd) {
	switch msg := msg.(type) {
	case calculatorHistoryMsg:
		c.history = msg

		return c, nil

	case currencyRatesMsg:
		c.rates = msg.rates
		c.ratesFailed = msg.failed

		liveBefore := c.liveCount()
		c.evaluate()

		if c.cursor > 0 {
			c.cursor = max(c.cursor+c.liveCount()-liveBefore, 0)
		}

		return c, nil

	case AppClosingMsg:
		return c, c.recordCalculationCmd()
	}

	return c, nil
}

func (c Calculator) SetQuery(query string) Mode {
	c.query = strings.TrimSpace(query)
	c.cursor = 0
	c.evaluate()

	return c
}

func (c Calculator) HasResults() bool { return c.valid }

func (c Calculator) liveCount() int {
	if c.valid {
		return 1
	}

	return 0
}

func (c Calculator) itemCount() int {
	return c.liveCount() + len(c.history)
}

func (c Calculator) MoveUp() Mode {
	if c.cursor > 0 {
		c.cursor--
	}

	return c
}

func (c Calculator) MoveDown() Mode {
	if c.cursor < c.itemCount()-1 {
		c.cursor++
	}

	return c
}

func (c Calculator) Activate() tea.Cmd {
	answer, ok := c.selectedAnswer()

	if !ok {
		return nil
	}

	return func() tea.Msg {
		copyToClipboard(answer)
		recordClipboardText(answer, 0)

		return RequestQuitMsg{}
	}
}

func (c Calculator) DeleteSelectedHistory() (Mode, tea.Cmd, bool) {
	index := c.cursor - c.liveCount()

	if index < 0 || index >= len(c.history) {
		return c, nil, false
	}

	c.history = append(append([]calculation{}, c.history[:index]...), c.history[index+1:]...)

	if c.cursor >= c.itemCount() {
		c.cursor = max(c.itemCount()-1, 0)
	}

	return c, saveCalculatorHistoryCmd(c.history), true
}

func (c Calculator) ClearHistory() (Mode, tea.Cmd) {
	c.history = nil
	c.cursor = min(c.cursor, max(c.itemCount()-1, 0))

	return c, saveCalculatorHistoryCmd(nil)
}

func saveCalculatorHistoryCmd(history []calculation) tea.Cmd {
	return func() tea.Msg {
		path, err := launtuiDataPath(calculatorHistoryFile)

		if err != nil {
			return nil
		}

		_ = saveJSON(path, history)

		return nil
	}
}

func (c Calculator) selectedAnswer() (string, bool) {
	if c.valid && c.cursor == 0 {
		return c.answer, true
	}

	index := c.cursor - c.liveCount()

	if index >= 0 && index < len(c.history) {
		return c.history[index].Answer, true
	}

	return "", false
}

func (c Calculator) View(width, rows int) string {
	var lines []string

	switch {
	case c.valid:
		lines = append(lines, c.renderLive(width))
	case c.note != "":
		lines = append(lines, subtleStyle.Render(c.note))
	case c.query != "":
		lines = append(lines, subtleStyle.Render("invalid expression"))
	case len(c.history) == 0:
		lines = append(lines, subtleStyle.Render("type an arithmetic expression"))
	}

	historyRows := rows - len(lines)

	if len(c.history) > 0 && historyRows > 0 {
		selected := c.cursor - c.liveCount()
		start, end := visibleRange(max(selected, 0), historyRows, len(c.history))

		for i := start; i < end; i++ {
			lines = append(lines, c.renderHistory(c.history[i], i == selected, width))
		}
	}

	return strings.Join(lines, "\n")
}

func (c Calculator) renderLive(width int) string {
	line := truncate("= "+c.answer, max(width-2, 1))

	if c.cursor == 0 {
		return calculatorBarStyle.Render("▌ ") + calculatorResultStyle.Render(line)
	}

	return "  " + calculatorResultStyle.Render(line)
}

func (c Calculator) renderHistory(entry calculation, selected bool, width int) string {
	line := truncate(entry.Expression+" = "+entry.Answer, max(width-2, 1))

	if selected {
		return calculatorBarStyle.Render("▌ ") + calculatorSelectedStyle.Render(line)
	}

	return "  " + subtleStyle.Render(line)
}

func (c *Calculator) evaluate() {
	c.answer, c.valid, c.note = "", false, ""

	if c.query == "" {
		return
	}

	if value, ok := evalExpression(c.query); ok {
		c.answer = formatNumber(value, c.cfg.Precision)
		c.valid = true

		return
	}

	c.evaluateConversion()
}

var conversionPattern = regexp.MustCompile(`^(.*?)\s*([a-zA-Z°][a-zA-Z0-9/²³°]*)\s+(?:to|in)\s+([a-zA-Z°][a-zA-Z0-9/²³°]*)$`)

func (c *Calculator) evaluateConversion() {
	match := conversionPattern.FindStringSubmatch(c.query)

	if match == nil {
		return
	}

	amountText, fromText, toText := match[1], match[2], match[3]

	amount := 1.0

	if strings.TrimSpace(amountText) != "" {
		value, ok := evalExpression(amountText)

		if !ok {
			return
		}

		amount = value
	}

	if answer, ok := convertUnits(amount, fromText, toText, c.cfg.Precision); ok {
		c.answer, c.valid = answer, true

		return
	}

	c.evaluateCurrency(amount, fromText, toText)
}

var currencyCodePattern = regexp.MustCompile(`^[A-Z]{3}$`)

func (c *Calculator) evaluateCurrency(amount float64, fromText, toText string) {
	from, to := strings.ToUpper(fromText), strings.ToUpper(toText)

	if !currencyCodePattern.MatchString(from) || !currencyCodePattern.MatchString(to) {
		return
	}

	if c.rates == nil {
		if c.ratesFailed {
			c.note = "exchange rates unavailable"
		} else {
			c.note = "fetching exchange rates…"
		}

		return
	}

	fromRate, fromOk := c.rates[from]

	if !fromOk {
		c.note = "unknown currency " + from

		return
	}

	toRate, toOk := c.rates[to]

	if !toOk {
		c.note = "unknown currency " + to

		return
	}

	c.answer = formatNumber(amount/fromRate*toRate, 2) + " " + to
	c.valid = true
}

func (c Calculator) recordCalculationCmd() tea.Cmd {
	entry, ok := c.completedCalculation()

	if !ok {
		return nil
	}

	limit := c.cfg.MaxHistory

	return func() tea.Msg {
		recordCalculation(entry, limit)

		return nil
	}
}

func (c Calculator) completedCalculation() (calculation, bool) {
	if !c.valid {
		return calculation{}, false
	}

	if _, err := strconv.ParseFloat(c.query, 64); err == nil {
		return calculation{}, false
	}

	if len(c.history) > 0 && c.history[0].Expression == c.query && c.history[0].Answer == c.answer {
		return calculation{}, false
	}

	return calculation{Expression: c.query, Answer: c.answer, Time: time.Now().Unix()}, true
}

func recordCalculation(entry calculation, limit int) {
	if limit <= 0 {
		limit = 50
	}

	path, err := launtuiDataPath(calculatorHistoryFile)

	if err != nil {
		return
	}

	previous, _ := loadJSON[[]calculation](path)

	if len(previous) > 0 && previous[0].Expression == entry.Expression && previous[0].Answer == entry.Answer {
		return
	}

	entries := append([]calculation{entry}, previous...)

	if len(entries) > limit {
		entries = entries[:limit]
	}

	_ = saveJSON(path, entries)
}

func loadCalculatorHistoryCmd() tea.Cmd {
	return func() tea.Msg {
		path, err := launtuiDataPath(calculatorHistoryFile)

		if err != nil {
			return calculatorHistoryMsg(nil)
		}

		history, _ := loadJSON[[]calculation](path)

		return calculatorHistoryMsg(history)
	}
}

type currencyCache struct {
	Fetched int64              `json:"fetched"`
	Rates   map[string]float64 `json:"rates"`
}

func loadCurrencyRatesCmd() tea.Cmd {
	return func() tea.Msg {
		cache, cached := readCurrencyCache()

		if cached && time.Now().Unix()-cache.Fetched < int64(currencyCacheMaxAge.Seconds()) {
			return currencyRatesMsg{rates: cache.Rates}
		}

		rates, err := fetchCurrencyRates()

		if err != nil {
			if cached {
				return currencyRatesMsg{rates: cache.Rates}
			}

			return currencyRatesMsg{failed: true}
		}

		writeCurrencyCache(rates)

		return currencyRatesMsg{rates: rates}
	}
}

func readCurrencyCache() (currencyCache, bool) {
	path, err := launtuiCachePath(currencyCacheFile)

	if err != nil {
		return currencyCache{}, false
	}

	cache, ok := loadJSON[currencyCache](path)

	return cache, ok && len(cache.Rates) > 0
}

func writeCurrencyCache(rates map[string]float64) {
	path, err := launtuiCachePath(currencyCacheFile)

	if err != nil {
		return
	}

	_ = saveJSON(path, currencyCache{Fetched: time.Now().Unix(), Rates: rates})
}

func fetchCurrencyRates() (map[string]float64, error) {
	client := http.Client{Timeout: 5 * time.Second}

	response, err := client.Get(currencyRatesURL)

	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, errors.New("unexpected response " + response.Status)
	}

	var payload struct {
		Base  string             `json:"base"`
		Rates map[string]float64 `json:"rates"`
	}

	err = json.NewDecoder(response.Body).Decode(&payload)

	if err != nil {
		return nil, err
	}

	if len(payload.Rates) == 0 {
		return nil, errors.New("empty rates table")
	}

	if payload.Base != "" {
		payload.Rates[payload.Base] = 1
	}

	return payload.Rates, nil
}

type unitDefinition struct {
	category string
	factor   float64
	offset   float64
	label    string
}

var unitDefinitions = map[string]unitDefinition{
	"mm":  {"length", 0.001, 0, "mm"},
	"cm":  {"length", 0.01, 0, "cm"},
	"m":   {"length", 1, 0, "m"},
	"km":  {"length", 1000, 0, "km"},
	"in":  {"length", 0.0254, 0, "in"},
	"ft":  {"length", 0.3048, 0, "ft"},
	"yd":  {"length", 0.9144, 0, "yd"},
	"mi":  {"length", 1609.344, 0, "mi"},
	"nmi": {"length", 1852, 0, "nmi"},

	"mg": {"mass", 1e-6, 0, "mg"},
	"g":  {"mass", 0.001, 0, "g"},
	"kg": {"mass", 1, 0, "kg"},
	"t":  {"mass", 1000, 0, "t"},
	"oz": {"mass", 0.028349523125, 0, "oz"},
	"lb": {"mass", 0.45359237, 0, "lb"},
	"st": {"mass", 6.35029318, 0, "st"},

	"ml":   {"volume", 0.001, 0, "ml"},
	"l":    {"volume", 1, 0, "l"},
	"floz": {"volume", 0.0295735295625, 0, "fl oz"},
	"cup":  {"volume", 0.2365882365, 0, "cups"},
	"pt":   {"volume", 0.473176473, 0, "pt"},
	"gal":  {"volume", 3.785411784, 0, "gal"},

	"c": {"temperature", 1, 273.15, "°C"},
	"f": {"temperature", 5.0 / 9.0, 459.67 * 5.0 / 9.0, "°F"},
	"k": {"temperature", 1, 0, "K"},

	"bit": {"data", 0.125, 0, "bit"},
	"b":   {"data", 1, 0, "B"},
	"kb":  {"data", 1e3, 0, "kB"},
	"mb":  {"data", 1e6, 0, "MB"},
	"gb":  {"data", 1e9, 0, "GB"},
	"tb":  {"data", 1e12, 0, "TB"},
	"kib": {"data", 1 << 10, 0, "KiB"},
	"mib": {"data", 1 << 20, 0, "MiB"},
	"gib": {"data", 1 << 30, 0, "GiB"},
	"tib": {"data", 1 << 40, 0, "TiB"},

	"mps":  {"speed", 1, 0, "m/s"},
	"kmh":  {"speed", 1.0 / 3.6, 0, "km/h"},
	"mph":  {"speed", 0.44704, 0, "mph"},
	"knot": {"speed", 1852.0 / 3600.0, 0, "kn"},

	"sqm":  {"area", 1, 0, "m²"},
	"sqkm": {"area", 1e6, 0, "km²"},
	"sqft": {"area", 0.09290304, 0, "ft²"},
	"sqmi": {"area", 2589988.110336, 0, "mi²"},
	"acre": {"area", 4046.8564224, 0, "acres"},
	"ha":   {"area", 10000, 0, "ha"},

	"ms":   {"time", 0.001, 0, "ms"},
	"s":    {"time", 1, 0, "s"},
	"min":  {"time", 60, 0, "min"},
	"h":    {"time", 3600, 0, "h"},
	"day":  {"time", 86400, 0, "days"},
	"week": {"time", 604800, 0, "weeks"},
	"year": {"time", 31557600, 0, "years"},
}

var unitAliases = map[string]string{
	"millimetres": "mm", "millimeters": "mm", "millimetre": "mm", "millimeter": "mm",
	"centimetres": "cm", "centimeters": "cm", "centimetre": "cm", "centimeter": "cm",
	"metres": "m", "meters": "m", "metre": "m", "meter": "m",
	"kilometres": "km", "kilometers": "km", "kilometre": "km", "kilometer": "km", "kms": "km",
	"inches": "in", "inch": "in",
	"feet": "ft", "foot": "ft",
	"yards": "yd", "yard": "yd",
	"miles": "mi", "mile": "mi",
	"milligrams": "mg", "milligram": "mg",
	"grams": "g", "gram": "g",
	"kilograms": "kg", "kilogram": "kg", "kilos": "kg", "kilo": "kg",
	"tonnes": "t", "tonne": "t", "tons": "t", "ton": "t",
	"ounces": "oz", "ounce": "oz",
	"pounds": "lb", "pound": "lb", "lbs": "lb",
	"stones": "st", "stone": "st",
	"millilitres": "ml", "milliliters": "ml", "millilitre": "ml", "milliliter": "ml",
	"litres": "l", "liters": "l", "litre": "l", "liter": "l",
	"cups":  "cup",
	"pints": "pt", "pint": "pt",
	"gallons": "gal", "gallon": "gal",
	"celsius": "c", "centigrade": "c",
	"fahrenheit": "f",
	"kelvin":     "k",
	"bits":       "bit",
	"bytes":      "b", "byte": "b",
	"kilobytes": "kb", "kilobyte": "kb",
	"megabytes": "mb", "megabyte": "mb",
	"gigabytes": "gb", "gigabyte": "gb",
	"terabytes": "tb", "terabyte": "tb",
	"m/s":  "mps",
	"km/h": "kmh", "kmph": "kmh", "kph": "kmh",
	"knots": "knot", "kn": "knot", "kt": "knot", "kts": "knot",
	"m2": "sqm", "km2": "sqkm", "ft2": "sqft", "mi2": "sqmi",
	"m²": "sqm", "km²": "sqkm", "ft²": "sqft", "mi²": "sqmi",
	"acres":    "acre",
	"hectares": "ha", "hectare": "ha",
	"secs": "s", "sec": "s", "seconds": "s", "second": "s",
	"mins": "min", "minutes": "min", "minute": "min",
	"hr": "h", "hrs": "h", "hours": "h", "hour": "h",
	"days":  "day",
	"weeks": "week", "wk": "week", "wks": "week",
	"years": "year", "yr": "year", "yrs": "year",
}

func resolveUnit(text string) (unitDefinition, bool) {
	key := strings.ToLower(strings.TrimPrefix(text, "°"))

	if canonical, ok := unitAliases[key]; ok {
		key = canonical
	}

	definition, ok := unitDefinitions[key]

	return definition, ok
}

func convertUnits(amount float64, fromText, toText string, precision int) (string, bool) {
	from, fromOk := resolveUnit(fromText)
	to, toOk := resolveUnit(toText)

	if !fromOk || !toOk || from.category != to.category {
		return "", false
	}

	base := amount*from.factor + from.offset
	value := (base - to.offset) / to.factor

	return formatNumber(value, precision) + " " + to.label, true
}

func formatNumber(value float64, precision int) string {
	if precision < 0 {
		precision = 0
	}

	text := strconv.FormatFloat(value, 'f', precision, 64)

	if strings.Contains(text, ".") {
		text = strings.TrimRight(text, "0")
		text = strings.TrimRight(text, ".")
	}

	return text
}

func evalExpression(input string) (float64, bool) {
	parser := &expression{runes: []rune(input)}

	parser.skipSpaces()

	if parser.pos >= len(parser.runes) {
		return 0, false
	}

	value, ok := parser.expr()

	if !ok {
		return 0, false
	}

	parser.skipSpaces()

	if parser.pos != len(parser.runes) {
		return 0, false
	}

	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, false
	}

	return value, true
}

type expression struct {
	runes []rune
	pos   int
}

func (e *expression) skipSpaces() {
	for e.pos < len(e.runes) && e.runes[e.pos] == ' ' {
		e.pos++
	}
}

func (e *expression) peek() (rune, bool) {
	e.skipSpaces()

	if e.pos >= len(e.runes) {
		return 0, false
	}

	return e.runes[e.pos], true
}

func (e *expression) expr() (float64, bool) {
	value, ok := e.term()

	if !ok {
		return 0, false
	}

	for {
		operator, present := e.peek()

		if !present || (operator != '+' && operator != '-') {
			return value, true
		}

		e.pos++

		right, ok := e.term()

		if !ok {
			return 0, false
		}

		if operator == '+' {
			value += right
		} else {
			value -= right
		}
	}
}

func (e *expression) term() (float64, bool) {
	value, ok := e.signed()

	if !ok {
		return 0, false
	}

	for {
		operator, present := e.peek()

		if !present || (operator != '*' && operator != '/' && operator != '%') {
			return value, true
		}

		e.pos++

		right, ok := e.signed()

		if !ok {
			return 0, false
		}

		switch operator {
		case '*':
			value *= right
		case '/':
			if right == 0 {
				return 0, false
			}

			value /= right
		case '%':
			if right == 0 {
				return 0, false
			}

			value = math.Mod(value, right)
		}
	}
}

func (e *expression) signed() (float64, bool) {
	operator, present := e.peek()

	if present && (operator == '+' || operator == '-') {
		e.pos++

		value, ok := e.signed()

		if !ok {
			return 0, false
		}

		if operator == '-' {
			return -value, true
		}

		return value, true
	}

	return e.power()
}

func (e *expression) power() (float64, bool) {
	base, ok := e.primary()

	if !ok {
		return 0, false
	}

	operator, present := e.peek()

	if present && operator == '^' {
		e.pos++

		exponent, ok := e.signed()

		if !ok {
			return 0, false
		}

		return math.Pow(base, exponent), true
	}

	return base, true
}

func (e *expression) primary() (float64, bool) {
	opening, present := e.peek()

	if !present {
		return 0, false
	}

	if opening == '(' {
		e.pos++

		value, ok := e.expr()

		if !ok {
			return 0, false
		}

		closing, present := e.peek()

		if !present || closing != ')' {
			return 0, false
		}

		e.pos++

		return value, true
	}

	return e.number()
}

func (e *expression) number() (float64, bool) {
	e.skipSpaces()

	start := e.pos

	for e.pos < len(e.runes) && e.runes[e.pos] >= '0' && e.runes[e.pos] <= '9' {
		e.pos++
	}

	if e.pos < len(e.runes) && e.runes[e.pos] == '.' {
		e.pos++

		for e.pos < len(e.runes) && e.runes[e.pos] >= '0' && e.runes[e.pos] <= '9' {
			e.pos++
		}
	}

	text := string(e.runes[start:e.pos])

	if text == "" || text == "." {
		return 0, false
	}

	value, err := strconv.ParseFloat(text, 64)

	if err != nil {
		return 0, false
	}

	return value, true
}
