package duration

import (
	"math"
	"strconv"
	"time"

	"github.com/dustin/go-humanize/english"
)

func FormatTime(t time.Time) string {
	if time.Since(t) < 0 {
		return "until " + Format(time.Until(t)) + " from now"
	}

	return Format(time.Since(t)) + " ago"
}

// Format returns d as a human-readable string.
// For consistency reasons, months are always treated as 30 days, and years are always treated as 365 days.
func Format(d time.Duration) string {
	if months, _ := Months(d); months < 1 {
		return formatWithoutMonth(d)
	} else if years, _ := Years(d); years < 1 {
		return formatWithoutYear(d)
	}

	s := make([]string, 0, 4)

	years, d := Years(d)
	if years > 0 {
		s = append(s, plural(years, "year"))
	}

	months, d := Months(d)
	if months > 0 {
		s = append(s, plural(months, "month"))
	}

	days, d := Days(d)
	if days > 0 {
		s = append(s, plural(days, "day"))
	}

	hours, d := Hours(d)
	if hours > 0 {
		s = append(s, plural(hours, "hour"))
	}

	return english.OxfordWordSeries(s, "and")
}

// formatWithoutYear formats d with no year unit, but with minutes.
func formatWithoutYear(d time.Duration) string {
	s := make([]string, 0, 4)

	months, d := Months(d)
	if months > 0 {
		s = append(s, plural(months, "month"))
	}

	days, d := Days(d)
	if days > 0 {
		s = append(s, plural(days, "day"))
	}

	hours, d := Hours(d)
	if hours > 0 {
		s = append(s, plural(hours, "hour"))
	}

	minutes, _ := Minutes(d)
	if minutes > 0 {
		s = append(s, plural(minutes, "minute"))
	}

	return english.OxfordWordSeries(s, "and")
}

// formatWithoutMonth formats d with no month or year units, but with minute and second units.
func formatWithoutMonth(d time.Duration) string {
	s := make([]string, 0, 4)

	days, d := Days(d)
	if days > 0 {
		s = append(s, plural(days, "day"))
	}

	hours, d := Hours(d)
	if hours > 0 {
		s = append(s, plural(hours, "hour"))
	}

	minutes, d := Minutes(d)
	if minutes > 0 {
		s = append(s, plural(minutes, "minute"))
	}

	if d.Seconds() > 0 {
		seconds := int(math.Round(d.Seconds()))
		s = append(s, plural(seconds, "second"))
	}

	return english.OxfordWordSeries(s, "and")
}

func plural(i int, word string) string {
	if i == 1 {
		return "1 " + word
	}
	return strconv.Itoa(i) + " " + word + "s"
}

const (
	day = 24 * time.Hour

	year  = 365 * day
	month = 30 * day
)

func Years(d time.Duration) (years int, left time.Duration) {
	years = int(math.Floor(float64(d.Seconds()/86400) / 365))

	return years, d - (time.Duration(years) * year)
}

func Months(d time.Duration) (months int, left time.Duration) {
	months = int(math.Floor(float64(d.Seconds()/86400) / 30))

	return months, d - (time.Duration(months) * month)
}

func Days(d time.Duration) (days int, left time.Duration) {
	days = int(math.Floor(float64(d.Seconds() / 86400)))

	return days, d - (time.Duration(days) * day)
}

func Hours(d time.Duration) (hours int, left time.Duration) {
	hours = int(math.Floor(float64(d.Seconds() / 3600)))

	return hours, d - (time.Duration(hours) * time.Hour)
}

func Minutes(d time.Duration) (minutes int, left time.Duration) {
	minutes = int(math.Floor(float64(d.Seconds() / 60)))

	return minutes, d - (time.Duration(minutes) * time.Minute)
}
