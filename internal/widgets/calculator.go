package widgets

// CalculatorConfig configures the calculator.
type CalculatorConfig struct {
	Precision int    `toml:"precision"` // digits after the decimal point
	Angle     string `toml:"angle"`     // "rad" or "deg"
}

func (CalculatorConfig) SectionName() string { return "calculator" }

func DefaultCalculatorConfig() CalculatorConfig {
	return CalculatorConfig{Precision: 2, Angle: "rad"}
}

// Calculator — evaluate expressions. To be defined together.
