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

	invalid := []string{"5 km to kg", "5 foo to bar", "gmail to usd"}

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
	calculator := NewCalculator(DefaultCalculatorConfig())

	plain := calculator.SetQuery("42").(Calculator)

	if _, ok := plain.completedCalculation(); ok {
		t.Error("plain numbers should not be recorded")
	}

	sum := calculator.SetQuery("4+5").(Calculator)

	entry, ok := sum.completedCalculation()

	if !ok || entry.Expression != "4+5" || entry.Answer != "9" {
		t.Fatalf("completedCalculation = %+v (ok=%v)", entry, ok)
	}

	sum.history = []calculation{{Expression: "4+5", Answer: "9"}}

	if _, ok := sum.completedCalculation(); ok {
		t.Error("duplicate of newest history entry should not be recorded")
	}
}

func TestCalculatorHistorySelection(t *testing.T) {
	mode, _ := NewCalculator(DefaultCalculatorConfig()).Update(calculatorHistoryMsg{
		{Expression: "1+1", Answer: "2"},
		{Expression: "2+2", Answer: "4"},
	})

	calculator := mode.SetQuery("3+3").(Calculator)

	if answer, ok := calculator.selectedAnswer(); !ok || answer != "6" {
		t.Fatalf("live answer = %q (ok=%v), want 6", answer, ok)
	}

	first := calculator.MoveDown().(Calculator)

	if answer, ok := first.selectedAnswer(); !ok || answer != "2" {
		t.Fatalf("first history answer = %q (ok=%v), want 2", answer, ok)
	}

	second := first.MoveDown().(Calculator)

	if answer, ok := second.selectedAnswer(); !ok || answer != "4" {
		t.Fatalf("second history answer = %q (ok=%v), want 4", answer, ok)
	}

	clamped := second.MoveDown().(Calculator)

	if clamped.cursor != second.cursor {
		t.Fatal("cursor should clamp at the last history entry")
	}

	reset := clamped.SetQuery("5+5").(Calculator)

	if reset.cursor != 0 {
		t.Fatal("cursor should reset when the query changes")
	}
}

func TestCalculatorDeleteSelectedHistory(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	mode, _ := NewCalculator(DefaultCalculatorConfig()).Update(calculatorHistoryMsg{
		{Expression: "1+1", Answer: "2"},
		{Expression: "2+2", Answer: "4"},
	})

	calculator := mode.SetQuery("3+3").(Calculator)

	if _, _, handled := calculator.DeleteSelectedHistory(); handled {
		t.Fatal("delete on the live result should not be handled")
	}

	deleted, cmd, handled := calculator.MoveDown().(Calculator).DeleteSelectedHistory()

	if !handled || cmd == nil {
		t.Fatal("deleting a history entry should be handled and persisted")
	}

	remaining := deleted.(Calculator)

	if len(remaining.history) != 1 || remaining.history[0].Expression != "2+2" {
		t.Fatalf("history after delete = %+v", remaining.history)
	}

	cleared, clearCmd := remaining.ClearHistory()

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
