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
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/expr-lang/expr"
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

var calculatorAccent = lipgloss.Color("5")

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
	config      CalculatorConfig
	query       string
	answer      string
	valid       bool
	note        string
	history     []calculation
	rates       map[string]float64
	ratesFailed bool
}

func NewCalculator(config CalculatorConfig) Calculator {
	return Calculator{config: config}
}

func (Calculator) Name() string    { return "Calc" }
func (Calculator) Hotkey() string  { return "ctrl+c" }
func (c Calculator) Enabled() bool { return c.config.Enabled }

func (c Calculator) Init() tea.Cmd {
	if !c.config.Enabled {
		return nil
	}

	return tea.Batch(
		func() tea.Msg {
			path, err := launtuiDataPath(calculatorHistoryFile)

			if err != nil {
				return calculatorHistoryMsg(nil)
			}

			history, _ := loadJSON[[]calculation](path)

			return calculatorHistoryMsg(history)
		},
		func() tea.Msg {
			cachePath, pathErr := launtuiCachePath(currencyCacheFile)

			var cache currencyCache
			cached := false

			if pathErr == nil {
				cache, cached = loadJSON[currencyCache](cachePath)
				cached = cached && len(cache.Rates) > 0
			}

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

			if pathErr == nil {
				_ = saveJSON(cachePath, currencyCache{Fetched: time.Now().Unix(), Rates: rates})
			}

			return currencyRatesMsg{rates: rates}
		},
	)
}

func (c Calculator) Update(msg tea.Msg) (Mode, tea.Cmd) {
	switch msg := msg.(type) {
	case calculatorHistoryMsg:
		c.history = msg

		return c, nil

	case currencyRatesMsg:
		c.rates = msg.rates
		c.ratesFailed = msg.failed
		c.evaluate()

		return c, nil

	case AppClosingMsg:
		if !c.valid {
			return c, nil
		}

		if _, err := strconv.ParseFloat(c.query, 64); err == nil {
			return c, nil
		}

		if len(c.history) > 0 && c.history[0].Expression == c.query && c.history[0].Answer == c.answer {
			return c, nil
		}

		entry := calculation{Expression: c.query, Answer: c.answer, Time: time.Now().Unix()}
		limit := c.config.MaxHistory

		return c, func() tea.Msg {
			if limit <= 0 {
				limit = 50
			}

			path, err := launtuiDataPath(calculatorHistoryFile)

			if err != nil {
				return nil
			}

			previous, _ := loadJSON[[]calculation](path)

			if len(previous) > 0 && previous[0].Expression == entry.Expression && previous[0].Answer == entry.Answer {
				return nil
			}

			_ = saveJSON(path, prependCapped(previous, entry, limit, nil))

			return nil
		}
	}

	return c, nil
}

func (c Calculator) SetQuery(query string) Mode {
	c.query = strings.TrimSpace(query)
	c.evaluate()

	return c
}

func (Calculator) Accent() lipgloss.Color { return calculatorAccent }

func (c Calculator) Status() string {
	switch {
	case c.valid:
		return ""
	case c.note != "":
		return subtleStyle.Render(c.note)
	case c.query != "":
		return subtleStyle.Render("invalid expression")
	case len(c.history) == 0:
		return subtleStyle.Render("type an arithmetic expression")
	}

	return ""
}

func (c Calculator) Rows() []Row {
	var rows []Row

	if c.valid {
		rows = append(rows, Row{left: "= " + c.answer, style: rowEmphasized})
	}

	now := time.Now().Unix()

	for _, entry := range c.history {
		rows = append(rows, Row{
			left:      entry.Expression + " = " + entry.Answer,
			right:     subtleStyle.Render(relativeAge(now - entry.Time)),
			style:     rowDim,
			Deletable: true,
		})
	}

	return rows
}

func (c Calculator) Activate(index int) tea.Cmd {
	live := 0

	if c.valid {
		live = 1
	}

	answer := ""

	if c.valid && index == 0 {
		answer = c.answer
	} else if entry := index - live; entry >= 0 && entry < len(c.history) {
		answer = c.history[entry].Answer
	} else {
		return nil
	}

	return func() tea.Msg {
		copyToClipboard(answer)
		recordClipboardText(answer, 0)

		return RequestQuitMsg{}
	}
}

func (c Calculator) RecallText(index int) (string, bool) {
	live := 0

	if c.valid {
		live = 1
	}

	entry := index - live

	if entry < 0 || entry >= len(c.history) {
		return "", false
	}

	return c.history[entry].Expression, true
}

func (c Calculator) DeleteRow(index int) (Mode, tea.Cmd) {
	live := 0

	if c.valid {
		live = 1
	}

	entry := index - live

	if entry < 0 || entry >= len(c.history) {
		return c, nil
	}

	c.history = removeAt(c.history, entry)

	return c, saveCalculatorHistoryCmd(c.history)
}

func (c Calculator) ClearRows() (Mode, tea.Cmd) {
	c.history = nil

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

func (c *Calculator) evaluate() {
	c.answer, c.valid, c.note = "", false, ""

	if c.query == "" {
		return
	}

	if value, ok := evalExpression(c.query); ok {
		c.answer = formatNumber(value, c.config.Precision)
		c.valid = true

		return
	}

	if answer, ok := convertBase(c.query); ok {
		c.answer = answer
		c.valid = true

		return
	}

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

	if answer, ok := convertDecibel(amount, fromText, toText, c.config.Precision); ok {
		c.answer = answer
		c.valid = true

		return
	}

	if from, fromOk := resolveUnit(fromText); fromOk {
		if to, toOk := resolveUnit(toText); toOk && from.category == to.category {
			base := amount*from.factor + from.offset
			value := (base - to.offset) / to.factor

			c.answer = formatNumber(value, c.config.Precision) + " " + to.label
			c.valid = true

			return
		}
	}

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

var conversionPattern = regexp.MustCompile(`^(.*?)\s*([a-zA-Z°µΩω][a-zA-Z0-9/²³°µΩω]*)\s+(?:to|in)\s+([a-zA-Z°µΩω][a-zA-Z0-9/²³°µΩω]*)$`)

var currencyCodePattern = regexp.MustCompile(`^[A-Z]{3}$`)

type currencyCache struct {
	Fetched int64              `json:"fetched"`
	Rates   map[string]float64 `json:"rates"`
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
	"nm":  {"length", 1e-9, 0, "nm"},
	"um":  {"length", 1e-6, 0, "µm"},
	"mil": {"length", 2.54e-5, 0, "mil"},
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

	"lps": {"flow", 1, 0, "L/s"},
	"lpm": {"flow", 1.0 / 60, 0, "L/min"},
	"lph": {"flow", 1.0 / 3600, 0, "L/h"},
	"m3s": {"flow", 1000, 0, "m³/s"},
	"m3h": {"flow", 1000.0 / 3600, 0, "m³/h"},
	"gpm": {"flow", 3.785411784 / 60, 0, "gpm"},
	"cfm": {"flow", 28.316846592 / 60, 0, "cfm"},
	"cfs": {"flow", 28.316846592, 0, "cfs"},

	"N":   {"force", 1, 0, "N"},
	"mN":  {"force", 1e-3, 0, "mN"},
	"kN":  {"force", 1e3, 0, "kN"},
	"MN":  {"force", 1e6, 0, "MN"},
	"dyn": {"force", 1e-5, 0, "dyn"},
	"lbf": {"force", 4.4482216152605, 0, "lbf"},
	"kgf": {"force", 9.80665, 0, "kgf"},
	"ozf": {"force", 0.27801385095378, 0, "ozf"},

	"Pa":   {"pressure", 1, 0, "Pa"},
	"hPa":  {"pressure", 100, 0, "hPa"},
	"kPa":  {"pressure", 1e3, 0, "kPa"},
	"MPa":  {"pressure", 1e6, 0, "MPa"},
	"bar":  {"pressure", 1e5, 0, "bar"},
	"mbar": {"pressure", 100, 0, "mbar"},
	"atm":  {"pressure", 101325, 0, "atm"},
	"psi":  {"pressure", 6894.757293168, 0, "psi"},
	"torr": {"pressure", 101325.0 / 760, 0, "torr"},
	"mmHg": {"pressure", 133.322387415, 0, "mmHg"},
	"inHg": {"pressure", 3386.389, 0, "inHg"},

	"Nm":   {"torque", 1, 0, "N·m"},
	"mNm":  {"torque", 1e-3, 0, "mN·m"},
	"kNm":  {"torque", 1e3, 0, "kN·m"},
	"lbft": {"torque", 1.3558179483314004, 0, "lbf·ft"},
	"lbin": {"torque", 0.1129848290276167, 0, "lbf·in"},
	"kgfm": {"torque", 9.80665, 0, "kgf·m"},
	"ozin": {"torque", 0.0070615518142260, 0, "ozf·in"},

	"c": {"temperature", 1, 273.15, "°C"},
	"f": {"temperature", 5.0 / 9.0, 459.67 * 5.0 / 9.0, "°F"},
	"k": {"temperature", 1, 0, "K"},

	"bit": {"data", 0.125, 0, "bit"},
	"b":   {"data", 0.125, 0, "b"},
	"B":   {"data", 1, 0, "B"},
	"kb":  {"data", 125, 0, "kb"},
	"kB":  {"data", 1e3, 0, "kB"},
	"Mb":  {"data", 125e3, 0, "Mb"},
	"MB":  {"data", 1e6, 0, "MB"},
	"Gb":  {"data", 125e6, 0, "Gb"},
	"GB":  {"data", 1e9, 0, "GB"},
	"Tb":  {"data", 125e9, 0, "Tb"},
	"TB":  {"data", 1e12, 0, "TB"},
	"Kib": {"data", 128, 0, "Kib"},
	"KiB": {"data", 1 << 10, 0, "KiB"},
	"Mib": {"data", 131072, 0, "Mib"},
	"MiB": {"data", 1 << 20, 0, "MiB"},
	"Gib": {"data", 134217728, 0, "Gib"},
	"GiB": {"data", 1 << 30, 0, "GiB"},
	"Tib": {"data", 137438953472, 0, "Tib"},
	"TiB": {"data", 1 << 40, 0, "TiB"},

	"mps":        {"speed", 1, 0, "m/s"},
	"kmh":        {"speed", 1.0 / 3.6, 0, "km/h"},
	"mph":        {"speed", 0.44704, 0, "mph"},
	"knot":       {"speed", 1852.0 / 3600.0, 0, "kn"},
	"lightspeed": {"speed", 299792458, 0, "c"},

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

	"rad":    {"angle", 1, 0, "rad"},
	"deg":    {"angle", math.Pi / 180, 0, "°"},
	"grad":   {"angle", math.Pi / 200, 0, "grad"},
	"arcmin": {"angle", math.Pi / 180 / 60, 0, "′"},
	"arcsec": {"angle", math.Pi / 180 / 3600, 0, "″"},
	"turn":   {"angle", 2 * math.Pi, 0, "turn"},

	"Hz":  {"frequency", 1, 0, "Hz"},
	"kHz": {"frequency", 1e3, 0, "kHz"},
	"MHz": {"frequency", 1e6, 0, "MHz"},
	"GHz": {"frequency", 1e9, 0, "GHz"},
	"THz": {"frequency", 1e12, 0, "THz"},

	"milliohm": {"resistance", 1e-3, 0, "mΩ"},
	"ohm":      {"resistance", 1, 0, "Ω"},
	"kohm":     {"resistance", 1e3, 0, "kΩ"},
	"megohm":   {"resistance", 1e6, 0, "MΩ"},

	"pF":    {"capacitance", 1e-12, 0, "pF"},
	"nF":    {"capacitance", 1e-9, 0, "nF"},
	"uF":    {"capacitance", 1e-6, 0, "µF"},
	"mF":    {"capacitance", 1e-3, 0, "mF"},
	"farad": {"capacitance", 1, 0, "F"},

	"nH":    {"inductance", 1e-9, 0, "nH"},
	"uH":    {"inductance", 1e-6, 0, "µH"},
	"mH":    {"inductance", 1e-3, 0, "mH"},
	"henry": {"inductance", 1, 0, "H"},

	"uV": {"voltage", 1e-6, 0, "µV"},
	"mV": {"voltage", 1e-3, 0, "mV"},
	"V":  {"voltage", 1, 0, "V"},
	"kV": {"voltage", 1e3, 0, "kV"},
	"MV": {"voltage", 1e6, 0, "MV"},

	"uA": {"current", 1e-6, 0, "µA"},
	"mA": {"current", 1e-3, 0, "mA"},
	"A":  {"current", 1, 0, "A"},
	"kA": {"current", 1e3, 0, "kA"},
	"MA": {"current", 1e6, 0, "MA"},

	"mW": {"power", 1e-3, 0, "mW"},
	"W":  {"power", 1, 0, "W"},
	"kW": {"power", 1e3, 0, "kW"},
	"MW": {"power", 1e6, 0, "MW"},
	"GW": {"power", 1e9, 0, "GW"},
	"TW": {"power", 1e12, 0, "TW"},
	"hp": {"power", 745.6998715822702, 0, "hp"},
}

var unitAliases = map[string]string{
	"nanometre":     "nm",
	"nanometer":     "nm",
	"micron":        "um",
	"micrometre":    "um",
	"micrometer":    "um",
	"µm":            "um",
	"thou":          "mil",
	"millimetre":    "mm",
	"millimeter":    "mm",
	"centimetre":    "cm",
	"centimeter":    "cm",
	"metre":         "m",
	"meter":         "m",
	"kilometre":     "km",
	"kilometer":     "km",
	"inch":          "in",
	"inches":        "in",
	"foot":          "ft",
	"feet":          "ft",
	"yard":          "yd",
	"mile":          "mi",
	"milligram":     "mg",
	"gram":          "g",
	"kilogram":      "kg",
	"kilo":          "kg",
	"tonne":         "t",
	"ton":           "t",
	"ounce":         "oz",
	"pound":         "lb",
	"stone":         "st",
	"millilitre":    "ml",
	"milliliter":    "ml",
	"litre":         "l",
	"liter":         "l",
	"pint":          "pt",
	"gallon":        "gal",
	"l/s":           "lps",
	"l/min":         "lpm",
	"l/h":           "lph",
	"m3/s":          "m3s",
	"m³/s":          "m3s",
	"m3/h":          "m3h",
	"m³/h":          "m3h",
	"newton":        "N",
	"millinewton":   "mN",
	"kilonewton":    "kN",
	"meganewton":    "MN",
	"dyne":          "dyn",
	"poundforce":    "lbf",
	"kilogramforce": "kgf",
	"pascal":        "Pa",
	"hectopascal":   "hPa",
	"kilopascal":    "kPa",
	"megapascal":    "MPa",
	"atmosphere":    "atm",
	"millibar":      "mbar",
	"newtonmetre":   "Nm",
	"newtonmeter":   "Nm",
	"poundfoot":     "lbft",
	"poundinch":     "lbin",
	"celsius":       "c",
	"centigrade":    "c",
	"fahrenheit":    "f",
	"kelvin":        "k",
	"byte":          "B",
	"kilobyte":      "kB",
	"megabyte":      "MB",
	"gigabyte":      "GB",
	"terabyte":      "TB",
	"kilobit":       "kb",
	"megabit":       "Mb",
	"gigabit":       "Gb",
	"terabit":       "Tb",
	"kibibyte":      "KiB",
	"mebibyte":      "MiB",
	"gibibyte":      "GiB",
	"tebibyte":      "TiB",
	"kibibit":       "Kib",
	"mebibit":       "Mib",
	"gibibit":       "Gib",
	"tebibit":       "Tib",
	"m/s":           "mps",
	"km/h":          "kmh",
	"kmph":          "kmh",
	"kph":           "kmh",
	"kn":            "knot",
	"kt":            "knot",
	"m2":            "sqm",
	"km2":           "sqkm",
	"ft2":           "sqft",
	"mi2":           "sqmi",
	"m²":            "sqm",
	"km²":           "sqkm",
	"ft²":           "sqft",
	"mi²":           "sqmi",
	"hectare":       "ha",
	"sec":           "s",
	"second":        "s",
	"minute":        "min",
	"hr":            "h",
	"hour":          "h",
	"wk":            "week",
	"yr":            "year",
	"radian":        "rad",
	"degree":        "deg",
	"gradian":       "grad",
	"gon":           "grad",
	"arcminute":     "arcmin",
	"arcsecond":     "arcsec",
	"rev":           "turn",
	"revolution":    "turn",
	"hertz":         "Hz",
	"kilohertz":     "kHz",
	"megahertz":     "MHz",
	"gigahertz":     "GHz",
	"terahertz":     "THz",
	"Ω":             "ohm",
	"ω":             "ohm",
	"mΩ":            "milliohm",
	"mω":            "milliohm",
	"kΩ":            "kohm",
	"kω":            "kohm",
	"kiloohm":       "kohm",
	"kilohm":        "kohm",
	"MΩ":            "megohm",
	"megaohm":       "megohm",
	"millifarad":    "mF",
	"microfarad":    "uF",
	"µf":            "uF",
	"µF":            "uF",
	"nanofarad":     "nF",
	"picofarad":     "pF",
	"H":             "henry",
	"henries":       "henry",
	"millihenry":    "mH",
	"microhenry":    "uH",
	"µh":            "uH",
	"µH":            "uH",
	"nanohenry":     "nH",
	"volt":          "V",
	"millivolt":     "mV",
	"microvolt":     "uV",
	"µv":            "uV",
	"µV":            "uV",
	"kilovolt":      "kV",
	"megavolt":      "MV",
	"amp":           "A",
	"ampere":        "A",
	"milliamp":      "mA",
	"milliampere":   "mA",
	"microamp":      "uA",
	"µa":            "uA",
	"µA":            "uA",
	"kiloamp":       "kA",
	"megaamp":       "MA",
	"watt":          "W",
	"milliwatt":     "mW",
	"kilowatt":      "kW",
	"megawatt":      "MW",
	"gigawatt":      "GW",
	"terawatt":      "TW",
	"horsepower":    "hp",
}

func resolveUnit(text string) (unitDefinition, bool) {
	token := strings.TrimPrefix(text, "°")

	if definition, ok := lookupUnit(token); ok {
		return definition, true
	}

	if singular := strings.TrimSuffix(token, "s"); singular != token {
		if definition, ok := lookupUnit(singular); ok {
			return definition, true
		}
	}

	if canonical, ok := unitFallback[strings.ToLower(token)]; ok {
		return unitDefinitions[canonical], true
	}

	return unitDefinition{}, false
}

func lookupUnit(token string) (unitDefinition, bool) {
	if canonical, ok := unitAliases[token]; ok {
		token = canonical
	}

	definition, ok := unitDefinitions[token]

	return definition, ok
}

var unitFallback = func() map[string]string {
	fallback := map[string]string{}

	for key := range unitDefinitions {
		fallback[strings.ToLower(key)] = key
	}

	for alias, canonical := range unitAliases {
		lower := strings.ToLower(alias)

		if _, exists := fallback[lower]; !exists {
			fallback[lower] = canonical
		}
	}

	for lower, canonical := range map[string]string{
		"mv":  "mV",
		"ma":  "mA",
		"mw":  "mW",
		"mn":  "mN",
		"nm":  "nm",
		"mb":  "MB",
		"gb":  "GB",
		"tb":  "TB",
		"kb":  "kB",
		"kib": "KiB",
		"mib": "MiB",
		"gib": "GiB",
		"tib": "TiB",
		"mω":  "milliohm",
	} {
		fallback[lower] = canonical
	}

	return fallback
}()

type decibelUnit struct {
	domain    string
	reference float64
	scale     float64
	label     string
}

var decibelUnits = map[string]decibelUnit{
	"dBV":  {"voltage", 1, 20, "dBV"},
	"dBmV": {"voltage", 1e-3, 20, "dBmV"},
	"dBuV": {"voltage", 1e-6, 20, "dBµV"},
	"dBµV": {"voltage", 1e-6, 20, "dBµV"},
	"dBm":  {"power", 1e-3, 10, "dBm"},
	"dBW":  {"power", 1, 10, "dBW"},
}

func lookupDecibel(text string) (decibelUnit, bool) {
	if unit, ok := decibelUnits[text]; ok {
		return unit, true
	}

	lower := strings.ToLower(text)

	for key, unit := range decibelUnits {
		if strings.ToLower(key) == lower {
			return unit, true
		}
	}

	return decibelUnit{}, false
}

func convertDecibel(amount float64, fromText, toText string, precision int) (string, bool) {
	fromDecibel, fromIsDecibel := lookupDecibel(fromText)
	toDecibel, toIsDecibel := lookupDecibel(toText)

	if !fromIsDecibel && !toIsDecibel {
		return "", false
	}

	if fromIsDecibel && toIsDecibel {
		if fromDecibel.domain != toDecibel.domain {
			return "", false
		}

		base := fromDecibel.reference * math.Pow(10, amount/fromDecibel.scale)
		result := toDecibel.scale * math.Log10(base/toDecibel.reference)

		return formatNumber(result, precision) + " " + toDecibel.label, true
	}

	if fromIsDecibel {
		unit, ok := resolveUnit(toText)

		if !ok || unit.category != fromDecibel.domain || unit.offset != 0 {
			return "", false
		}

		base := fromDecibel.reference * math.Pow(10, amount/fromDecibel.scale)

		return formatNumber(base/unit.factor, precision) + " " + unit.label, true
	}

	unit, ok := resolveUnit(fromText)

	if !ok || unit.category != toDecibel.domain || unit.offset != 0 {
		return "", false
	}

	base := amount * unit.factor

	if base <= 0 {
		return "", false
	}

	result := toDecibel.scale * math.Log10(base/toDecibel.reference)

	return formatNumber(result, precision) + " " + toDecibel.label, true
}

var baseConversionPattern = regexp.MustCompile(`(?i)^\s*(.+?)\s+(?:to|in)\s+(dec|decimal|hex|hexadecimal|bin|binary|oct|octal|ascii|char|rune)\s*$`)

func convertBase(query string) (string, bool) {
	match := baseConversionPattern.FindStringSubmatch(query)

	if match == nil {
		return "", false
	}

	value, ok := parseInteger(strings.TrimSpace(match[1]))

	if !ok {
		return "", false
	}

	switch strings.ToLower(match[2]) {
	case "dec", "decimal":
		return strconv.FormatInt(value, 10), true
	case "hex", "hexadecimal":
		return formatRadix(value, 16, "0x"), true
	case "bin", "binary":
		return formatRadix(value, 2, "0b"), true
	case "oct", "octal":
		return formatRadix(value, 8, "0o"), true
	case "ascii", "char", "rune":
		if value < 0 || value > 0x10FFFF || !utf8.ValidRune(rune(value)) {
			return "", false
		}

		return string(rune(value)), true
	}

	return "", false
}

func formatRadix(value int64, radix int, prefix string) string {
	if value < 0 {
		return "-" + prefix + strconv.FormatInt(-value, radix)
	}

	return prefix + strconv.FormatInt(value, radix)
}

func parseInteger(text string) (int64, bool) {
	runes := []rune(text)

	if len(runes) == 3 && (runes[0] == '\'' || runes[0] == '"') && runes[2] == runes[0] {
		return int64(runes[1]), true
	}

	body := text
	negative := strings.HasPrefix(body, "-")

	if negative || strings.HasPrefix(body, "+") {
		body = body[1:]
	}

	radix := 10

	switch {
	case strings.HasPrefix(strings.ToLower(body), "0x"):
		radix, body = 16, body[2:]
	case strings.HasPrefix(strings.ToLower(body), "0b"):
		radix, body = 2, body[2:]
	case strings.HasPrefix(strings.ToLower(body), "0o"):
		radix, body = 8, body[2:]
	case strings.HasPrefix(strings.ToLower(body), "0d"):
		radix, body = 10, body[2:]
	}

	if value, err := strconv.ParseInt(body, radix, 64); err == nil {
		if negative {
			value = -value
		}

		return value, true
	}

	if value, ok := evalExpression(text); ok && value == math.Trunc(value) && math.Abs(value) < 9.2e18 {
		return int64(value), true
	}

	return 0, false
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
	trimmed := strings.TrimSpace(input)

	if trimmed == "" {
		return 0, false
	}

	source := trimmed

	if strings.ContainsAny(trimmed, "&|~") || strings.Contains(trimmed, "<<") || strings.Contains(trimmed, ">>") || containsWord(trimmed, "xor") {
		source = rewriteBitwise(trimmed)
	}

	return runExpression(source)
}

func runExpression(source string) (value float64, ok bool) {
	defer func() {
		if recover() != nil {
			value, ok = 0, false
		}
	}()

	program, err := expr.Compile(source, calculatorExprOptions...)

	if err != nil {
		return 0, false
	}

	result, err := expr.Run(program, calculatorExprEnv)

	if err != nil {
		return 0, false
	}

	switch number := result.(type) {
	case float64:
		if math.IsNaN(number) || math.IsInf(number, 0) {
			return 0, false
		}

		return number, true
	case int:
		return float64(number), true
	case int64:
		return float64(number), true
	}

	return 0, false
}

var calculatorExprEnv = map[string]any{
	"pi":  math.Pi,
	"e":   math.E,
	"tau": 2 * math.Pi,
	"phi": math.Phi,
}

var calculatorExprOptions = []expr.Option{
	expr.Env(calculatorExprEnv),
	expr.Function("sqrt", func(p ...any) (any, error) { return math.Sqrt(toFloatArg(p[0])), nil }),
	expr.Function("cbrt", func(p ...any) (any, error) { return math.Cbrt(toFloatArg(p[0])), nil }),
	expr.Function("abs", func(p ...any) (any, error) { return math.Abs(toFloatArg(p[0])), nil }),
	expr.Function("exp", func(p ...any) (any, error) { return math.Exp(toFloatArg(p[0])), nil }),
	expr.Function("ln", func(p ...any) (any, error) { return math.Log(toFloatArg(p[0])), nil }),
	expr.Function("log", func(p ...any) (any, error) { return math.Log10(toFloatArg(p[0])), nil }),
	expr.Function("log2", func(p ...any) (any, error) { return math.Log2(toFloatArg(p[0])), nil }),
	expr.Function("sin", func(p ...any) (any, error) { return math.Sin(toFloatArg(p[0])), nil }),
	expr.Function("cos", func(p ...any) (any, error) { return math.Cos(toFloatArg(p[0])), nil }),
	expr.Function("tan", func(p ...any) (any, error) { return math.Tan(toFloatArg(p[0])), nil }),
	expr.Function("asin", func(p ...any) (any, error) { return math.Asin(toFloatArg(p[0])), nil }),
	expr.Function("acos", func(p ...any) (any, error) { return math.Acos(toFloatArg(p[0])), nil }),
	expr.Function("atan", func(p ...any) (any, error) { return math.Atan(toFloatArg(p[0])), nil }),
	expr.Function("sinh", func(p ...any) (any, error) { return math.Sinh(toFloatArg(p[0])), nil }),
	expr.Function("cosh", func(p ...any) (any, error) { return math.Cosh(toFloatArg(p[0])), nil }),
	expr.Function("tanh", func(p ...any) (any, error) { return math.Tanh(toFloatArg(p[0])), nil }),
	expr.Function("floor", func(p ...any) (any, error) { return math.Floor(toFloatArg(p[0])), nil }),
	expr.Function("ceil", func(p ...any) (any, error) { return math.Ceil(toFloatArg(p[0])), nil }),
	expr.Function("round", func(p ...any) (any, error) { return math.Round(toFloatArg(p[0])), nil }),
	expr.Function("factorial", func(p ...any) (any, error) {
		operand := toFloatArg(p[0])

		if operand < 0 || operand != math.Trunc(operand) {
			return math.Gamma(operand + 1), nil
		}

		result := 1.0

		for factor := 2.0; factor <= operand; factor++ {
			result *= factor
		}

		return result, nil
	}),
	expr.Function("bitand", func(p ...any) (any, error) { return float64(toIntArg(p[0]) & toIntArg(p[1])), nil }),
	expr.Function("bitor", func(p ...any) (any, error) { return float64(toIntArg(p[0]) | toIntArg(p[1])), nil }),
	expr.Function("bitxor", func(p ...any) (any, error) { return float64(toIntArg(p[0]) ^ toIntArg(p[1])), nil }),
	expr.Function("bitnot", func(p ...any) (any, error) { return float64(^toIntArg(p[0])), nil }),
	expr.Function("shl", func(p ...any) (any, error) { return float64(toIntArg(p[0]) << uint64(toIntArg(p[1]))), nil }),
	expr.Function("shr", func(p ...any) (any, error) { return float64(toIntArg(p[0]) >> uint64(toIntArg(p[1]))), nil }),
}

func toFloatArg(value any) float64 {
	switch number := value.(type) {
	case float64:
		return number
	case int:
		return float64(number)
	case int64:
		return float64(number)
	}

	return math.NaN()
}

func toIntArg(value any) int64 {
	return int64(toFloatArg(value))
}

func containsWord(text, word string) bool {
	for index := strings.Index(text, word); index >= 0; {
		before := index == 0 || !isIdentPart(rune(text[index-1]))
		after := index+len(word) >= len(text) || !isIdentPart(rune(text[index+len(word)]))

		if before && after {
			return true
		}

		next := strings.Index(text[index+1:], word)

		if next < 0 {
			break
		}

		index += next + 1
	}

	return false
}

type bitwiseRewriter struct {
	source   []rune
	position int
}

func rewriteBitwise(input string) string {
	rewriter := &bitwiseRewriter{source: []rune(input)}
	output := rewriter.parseOr()

	if rewriter.position < len(rewriter.source) {
		output += string(rewriter.source[rewriter.position:])
	}

	return output
}

func (r *bitwiseRewriter) parseOr() string {
	left := r.parseXor()

	for r.consumeSingle('|') {
		left = "bitor(" + left + "," + r.parseXor() + ")"
	}

	return left
}

func (r *bitwiseRewriter) parseXor() string {
	left := r.parseAnd()

	for r.consumeWord("xor") {
		left = "bitxor(" + left + "," + r.parseAnd() + ")"
	}

	return left
}

func (r *bitwiseRewriter) parseAnd() string {
	left := r.parseShift()

	for r.consumeSingle('&') {
		left = "bitand(" + left + "," + r.parseShift() + ")"
	}

	return left
}

func (r *bitwiseRewriter) parseShift() string {
	left := r.parseArithmetic()

	for {
		if r.consumeDouble('<') {
			left = "shl(" + left + "," + r.parseArithmetic() + ")"
		} else if r.consumeDouble('>') {
			left = "shr(" + left + "," + r.parseArithmetic() + ")"
		} else {
			return left
		}
	}
}

func (r *bitwiseRewriter) parseArithmetic() string {
	var builder strings.Builder

	for r.position < len(r.source) {
		char := r.source[r.position]

		if char == ')' || char == ',' || r.atBitwiseBoundary() {
			break
		}

		switch {
		case char == '~':
			r.position++
			builder.WriteString("bitnot(" + r.parseAtom() + ")")
		case char == '(':
			r.position++
			builder.WriteString("(" + r.parseArguments() + ")")
			r.consume(')')
		case char == '&' && r.peekNext() == '&':
			builder.WriteString("&&")
			r.position += 2
		case char == '|' && r.peekNext() == '|':
			builder.WriteString("||")
			r.position += 2
		case isDigit(char):
			builder.WriteString(r.readNumber())
		case isIdentStart(char):
			builder.WriteString(r.parseIdentifier())
		default:
			builder.WriteRune(char)
			r.position++
		}
	}

	return builder.String()
}

func (r *bitwiseRewriter) parseIdentifier() string {
	name := r.readIdent()
	mark := r.position

	r.skipSpaces()

	if r.position < len(r.source) && r.source[r.position] == '(' {
		r.position++
		call := name + "(" + r.parseArguments() + ")"
		r.consume(')')

		return call
	}

	r.position = mark

	return name
}

func (r *bitwiseRewriter) parseAtom() string {
	r.skipSpaces()

	if r.position >= len(r.source) {
		return ""
	}

	char := r.source[r.position]

	switch {
	case char == '~':
		r.position++
		return "bitnot(" + r.parseAtom() + ")"
	case char == '(':
		r.position++
		group := "(" + r.parseArguments() + ")"
		r.consume(')')

		return group
	case isIdentStart(char):
		return r.parseIdentifier()
	default:
		return r.readNumber()
	}
}

func (r *bitwiseRewriter) parseArguments() string {
	parts := []string{r.parseOr()}

	for r.consume(',') {
		parts = append(parts, r.parseOr())
	}

	return strings.Join(parts, ",")
}

func (r *bitwiseRewriter) atBitwiseBoundary() bool {
	switch r.source[r.position] {
	case '|':
		return r.peekNext() != '|'
	case '&':
		return r.peekNext() != '&'
	case '<':
		return r.peekNext() == '<'
	case '>':
		return r.peekNext() == '>'
	}

	return r.atWord("xor")
}

func (r *bitwiseRewriter) atWord(word string) bool {
	runes := []rune(word)

	if r.position+len(runes) > len(r.source) {
		return false
	}

	for offset, expected := range runes {
		if r.source[r.position+offset] != expected {
			return false
		}
	}

	after := r.position + len(runes)

	return after >= len(r.source) || !isIdentPart(r.source[after])
}

func (r *bitwiseRewriter) consume(char rune) bool {
	r.skipSpaces()

	if r.position < len(r.source) && r.source[r.position] == char {
		r.position++
		return true
	}

	return false
}

func (r *bitwiseRewriter) consumeSingle(char rune) bool {
	r.skipSpaces()

	if r.position < len(r.source) && r.source[r.position] == char && r.peekNext() != char {
		r.position++
		return true
	}

	return false
}

func (r *bitwiseRewriter) consumeDouble(char rune) bool {
	r.skipSpaces()

	if r.position+1 < len(r.source) && r.source[r.position] == char && r.source[r.position+1] == char {
		r.position += 2
		return true
	}

	return false
}

func (r *bitwiseRewriter) consumeWord(word string) bool {
	r.skipSpaces()

	if r.atWord(word) {
		r.position += len([]rune(word))
		return true
	}

	return false
}

func (r *bitwiseRewriter) skipSpaces() {
	for r.position < len(r.source) && (r.source[r.position] == ' ' || r.source[r.position] == '\t') {
		r.position++
	}
}

func (r *bitwiseRewriter) peekNext() rune {
	if r.position+1 < len(r.source) {
		return r.source[r.position+1]
	}

	return 0
}

func (r *bitwiseRewriter) readIdent() string {
	start := r.position

	for r.position < len(r.source) && isIdentPart(r.source[r.position]) {
		r.position++
	}

	return string(r.source[start:r.position])
}

func (r *bitwiseRewriter) readNumber() string {
	start := r.position

	if r.source[r.position] == '0' && r.position+1 < len(r.source) {
		switch r.source[r.position+1] {
		case 'x', 'X', 'b', 'B', 'o', 'O':
			r.position += 2

			for r.position < len(r.source) && isHexDigit(r.source[r.position]) {
				r.position++
			}

			return string(r.source[start:r.position])
		}
	}

	for r.position < len(r.source) && (isDigit(r.source[r.position]) || r.source[r.position] == '.') {
		r.position++
	}

	if r.position < len(r.source) && (r.source[r.position] == 'e' || r.source[r.position] == 'E') {
		r.position++

		if r.position < len(r.source) && (r.source[r.position] == '+' || r.source[r.position] == '-') {
			r.position++
		}

		for r.position < len(r.source) && isDigit(r.source[r.position]) {
			r.position++
		}
	}

	if r.position == start {
		r.position++
	}

	return string(r.source[start:r.position])
}

func isIdentStart(char rune) bool {
	return char == '_' || (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z')
}

func isIdentPart(char rune) bool {
	return isIdentStart(char) || isDigit(char)
}

func isDigit(char rune) bool {
	return char >= '0' && char <= '9'
}

func isHexDigit(char rune) bool {
	return isDigit(char) || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')
}
