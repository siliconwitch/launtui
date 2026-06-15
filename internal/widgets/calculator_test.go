package widgets

import "testing"

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

	history, _ := loadJSON[[]calculation](path)

	if len(history) != 1 || history[0].Expression != "4+5" || history[0].Answer != "9" {
		t.Fatalf("recorded history = %+v", history)
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

func TestCalculatorRecordsHistoryOnClose(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	calculator := NewCalculator(DefaultCalculatorConfig()).SetQuery("4+5").(Calculator)

	_, cmd := calculator.Update(AppClosingMsg{})

	if cmd == nil {
		t.Fatal("expected a persist command on close")
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
}
