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
	cases := []string{"", "   ", "gmail", "4+", "(1+2", "/3", "%5", "5 5", "1.2.3", "5/0", "9%0"}

	for _, input := range cases {
		if value, ok := evalExpression(input); ok {
			t.Errorf("evalExpression(%q) = %v, want failure", input, value)
		}
	}
}

func TestCalculatorFormatted(t *testing.T) {
	calculator := NewCalculator(DefaultCalculatorConfig())

	cases := map[string]string{
		"4+5":  "9",
		"10/4": "2.5",
		"2^10": "1024",
		"1/3":  "0.333333",
	}

	for input, want := range cases {
		updated := calculator.SetQuery(input).(Calculator)

		if got := updated.formatted(); got != want {
			t.Errorf("formatted(%q) = %q, want %q", input, got, want)
		}
	}
}
