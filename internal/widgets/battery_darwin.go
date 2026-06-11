//go:build darwin

package widgets

import (
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var (
	batteryPercentPattern = regexp.MustCompile(`(\d+)%`)
	batteryTimePattern    = regexp.MustCompile(`(\d+):(\d{2})`)
)

func readBattery(device string) batteryReading {
	output, err := exec.Command("pmset", "-g", "batt").Output()

	if err != nil {
		return batteryReading{}
	}

	return parseBatteryPmset(string(output))
}

func parseBatteryPmset(output string) batteryReading {
	line := batteryStatusLine(output)

	if line == "" {
		return batteryReading{}
	}

	percentMatch := batteryPercentPattern.FindStringSubmatch(line)

	if percentMatch == nil {
		return batteryReading{}
	}

	percent, _ := strconv.Atoi(percentMatch[1])

	return batteryReading{
		present: true,
		status:  batteryStatusFromLine(line),
		percent: percent,
		hours:   batteryHoursFromLine(line),
	}
}

func batteryStatusLine(output string) string {
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "%") && strings.Contains(line, ";") {
			return line
		}
	}

	return ""
}

func batteryStatusFromLine(line string) string {
	lower := strings.ToLower(line)

	switch {
	case strings.Contains(lower, "discharging"):
		return "Discharging"
	case strings.Contains(lower, "charged"):
		return "Full"
	case strings.Contains(lower, "finishing charge"):
		return "Charging"
	case strings.Contains(lower, "not charging"):
		return "Not charging"
	case strings.Contains(lower, "charging"):
		return "Charging"
	}

	return ""
}

func batteryHoursFromLine(line string) float64 {
	if !strings.Contains(line, "remaining") {
		return 0
	}

	match := batteryTimePattern.FindStringSubmatch(line)

	if match == nil {
		return 0
	}

	hours, _ := strconv.Atoi(match[1])
	minutes, _ := strconv.Atoi(match[2])

	return float64(hours) + float64(minutes)/60
}
