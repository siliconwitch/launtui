//go:build darwin

package widgets

import (
	"math"
	"testing"
)

func TestParseBatteryPmset(t *testing.T) {
	cases := []struct {
		name    string
		output  string
		present bool
		percent int
		status  string
		hours   float64
	}{
		{
			name:    "discharging",
			output:  "Now drawing from 'Battery Power'\n -InternalBattery-0 (id=22347875)\t83%; discharging; 4:30 remaining present: true",
			present: true,
			percent: 83,
			status:  "Discharging",
			hours:   4.5,
		},
		{
			name:    "charging",
			output:  "Now drawing from 'AC Power'\n -InternalBattery-0 (id=22347875)\t45%; charging; 1:12 remaining present: true",
			present: true,
			percent: 45,
			status:  "Charging",
			hours:   1.2,
		},
		{
			name:    "charged on AC",
			output:  "Now drawing from 'AC Power'\n -InternalBattery-0 (id=22347875)\t100%; charged; 0:00 remaining present: true",
			present: true,
			percent: 100,
			status:  "Full",
			hours:   0,
		},
		{
			name:    "AC attached not charging",
			output:  "Now drawing from 'AC Power'\n -InternalBattery-0 (id=22347875)\t90%; AC attached; not charging present: true",
			present: true,
			percent: 90,
			status:  "Not charging",
			hours:   0,
		},
		{
			name:    "no estimate yet",
			output:  "Now drawing from 'AC Power'\n -InternalBattery-0 (id=22347875)\t60%; charging; (no estimate) present: true",
			present: true,
			percent: 60,
			status:  "Charging",
			hours:   0,
		},
		{
			name:    "desktop without battery",
			output:  "Now drawing from 'AC Power'",
			present: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reading := parseBatteryPmset(tc.output)

			if reading.present != tc.present {
				t.Fatalf("present = %v, want %v", reading.present, tc.present)
			}

			if !tc.present {
				return
			}

			if reading.percent != tc.percent || reading.status != tc.status {
				t.Fatalf("reading = %+v", reading)
			}

			if math.Abs(reading.hours-tc.hours) > 0.001 {
				t.Fatalf("hours = %v, want %v", reading.hours, tc.hours)
			}
		})
	}
}
