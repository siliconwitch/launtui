package widgets

import (
	"math"
	"testing"
)

func TestEvalExpressionValid(t *testing.T) {
	cases := map[string]float64{
		"4+5":       9,
		"2+3*4":     14,
		"(2+3)*4":   20,
		"2^10":      1024,
		"2^3^2":     512,
		"10/4":      2.5,
		"7 % 3":     1,
		"-5+2":      -3,
		"-(3+4)":    -7,
		"1++2":      3,
		"  3.5*2  ": 7,
		".5+.5":     1,
		"-2^2":      -4,
		"2^-1":      0.5,
		"2*-3":      -6,
	}

	for input, want := range cases {
		got, ok := evalExpression(input)

		if !ok {
			t.Errorf("evalExpression(%q) failed, want %v", input, want)
			continue
		}

		if got != want {
			t.Errorf("evalExpression(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestEvalExpressionInvalid(t *testing.T) {
	cases := []string{"", "   ", "gmail", "4+", "(1+2", "/3", "%5", "5 5", "1.2.3", "5/0", "9%0", "9^9^9", "0^-1"}

	for _, input := range cases {
		if value, ok := evalExpression(input); ok {
			t.Errorf("evalExpression(%q) = %v, want failure", input, value)
		}
	}
}

func TestCalculatorAnswers(t *testing.T) {
	calculator := NewCalculator(DefaultCalculatorConfig())

	cases := map[string]string{
		"4+5":  "9",
		"10/4": "2.5",
		"2^10": "1024",
		"1/3":  "0.333333",
	}

	for input, want := range cases {
		updated := calculator.SetQuery(input).(Calculator)

		if !updated.valid {
			t.Errorf("SetQuery(%q) should be valid", input)
			continue
		}

		if updated.answer != want {
			t.Errorf("answer(%q) = %q, want %q", input, updated.answer, want)
		}
	}
}

func TestUnitConversions(t *testing.T) {
	calculator := NewCalculator(DefaultCalculatorConfig())

	cases := map[string]string{
		"5 miles to km":   "8.04672 km",
		"miles to km":     "1.609344 km",
		"10km to mi":      "6.213712 mi",
		"100 f to c":      "37.777778 °C",
		"0 c to f":        "32 °F",
		"2*3 m to ft":     "19.685039 ft",
		"1 gib to mb":     "1073.741824 MB",
		"90 min in h":     "1.5 h",
		"12 in to cm":     "30.48 cm",
		"5 ms to s":       "0.005 s",
		"3 kms to m":      "3000 m",
		"2 hours to mins": "120 min",
		"180 deg to rad":  "3.141593 rad",
		"1 turn to deg":   "360 °",
		"1 micron to nm":  "1000 nm",
		"1 mil to um":     "25.4 µm",
		"1 ghz to mhz":    "1000 MHz",
		"1 nf to pf":      "1000 pF",
		"1 µf to nf":      "1000 nF",
		"5 kohm to ohm":   "5000 Ω",
		"1 megohm to ohm": "1000000 Ω",
		"1 henry to mh":   "1000 mH",
		"3.3 v to mv":     "3300 mV",
		"2 a to ma":       "2000 mA",
		"1 kw to w":       "1000 W",
	}

	for input, want := range cases {
		updated := calculator.SetQuery(input).(Calculator)

		if !updated.valid {
			t.Errorf("SetQuery(%q) should be valid, note=%q", input, updated.note)
			continue
		}

		if updated.answer != want {
			t.Errorf("answer(%q) = %q, want %q", input, updated.answer, want)
		}
	}

	invalid := []string{"5 km to kg", "5 foo to bar", "gmail to usd", "5 ohm to farad", "1 deg to hz"}

	for _, input := range invalid {
		if updated := calculator.SetQuery(input).(Calculator); updated.valid {
			t.Errorf("SetQuery(%q) should be invalid, got %q", input, updated.answer)
		}
	}
}

func TestCurrencyConversion(t *testing.T) {
	calculator := NewCalculator(DefaultCalculatorConfig())
	calculator.rates = map[string]float64{"EUR": 1, "USD": 1.08, "GBP": 0.85}

	converted := calculator.SetQuery("10 gbp to usd").(Calculator)

	if !converted.valid || converted.answer != "12.71 USD" {
		t.Fatalf("10 gbp to usd = %q (valid=%v), want 12.71 USD", converted.answer, converted.valid)
	}

	rateOnly := calculator.SetQuery("gbp to usd").(Calculator)

	if !rateOnly.valid || rateOnly.answer != "1.27 USD" {
		t.Fatalf("gbp to usd = %q (valid=%v), want 1.27 USD", rateOnly.answer, rateOnly.valid)
	}

	unknown := calculator.SetQuery("10 xxx to usd").(Calculator)

	if unknown.valid || unknown.note != "unknown currency XXX" {
		t.Fatalf("unknown currency: valid=%v note=%q", unknown.valid, unknown.note)
	}

	pending := NewCalculator(DefaultCalculatorConfig()).SetQuery("gbp to usd").(Calculator)

	if pending.valid || pending.note != "fetching exchange rates…" {
		t.Fatalf("pending rates: valid=%v note=%q", pending.valid, pending.note)
	}
}

func TestCompletedCalculation(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	plain := NewCalculator(DefaultCalculatorConfig()).SetQuery("42").(Calculator)

	if _, cmd := plain.Update(AppClosingMsg{}); cmd != nil {
		t.Error("plain numbers should not be recorded")
	}

	sum := NewCalculator(DefaultCalculatorConfig()).SetQuery("4+5").(Calculator)

	_, cmd := sum.Update(AppClosingMsg{})

	if cmd == nil {
		t.Fatal("a fresh expression should be recorded on close")
	}

	cmd()

	path, err := launtuiDataPath(calculatorHistoryFile)

	if err != nil {
		t.Fatal(err)
	}

	history, ok := loadJSON[[]calculation](path)

	if !ok || len(history) != 1 || history[0].Expression != "4+5" || history[0].Answer != "9" {
		t.Fatalf("recorded history = %+v (ok=%v)", history, ok)
	}

	duplicate := sum
	duplicate.history = []calculation{{Expression: "4+5", Answer: "9"}}

	if _, cmd := duplicate.Update(AppClosingMsg{}); cmd != nil {
		t.Error("duplicate of newest history entry should not be recorded")
	}
}

func TestCalculatorHistorySelection(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	mode, _ := NewCalculator(DefaultCalculatorConfig()).Update(calculatorHistoryMsg{
		{Expression: "1+1", Answer: "2"},
		{Expression: "2+2", Answer: "4"},
	})

	calculator := mode.SetQuery("3+3").(Calculator)

	activatedAnswer := func(index int) string {
		t.Helper()

		cmd := calculator.Activate(index)

		if cmd == nil {
			t.Fatalf("activating index %d should produce a command", index)
		}

		cmd()

		entries := loadClipboardHistory()

		if len(entries) == 0 {
			t.Fatal("activation should copy the selected answer into clipboard history")
		}

		return entries[0].Text
	}

	if answer := activatedAnswer(0); answer != "6" {
		t.Fatalf("live answer = %q, want 6", answer)
	}

	if answer := activatedAnswer(1); answer != "2" {
		t.Fatalf("first history answer = %q, want 2", answer)
	}

	if answer := activatedAnswer(2); answer != "4" {
		t.Fatalf("second history answer = %q, want 4", answer)
	}

	if cmd := calculator.Activate(3); cmd != nil {
		t.Fatal("activating past the last row should do nothing")
	}
}

func TestCalculatorRecall(t *testing.T) {
	mode, _ := NewCalculator(DefaultCalculatorConfig()).Update(calculatorHistoryMsg{
		{Expression: "1+1", Answer: "2"},
		{Expression: "2+2", Answer: "4"},
	})

	withLive := mode.SetQuery("3+3").(Calculator)

	cases := []struct {
		index int
		text  string
		ok    bool
	}{
		{0, "", false},
		{1, "1+1", true},
		{2, "2+2", true},
		{3, "", false},
	}

	for _, testCase := range cases {
		text, ok := withLive.RecallText(testCase.index)

		if text != testCase.text || ok != testCase.ok {
			t.Errorf("RecallText(%d) = (%q, %v), want (%q, %v)", testCase.index, text, ok, testCase.text, testCase.ok)
		}
	}

	empty := mode.SetQuery("").(Calculator)

	if text, ok := empty.RecallText(0); text != "1+1" || !ok {
		t.Errorf("RecallText(0) without a live row = (%q, %v), want (1+1, true)", text, ok)
	}
}

func TestCalculatorDeleteHistory(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	mode, _ := NewCalculator(DefaultCalculatorConfig()).Update(calculatorHistoryMsg{
		{Expression: "1+1", Answer: "2"},
		{Expression: "2+2", Answer: "4"},
	})

	calculator := mode.SetQuery("3+3").(Calculator)

	if _, cmd := calculator.DeleteRow(0); cmd != nil {
		t.Fatal("deleting the live result should do nothing")
	}

	deleted, cmd := calculator.DeleteRow(1)

	if cmd == nil {
		t.Fatal("deleting a history entry should be persisted")
	}

	remaining := deleted.(Calculator)

	if len(remaining.history) != 1 || remaining.history[0].Expression != "2+2" {
		t.Fatalf("history after delete = %+v", remaining.history)
	}

	cleared, clearCmd := remaining.ClearRows()

	if len(cleared.(Calculator).history) != 0 || clearCmd == nil {
		t.Fatalf("history after clear = %+v", cleared.(Calculator).history)
	}
}

func TestUnitConversionsExtended(t *testing.T) {
	calculator := NewCalculator(DefaultCalculatorConfig())

	cases := map[string]string{
		"1 bar to kPa":   "100 kPa",
		"1 atm to kPa":   "101.325 kPa",
		"100 psi to kPa": "689.475729 kPa",
		"1 kgf to N":     "9.80665 N",
		"1 lbf to N":     "4.448222 N",
		"1 lbft to Nm":   "1.355818 N·m",
		"1 lps to lpm":   "60 L/min",
		"60 lpm to lps":  "1 L/s",
		"2 MV to kV":     "2000 kV",
		"2 MA to kA":     "2000 kA",
		"1 TW to GW":     "1000 GW",
		"1 hp to W":      "745.699872 W",
		"1 GiB to MiB":   "1024 MiB",
	}

	for input, want := range cases {
		updated := calculator.SetQuery(input).(Calculator)

		if !updated.valid {
			t.Errorf("SetQuery(%q) should be valid, note=%q", input, updated.note)
			continue
		}

		if updated.answer != want {
			t.Errorf("answer(%q) = %q, want %q", input, updated.answer, want)
		}
	}
}

func TestUnitCaseSensitivity(t *testing.T) {
	calculator := NewCalculator(DefaultCalculatorConfig())

	cases := map[string]string{
		"1 mV to V":    "0.001 V",
		"1 MV to V":    "1000000 V",
		"1 mA to A":    "0.001 A",
		"1 MA to A":    "1000000 A",
		"1 kb to kB":   "0.125 kB",
		"1 kB to kb":   "8 kb",
		"8 b to B":     "1 B",
		"1 MB to Mb":   "8 Mb",
		"5 v to mv":    "5000 mV",
		"1 ghz to mhz": "1000 MHz",
	}

	for input, want := range cases {
		updated := calculator.SetQuery(input).(Calculator)

		if !updated.valid {
			t.Errorf("SetQuery(%q) should be valid, note=%q", input, updated.note)
			continue
		}

		if updated.answer != want {
			t.Errorf("answer(%q) = %q, want %q", input, updated.answer, want)
		}
	}
}

func TestBaseConversion(t *testing.T) {
	calculator := NewCalculator(DefaultCalculatorConfig())

	cases := map[string]string{
		"255 to hex":    "0xff",
		"0xff to dec":   "255",
		"10 to bin":     "0b1010",
		"0b1010 to dec": "10",
		"255 to oct":    "0o377",
		"0o377 to dec":  "255",
		"65 to ascii":   "A",
		"0x41 to ascii": "A",
		"2^8 to hex":    "0x100",
		"'A' to dec":    "65",
		"255 in hex":    "0xff",
	}

	for input, want := range cases {
		updated := calculator.SetQuery(input).(Calculator)

		if !updated.valid {
			t.Errorf("SetQuery(%q) should be valid, note=%q", input, updated.note)
			continue
		}

		if updated.answer != want {
			t.Errorf("answer(%q) = %q, want %q", input, updated.answer, want)
		}
	}
}

func TestBitwiseExpressions(t *testing.T) {
	cases := map[string]float64{
		"12 & 10":       8,
		"12 | 10":       14,
		"5 xor 3":       6,
		"1 << 4":        16,
		"255 >> 4":      15,
		"~5":            -6,
		"0xff & 0x0f":   15,
		"0b1100 | 0b11": 15,
		"(1 | 2) << 3":  24,
		"1 << 2 + 3":    32,
		"1 | 2 & 3":     3,
		"2 * 3 & 4":     4,
		"~5 & 3":        2,
		"shl(1, 8)":     256,
	}

	for input, want := range cases {
		got, ok := evalExpression(input)

		if !ok {
			t.Errorf("evalExpression(%q) failed, want %v", input, want)
			continue
		}

		if got != want {
			t.Errorf("evalExpression(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestMathFunctions(t *testing.T) {
	cases := map[string]float64{
		"sqrt(144)":    12,
		"sqrt(2)":      1.4142135623730951,
		"cbrt(27)":     3,
		"abs(-7)":      7,
		"factorial(5)": 120,
		"log(1000)":    3,
		"log2(8)":      3,
		"ln(e)":        1,
		"sin(0)":       0,
		"floor(2.7)":   2,
		"ceil(2.1)":    3,
		"round(2.5)":   3,
		"pi":           math.Pi,
		"2 * pi":       2 * math.Pi,
		"exp(0)":       1,
	}

	for input, want := range cases {
		got, ok := evalExpression(input)

		if !ok {
			t.Errorf("evalExpression(%q) failed, want %v", input, want)
			continue
		}

		if math.Abs(got-want) > 1e-9 {
			t.Errorf("evalExpression(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestDecibelConversion(t *testing.T) {
	calculator := NewCalculator(DefaultCalculatorConfig())

	cases := map[string]string{
		"1 V to dBV":    "0 dBV",
		"0 dBV to V":    "1 V",
		"1 W to dBm":    "30 dBm",
		"0 dBm to mW":   "1 mW",
		"30 dBm to W":   "1 W",
		"1 W to dBW":    "0 dBW",
		"60 dBµV to mV": "1 mV",
		"1 mV to dBµV":  "60 dBµV",
	}

	for input, want := range cases {
		updated := calculator.SetQuery(input).(Calculator)

		if !updated.valid {
			t.Errorf("SetQuery(%q) should be valid, note=%q", input, updated.note)
			continue
		}

		if updated.answer != want {
			t.Errorf("answer(%q) = %q, want %q", input, updated.answer, want)
		}
	}
}
