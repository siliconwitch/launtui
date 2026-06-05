package widgets

import (
	"math"
	"os/exec"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type CalculatorConfig struct {
	Enabled   bool `toml:"enabled"`
	Precision int  `toml:"precision"`
}

func (CalculatorConfig) SectionName() string { return "calculator" }

func DefaultCalculatorConfig() CalculatorConfig {
	return CalculatorConfig{Enabled: true, Precision: 6}
}

var calculatorResultStyle = lipgloss.NewStyle().Foreground(clockColor).Bold(true)

type Calculator struct {
	cfg   CalculatorConfig
	query string
	value float64
	valid bool
}

func NewCalculator(cfg CalculatorConfig) Calculator {
	return Calculator{cfg: cfg}
}

func (Calculator) Name() string    { return "Calc" }
func (Calculator) Hotkey() string  { return "ctrl+c" }
func (c Calculator) Enabled() bool { return c.cfg.Enabled }
func (Calculator) Init() tea.Cmd   { return nil }

func (c Calculator) Update(tea.Msg) (Mode, tea.Cmd) { return c, nil }

func (c Calculator) SetQuery(query string) Mode {
	c.query = strings.TrimSpace(query)
	c.value, c.valid = evalExpression(c.query)

	return c
}

func (c Calculator) HasResults() bool { return c.valid }
func (c Calculator) MoveUp() Mode     { return c }
func (c Calculator) MoveDown() Mode   { return c }

func (c Calculator) Activate() tea.Cmd {
	if !c.valid {
		return nil
	}

	result := c.formatted()

	return func() tea.Msg {
		copyToClipboard(result)

		return tea.QuitMsg{}
	}
}

func (c Calculator) View(width, rows int) string {
	if c.valid {
		return calculatorResultStyle.Render("= " + c.formatted())
	}

	if c.query == "" {
		return subtleStyle.Render("type an arithmetic expression")
	}

	return subtleStyle.Render("invalid expression")
}

func (c Calculator) formatted() string {
	precision := c.cfg.Precision

	if precision < 0 {
		precision = 0
	}

	text := strconv.FormatFloat(c.value, 'f', precision, 64)

	if strings.Contains(text, ".") {
		text = strings.TrimRight(text, "0")
		text = strings.TrimRight(text, ".")
	}

	return text
}

func copyToClipboard(text string) {
	tools := [][]string{
		{"wl-copy"},
		{"xclip", "-selection", "clipboard"},
		{"xsel", "--clipboard", "--input"},
	}

	for _, tool := range tools {
		path, err := exec.LookPath(tool[0])

		if err != nil {
			continue
		}

		cmd := exec.Command(path, tool[1:]...)
		cmd.Stdin = strings.NewReader(text)

		if cmd.Run() == nil {
			return
		}
	}
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
	value, ok := e.factor()

	if !ok {
		return 0, false
	}

	for {
		operator, present := e.peek()

		if !present || (operator != '*' && operator != '/' && operator != '%') {
			return value, true
		}

		e.pos++

		right, ok := e.factor()

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

func (e *expression) factor() (float64, bool) {
	base, ok := e.unary()

	if !ok {
		return 0, false
	}

	operator, present := e.peek()

	if present && operator == '^' {
		e.pos++

		exponent, ok := e.factor()

		if !ok {
			return 0, false
		}

		return math.Pow(base, exponent), true
	}

	return base, true
}

func (e *expression) unary() (float64, bool) {
	operator, present := e.peek()

	if present && (operator == '+' || operator == '-') {
		e.pos++

		value, ok := e.unary()

		if !ok {
			return 0, false
		}

		if operator == '-' {
			return -value, true
		}

		return value, true
	}

	return e.primary()
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
