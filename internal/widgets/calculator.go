package widgets

type CalculatorConfig struct {
	Precision int    `toml:"precision"`
	Angle     string `toml:"angle"`
}

func (CalculatorConfig) SectionName() string { return "calculator" }

func DefaultCalculatorConfig() CalculatorConfig {
	return CalculatorConfig{Precision: 2, Angle: "rad"}
}
