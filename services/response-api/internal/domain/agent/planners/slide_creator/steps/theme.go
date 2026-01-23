package steps

import (
	"fmt"
	"hash/fnv"
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

func formatThemePreferences(config map[string]interface{}) string {
	if config == nil {
		return ""
	}
	theme := strings.TrimSpace(stringValue(config, "theme"))
	style := strings.TrimSpace(stringValue(config, "style"))
	colorScheme := strings.TrimSpace(stringValue(config, "color_scheme"))
	lines := []string{}
	if theme != "" {
		lines = append(lines, "- theme: "+theme)
	}
	if style != "" {
		lines = append(lines, "- style: "+style)
	}
	if colorScheme != "" {
		lines = append(lines, "- color scheme: "+colorScheme)
	}
	return strings.Join(lines, "\n")
}

func applyColorScheme(theme Theme, scheme string, seed string) Theme {
	switch strings.ToLower(strings.TrimSpace(scheme)) {
	case "bright", "light", "vibrant", "colorful", "colourful", "pastel":
		palette := pickBrightPalette(seed)
		theme.PrimaryColor = palette.PrimaryColor
		theme.AccentColor = palette.AccentColor
		theme.BackgroundColor = palette.BackgroundColor
		theme.TextColor = palette.TextColor
		return theme
	case "dark":
		if !isDarkColor(theme.BackgroundColor) {
			theme.BackgroundColor = "#0B1220"
		}
		theme.TextColor = "#F8FAFC"
		if !isHexColor(theme.PrimaryColor) {
			theme.PrimaryColor = "#38BDF8"
		}
		if !isHexColor(theme.AccentColor) {
			theme.AccentColor = "#F97316"
		}
	}
	return theme
}

func pickBrightPalette(seed string) Theme {
	palettes := []Theme{
		{PrimaryColor: "#F97316", AccentColor: "#14B8A6", BackgroundColor: "#FFF7ED", TextColor: "#0F172A"},
		{PrimaryColor: "#E11D48", AccentColor: "#F59E0B", BackgroundColor: "#FFF1F2", TextColor: "#111827"},
		{PrimaryColor: "#10B981", AccentColor: "#FB7185", BackgroundColor: "#F0FDF4", TextColor: "#0F172A"},
		{PrimaryColor: "#8B5CF6", AccentColor: "#22C55E", BackgroundColor: "#F5F3FF", TextColor: "#111827"},
		{PrimaryColor: "#06B6D4", AccentColor: "#F97316", BackgroundColor: "#ECFEFF", TextColor: "#0F172A"},
	}
	if len(palettes) == 0 {
		return Theme{PrimaryColor: "#F97316", AccentColor: "#14B8A6", BackgroundColor: "#F8FAFC", TextColor: "#0F172A"}
	}
	idx := int(hashString(seed)) % len(palettes)
	if idx < 0 {
		idx = 0
	}
	return palettes[idx]
}

func hashString(s string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return h.Sum32()
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
