package steps

import (
	"fmt"
	"html/template"
	"math"
	"strconv"
	"strings"
)

func normalizeTheme(theme Theme) Theme {
	if !isHexColor(theme.PrimaryColor) {
		theme.PrimaryColor = "#2563EB"
	}
	if !isHexColor(theme.AccentColor) {
		theme.AccentColor = "#F97316"
	}
	if !isHexColor(theme.BackgroundColor) {
		theme.BackgroundColor = "#FFFFFF"
	}
	if !isHexColor(theme.TextColor) {
		theme.TextColor = "#0F172A"
	}
	theme.FontFamily = strings.TrimSpace(theme.FontFamily)
	if theme.FontFamily == "" {
		theme.FontFamily = "Segoe UI, Arial, Helvetica, sans-serif"
	}
	return theme
}

func parseHex(s string) (int, int, int, bool) {
	s = strings.TrimSpace(s)
	if !isHexColor(s) {
		return 0, 0, 0, false
	}
	r, err1 := strconv.ParseInt(s[1:3], 16, 0)
	g, err2 := strconv.ParseInt(s[3:5], 16, 0)
	b, err3 := strconv.ParseInt(s[5:7], 16, 0)
	if err1 != nil || err2 != nil || err3 != nil {
		return 0, 0, 0, false
	}
	return int(r), int(g), int(b), true
}

func isDarkColor(hex string) bool {
	r, g, b, ok := parseHex(hex)
	if !ok {
		return false
	}
	L := (0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(b)) / 255.0
	return L < 0.45
}

func safeCSS(value string) template.CSS {
	return template.CSS(value)
}

func withAlpha(hex string, alpha float64) template.CSS {
	r, g, b, ok := parseHex(hex)
	if !ok {
		return safeCSS(fmt.Sprintf("rgba(15,23,42,%.3f)", clampFloat(alpha, 0, 1)))
	}
	a := clampFloat(alpha, 0, 1)
	return safeCSS(fmt.Sprintf("rgba(%d,%d,%d,%.3f)", r, g, b, a))
}

func mixHex(aHex, bHex string, t float64) string {
	ar, ag, ab, okA := parseHex(aHex)
	br, bg, bb, okB := parseHex(bHex)
	if !okA || !okB {
		return aHex
	}
	t = clampFloat(t, 0, 1)
	mix := func(x, y int) int {
		v := float64(x) + (float64(y)-float64(x))*t
		return int(math.Round(clampFloat(v, 0, 255)))
	}
	r := mix(ar, br)
	g := mix(ag, bg)
	b := mix(ab, bb)
	return fmt.Sprintf("#%02X%02X%02X", r, g, b)
}

func clampFloat(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func ternaryFloat(cond bool, a, b float64) float64 {
	if cond {
		return a
	}
	return b
}

func ternaryString(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}
