package widgets

import (
	"strings"
	"time"
	_ "time/tzdata"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ClockConfig struct {
	Enabled bool     `toml:"enabled"`
	Format  string   `toml:"format"`
	Zones   []string `toml:"zones"`
}

func (ClockConfig) SectionName() string { return "clock" }

func DefaultClockConfig() ClockConfig {
	return ClockConfig{Enabled: true, Format: "Mon 2 Jan - 15:04"}
}

var clockStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Bold(true)

type clockTickMsg time.Time

type clockZone struct {
	location *time.Location
	label    string
}

type Clock struct {
	cfg     ClockConfig
	now     time.Time
	zones   []clockZone
	current int
}

func NewClock(cfg ClockConfig) Clock {
	zones := []clockZone{{location: time.Local}}

	for _, entry := range cfg.Zones {
		trimmed := strings.TrimSpace(entry)
		name := trimmed
		label := ""

		if city, ok := cityZones[strings.ToLower(trimmed)]; ok {
			name = city.zone
			label = city.display
		}

		location, err := time.LoadLocation(name)

		if err != nil {
			continue
		}

		if label == "" {
			label = name

			if index := strings.LastIndex(label, "/"); index >= 0 {
				label = label[index+1:]
			}

			label = strings.ReplaceAll(label, "_", " ")
		}

		zones = append(zones, clockZone{location: location, label: label})
	}

	return Clock{cfg: cfg, now: time.Now(), zones: zones}
}

func (c Clock) Enabled() bool { return c.cfg.Enabled }

func (c Clock) NextZone() Clock {
	if len(c.zones) > 1 {
		c.current = (c.current + 1) % len(c.zones)
	}

	return c
}

func (c Clock) Init() tea.Cmd {
	if !c.cfg.Enabled {
		return nil
	}

	return clockTick()
}

func (c Clock) Update(msg tea.Msg) (Clock, tea.Cmd) {
	tick, ok := msg.(clockTickMsg)

	if !ok {
		return c, nil
	}

	c.now = time.Time(tick)

	return c, clockTick()
}

func (c Clock) View() string {
	if !c.cfg.Enabled {
		return ""
	}

	zone := c.zones[c.current]
	text := c.now.In(zone.location).Format(c.cfg.Format)

	if c.current != 0 {
		text += " (" + zone.label + ")"
	}

	return clockStyle.Render(text)
}

func clockTick() tea.Cmd {
	return tea.Every(time.Second, func(t time.Time) tea.Msg {
		return clockTickMsg(t)
	})
}

type cityZone struct {
	display string
	zone    string
}

var cityZones = map[string]cityZone{
	"utc": {"UTC", "UTC"},
	"gmt": {"GMT", "UTC"},

	"new york":      {"New York", "America/New_York"},
	"nyc":           {"New York", "America/New_York"},
	"washington":    {"Washington", "America/New_York"},
	"boston":        {"Boston", "America/New_York"},
	"atlanta":       {"Atlanta", "America/New_York"},
	"miami":         {"Miami", "America/New_York"},
	"chicago":       {"Chicago", "America/Chicago"},
	"dallas":        {"Dallas", "America/Chicago"},
	"houston":       {"Houston", "America/Chicago"},
	"austin":        {"Austin", "America/Chicago"},
	"denver":        {"Denver", "America/Denver"},
	"phoenix":       {"Phoenix", "America/Phoenix"},
	"los angeles":   {"Los Angeles", "America/Los_Angeles"},
	"la":            {"Los Angeles", "America/Los_Angeles"},
	"san francisco": {"San Francisco", "America/Los_Angeles"},
	"sf":            {"San Francisco", "America/Los_Angeles"},
	"seattle":       {"Seattle", "America/Los_Angeles"},
	"toronto":       {"Toronto", "America/Toronto"},
	"montreal":      {"Montreal", "America/Toronto"},
	"vancouver":     {"Vancouver", "America/Vancouver"},
	"mexico city":   {"Mexico City", "America/Mexico_City"},

	"sao paulo":      {"São Paulo", "America/Sao_Paulo"},
	"são paulo":      {"São Paulo", "America/Sao_Paulo"},
	"rio":            {"Rio de Janeiro", "America/Sao_Paulo"},
	"rio de janeiro": {"Rio de Janeiro", "America/Sao_Paulo"},
	"buenos aires":   {"Buenos Aires", "America/Argentina/Buenos_Aires"},
	"santiago":       {"Santiago", "America/Santiago"},
	"lima":           {"Lima", "America/Lima"},
	"bogota":         {"Bogotá", "America/Bogota"},
	"bogotá":         {"Bogotá", "America/Bogota"},

	"london":     {"London", "Europe/London"},
	"dublin":     {"Dublin", "Europe/Dublin"},
	"lisbon":     {"Lisbon", "Europe/Lisbon"},
	"madrid":     {"Madrid", "Europe/Madrid"},
	"barcelona":  {"Barcelona", "Europe/Madrid"},
	"paris":      {"Paris", "Europe/Paris"},
	"brussels":   {"Brussels", "Europe/Brussels"},
	"amsterdam":  {"Amsterdam", "Europe/Amsterdam"},
	"berlin":     {"Berlin", "Europe/Berlin"},
	"munich":     {"Munich", "Europe/Berlin"},
	"zurich":     {"Zürich", "Europe/Zurich"},
	"geneva":     {"Geneva", "Europe/Zurich"},
	"rome":       {"Rome", "Europe/Rome"},
	"vienna":     {"Vienna", "Europe/Vienna"},
	"prague":     {"Prague", "Europe/Prague"},
	"warsaw":     {"Warsaw", "Europe/Warsaw"},
	"stockholm":  {"Stockholm", "Europe/Stockholm"},
	"oslo":       {"Oslo", "Europe/Oslo"},
	"copenhagen": {"Copenhagen", "Europe/Copenhagen"},
	"helsinki":   {"Helsinki", "Europe/Helsinki"},
	"athens":     {"Athens", "Europe/Athens"},
	"istanbul":   {"Istanbul", "Europe/Istanbul"},
	"moscow":     {"Moscow", "Europe/Moscow"},
	"kyiv":       {"Kyiv", "Europe/Kyiv"},
	"kiev":       {"Kyiv", "Europe/Kyiv"},

	"cairo":        {"Cairo", "Africa/Cairo"},
	"casablanca":   {"Casablanca", "Africa/Casablanca"},
	"lagos":        {"Lagos", "Africa/Lagos"},
	"nairobi":      {"Nairobi", "Africa/Nairobi"},
	"johannesburg": {"Johannesburg", "Africa/Johannesburg"},
	"cape town":    {"Cape Town", "Africa/Johannesburg"},

	"dubai":     {"Dubai", "Asia/Dubai"},
	"abu dhabi": {"Abu Dhabi", "Asia/Dubai"},
	"doha":      {"Doha", "Asia/Qatar"},
	"riyadh":    {"Riyadh", "Asia/Riyadh"},
	"tel aviv":  {"Tel Aviv", "Asia/Jerusalem"},
	"jerusalem": {"Jerusalem", "Asia/Jerusalem"},
	"tehran":    {"Tehran", "Asia/Tehran"},

	"karachi":      {"Karachi", "Asia/Karachi"},
	"mumbai":       {"Mumbai", "Asia/Kolkata"},
	"bombay":       {"Mumbai", "Asia/Kolkata"},
	"delhi":        {"Delhi", "Asia/Kolkata"},
	"new delhi":    {"New Delhi", "Asia/Kolkata"},
	"bangalore":    {"Bangalore", "Asia/Kolkata"},
	"bengaluru":    {"Bengaluru", "Asia/Kolkata"},
	"kolkata":      {"Kolkata", "Asia/Kolkata"},
	"calcutta":     {"Kolkata", "Asia/Kolkata"},
	"dhaka":        {"Dhaka", "Asia/Dhaka"},
	"bangkok":      {"Bangkok", "Asia/Bangkok"},
	"hanoi":        {"Hanoi", "Asia/Ho_Chi_Minh"},
	"ho chi minh":  {"Ho Chi Minh City", "Asia/Ho_Chi_Minh"},
	"saigon":       {"Ho Chi Minh City", "Asia/Ho_Chi_Minh"},
	"jakarta":      {"Jakarta", "Asia/Jakarta"},
	"singapore":    {"Singapore", "Asia/Singapore"},
	"kuala lumpur": {"Kuala Lumpur", "Asia/Kuala_Lumpur"},
	"manila":       {"Manila", "Asia/Manila"},
	"hong kong":    {"Hong Kong", "Asia/Hong_Kong"},
	"beijing":      {"Beijing", "Asia/Shanghai"},
	"shanghai":     {"Shanghai", "Asia/Shanghai"},
	"shenzhen":     {"Shenzhen", "Asia/Shanghai"},
	"taipei":       {"Taipei", "Asia/Taipei"},
	"seoul":        {"Seoul", "Asia/Seoul"},
	"tokyo":        {"Tokyo", "Asia/Tokyo"},
	"osaka":        {"Osaka", "Asia/Tokyo"},

	"sydney":     {"Sydney", "Australia/Sydney"},
	"melbourne":  {"Melbourne", "Australia/Melbourne"},
	"brisbane":   {"Brisbane", "Australia/Brisbane"},
	"perth":      {"Perth", "Australia/Perth"},
	"auckland":   {"Auckland", "Pacific/Auckland"},
	"wellington": {"Wellington", "Pacific/Auckland"},
}
