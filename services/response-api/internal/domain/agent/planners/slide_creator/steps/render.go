package steps

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"math"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	slideWidth  = 1920
	slideHeight = 1080
)

type SplitTemplateData struct {
	Bullets      []string
	BulletFont   int
	Label        string
	Subtitle     string
	Image        SlideImage
	ImageAlt     string
	ImageCaption string
	HasImage     bool
	Stats        []StatItem
}

type StatItem struct {
	Value string
	Label string
}

type BulletsTemplateData struct {
	Bullets      []string
	ItemFont     int
	Subtitle     string
	Label        string
	Stats        []StatItem
	Image        SlideImage
	ImageAlt     string
	ImageCaption string
	HasImage     bool
}

type HeroTemplateData struct {
	Image        SlideImage
	ImageAlt     string
	ImageCaption string
	Kicker       string
	Badge        string
}

type TitleTemplateData struct {
	Notes string
}

type TemplateData struct {
	SlideW     int
	Plan       DeckPlan
	Slide      SlidePlan
	Index      int
	Total      int
	Theme      Theme
	Muted      template.CSS
	Line       template.CSS
	CardBorder template.CSS
	Shadow     template.CSS
	Background template.CSS
	AccGlow    template.CSS
	PriGlow    template.CSS
	CardBg     template.CSS
	BodyBorder template.CSS
	Split      SplitTemplateData
	Bullets    BulletsTemplateData
	Table      TableTemplateData
	Chart      ChartTemplateData
	Hero       HeroTemplateData
	Title      TitleTemplateData
	BodyHTML   template.HTML
}

type TableTemplateData struct {
	Title   string
	Columns []string
	Rows    [][]string
	Notes   string
	Has     bool
}

type ChartBar struct {
	X     float64
	Y     float64
	W     float64
	H     float64
	Value float64
	Color string
	Label string
}

type ChartPoint struct {
	X     float64
	Y     float64
	Value float64
}

type ChartSlice struct {
	Path  string
	Color string
	Label string
}

type ChartTemplateData struct {
	Type       string
	Title      string
	Categories []string
	Series     []ChartSeries
	Bars       []ChartBar
	LinePaths  []string
	Points     [][]ChartPoint
	Slices     []ChartSlice
	Colors     []string
	Notes      string
	Has        bool
}

func buildTemplateData(plan DeckPlan, slide SlidePlan, index, total int, bodyHTML string) TemplateData {
	theme := normalizeTheme(plan.Theme)
	isDark := isDarkColor(theme.BackgroundColor)
	muted := withAlpha(theme.TextColor, 0.86)
	line := safeCSS("rgba(15,23,42,0.10)")
	cardBorder := safeCSS("rgba(15,23,42,0.10)")
	shadow := safeCSS("rgba(15,23,42,0.12)")
	if isDark {
		line = safeCSS("rgba(255,255,255,0.14)")
		cardBorder = safeCSS("rgba(255,255,255,0.14)")
		shadow = safeCSS("rgba(0,0,0,0.35)")
	}
	bg2 := theme.BackgroundColor
	if isDark {
		bg2 = mixHex(theme.BackgroundColor, "#000000", 0.18)
	} else {
		bg2 = mixHex(theme.BackgroundColor, "#FFFFFF", 0.18)
	}
	background := safeCSS(fmt.Sprintf("linear-gradient(135deg, %s 0%%, %s 100%%)", theme.BackgroundColor, bg2))
	accGlow := withAlpha(theme.AccentColor, 0.22)
	priGlow := withAlpha(theme.PrimaryColor, 0.12)
	cardBg := withAlpha("#FFFFFF", 0.92)
	if isDark {
		cardBg = withAlpha("#0B1220", 0.78)
	}

	bodyBorder := safeCSS("rgba(148,163,184,0.22)")
	if isDark {
		bodyBorder = safeCSS("rgba(255,255,255,0.18)")
	}

	tableInfo := normalizeTable(slide.Table, 6, 9)
	chartInfo := normalizeChart(slide.Chart, theme)

	splitImage := SlideImage{}
	if len(slide.Images) > 0 {
		splitImage = slide.Images[0]
	}

	splitLabel := strings.TrimSpace(slide.Subtitle)
	if splitLabel == "" && len(slide.Bullets) > 0 {
		splitLabel = slide.Bullets[0]
	}

	data := TemplateData{
		SlideW:     slideWidth,
		Plan:       plan,
		Slide:      slide,
		Index:      index,
		Total:      total,
		Theme:      theme,
		Muted:      muted,
		Line:       line,
		CardBorder: cardBorder,
		Shadow:     shadow,
		Background: background,
		AccGlow:    accGlow,
		PriGlow:    priGlow,
		CardBg:     cardBg,
		BodyBorder: bodyBorder,
		Split: SplitTemplateData{
			Bullets:      slide.Bullets,
			BulletFont:   ternaryInt(len(slide.Bullets) > 4, 20, 22),
			Label:        splitLabel,
			Subtitle:     slide.Subtitle,
			Image:        splitImage,
			ImageAlt:     splitImage.Alt,
			ImageCaption: splitImage.Caption,
			HasImage:     splitImage.Src != "",
			Stats:        extractStats(slide),
		},
		Bullets: BulletsTemplateData{
			Bullets:      slide.Bullets,
			ItemFont:     ternaryInt(len(slide.Bullets) > 4, 22, 24),
			Subtitle:     slide.Subtitle,
			Label:        extractBulletsLabel(slide),
			Stats:        extractStats(slide),
			Image:        splitImage,
			ImageAlt:     splitImage.Alt,
			ImageCaption: splitImage.Caption,
			HasImage:     splitImage.Src != "",
		},
		Table: tableInfo,
		Chart: chartInfo,
		Hero: HeroTemplateData{
			Image:        splitImage,
			ImageAlt:     splitImage.Alt,
			ImageCaption: splitImage.Caption,
			Kicker:       planSafeDeckTitle(slide),
			Badge:        mutedLabel(slide),
		},
		Title:    TitleTemplateData{Notes: slide.Notes},
		BodyHTML: template.HTML(bodyHTML),
	}

	return data
}

func renderSlideFromTemplate(layout string, data TemplateData, templateFS fs.FS, templateDir string, fallbackFS fs.FS, fallbackDir string) (string, error) {
	basePath, _, err := resolveTemplatePaths(layout, templateFS, templateDir, fallbackFS, fallbackDir)
	if err != nil {
		return "", err
	}

	funcs := template.FuncMap{
		"trimToRunes":       trimToRunes,
		"withAlpha":         withAlpha,
		"planSafeDeckTitle": planSafeDeckTitle,
		"mutedLabel":        mutedLabel,
	}

	tmpl, err := template.New("base.html").Funcs(funcs).ParseFS(basePath.fs, basePath.base, basePath.layout)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "base.html", data); err != nil {
		return "", err
	}
	html := buf.String()
	html = tuneImageOverlays(html, data.Theme)
	return html, nil
}

func tuneImageOverlays(html string, theme Theme) string {
	if html == "" || isDarkColor(theme.BackgroundColor) {
		return html
	}
	// Lighten image overlays for bright themes to keep text readable.
	replacements := map[string]string{
		"rgba(0,0,0,0.70)": "rgba(0,0,0,0.45)",
		"rgba(0,0,0,0.62)": "rgba(0,0,0,0.40)",
		"rgba(0,0,0,0.30)": "rgba(0,0,0,0.18)",
		"rgba(0,0,0,0.18)": "rgba(0,0,0,0.10)",
		"rgba(0,0,0,0.05)": "rgba(0,0,0,0.02)",
	}
	for oldValue, newValue := range replacements {
		html = strings.ReplaceAll(html, oldValue, newValue)
	}
	return html
}

type templatePaths struct {
	fs     fs.FS
	base   string
	layout string
}

func resolveTemplatePaths(layout string, templateFS fs.FS, templateDir string, fallbackFS fs.FS, fallbackDir string) (*templatePaths, string, error) {
	layout = strings.ToLower(strings.TrimSpace(layout))
	if layout == "" {
		layout = "split"
	}

	basePrimary := path.Join(templateDir, "base.html")
	layoutPrimary := path.Join(templateDir, layout+".html")
	if fileExists(templateFS, basePrimary) && fileExists(templateFS, layoutPrimary) {
		return &templatePaths{fs: templateFS, base: basePrimary, layout: layoutPrimary}, layout, nil
	}

	baseFallback := path.Join(fallbackDir, "base.html")
	layoutFallback := path.Join(fallbackDir, layout+".html")
	if fileExists(fallbackFS, baseFallback) && fileExists(fallbackFS, layoutFallback) {
		return &templatePaths{fs: fallbackFS, base: baseFallback, layout: layoutFallback}, layout, nil
	}

	return nil, "", fmt.Errorf("template files missing for layout %q", layout)
}

func fileExists(templateFS fs.FS, path string) bool {
	if templateFS == nil {
		return false
	}
	info, err := fs.Stat(templateFS, path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func writeSlideHTML(outDir string, slideID int, html string) error {
	return writeFile(filepath.Join(outDir, fmt.Sprintf("slide-%d.html", slideID)), []byte(html))
}

func writeIndexHTML(outDir string, plan DeckPlan) error {
	items := make([]string, 0, len(plan.Slides))
	for _, s := range plan.Slides {
		items = append(items, fmt.Sprintf(`<li style="margin:10px 0;"><a style="color:#2563EB; text-decoration:none; font-weight:700;" href="slide-%d.html">Slide %d - %s</a></li>`, s.ID, s.ID, escapeHTML(s.Title)))
	}
	html := "<!doctype html>\n<html><head><meta charset=\"utf-8\"></head><body style=\"font-family:Segoe UI, Arial, sans-serif; padding:24px;\">" +
		"<h1 style=\"margin:0 0 12px 0;\">" + escapeHTML(plan.Title) + "</h1>" +
		"<p style=\"margin:0 0 18px 0; color:#334155;\">Open each slide HTML to preview before exporting to PPTX.</p>" +
		"<ol style=\"padding-left:18px;\">" + strings.Join(items, "") + "</ol>" +
		"</body></html>\n"
	return writeFile(filepath.Join(outDir, "index.html"), []byte(html))
}

func writeFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func trimToRunes(s string, maxRunes int) string {
	s = strings.TrimSpace(s)
	if maxRunes <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	out := make([]rune, 0, maxRunes)
	for _, r := range s {
		out = append(out, r)
		if len(out) >= maxRunes {
			break
		}
	}
	return strings.TrimSpace(string(out)) + "..."
}

func planSafeDeckTitle(slide SlidePlan) string {
	_ = slide
	return "Key visual"
}

func mutedLabel(slide SlidePlan) string {
	if strings.TrimSpace(slide.Subtitle) != "" {
		return slide.Subtitle
	}
	if len(slide.Bullets) > 0 {
		return slide.Bullets[0]
	}
	return slide.Title
}

func escapeHTML(s string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
		"'", "&#39;",
	)
	return replacer.Replace(s)
}

func ternaryInt(cond bool, a, b int) int {
	if cond {
		return a
	}
	return b
}

func normalizeTable(table *TableData, maxCols, maxRows int) TableTemplateData {
	if table == nil {
		return TableTemplateData{}
	}
	title := strings.TrimSpace(table.Title)
	notes := strings.TrimSpace(table.Notes)

	cols := make([]string, 0, maxCols)
	for _, c := range table.Columns {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		cols = append(cols, c)
		if len(cols) >= maxCols {
			break
		}
	}

	rows := make([][]string, 0, maxRows)
	for _, r := range table.Rows {
		if len(rows) >= maxRows {
			break
		}
		row := make([]string, 0, len(cols))
		for i := 0; i < len(cols); i++ {
			var cell string
			if i < len(r) {
				switch v := r[i].(type) {
				case string:
					cell = strings.TrimSpace(v)
				case float64:
					cell = strings.TrimSpace(strconv.FormatFloat(v, 'f', -1, 64))
				case int:
					cell = strconv.Itoa(v)
				case int64:
					cell = strconv.FormatInt(v, 10)
				case json.Number:
					cell = v.String()
				default:
					if v != nil {
						cell = strings.TrimSpace(fmt.Sprintf("%v", v))
					}
				}
			}
			if cell == "" {
				cell = "-"
			}
			row = append(row, cell)
		}
		if len(row) > 0 {
			rows = append(rows, row)
		}
	}

	return TableTemplateData{
		Title:   title,
		Columns: cols,
		Rows:    rows,
		Notes:   notes,
		Has:     len(cols) > 0 && len(rows) > 0,
	}
}

func normalizeChart(chart *ChartData, theme Theme) ChartTemplateData {
	if chart == nil {
		return ChartTemplateData{}
	}

	chartType := strings.ToLower(strings.TrimSpace(chart.Type))
	if chartType == "" {
		chartType = "bar"
	}
	if chartType != "bar" && chartType != "line" && chartType != "pie" {
		return ChartTemplateData{}
	}

	title := strings.TrimSpace(chart.Title)
	notes := strings.TrimSpace(chart.Notes)
	categories := make([]string, 0, len(chart.Categories))
	for _, c := range chart.Categories {
		c = strings.TrimSpace(c)
		if c != "" {
			categories = append(categories, c)
		}
	}

	series := make([]ChartSeries, 0, len(chart.Series))
	maxLen := 0
	for _, s := range chart.Series {
		if len(s.Values) > maxLen {
			maxLen = len(s.Values)
		}
	}
	if len(categories) == 0 && maxLen > 0 {
		for i := 0; i < maxLen; i++ {
			categories = append(categories, fmt.Sprintf("%d", i+1))
		}
	}
	for _, s := range chart.Series {
		values := make([]float64, 0, len(categories))
		for i := 0; i < len(categories); i++ {
			if i < len(s.Values) {
				values = append(values, s.Values[i])
			} else {
				values = append(values, 0)
			}
		}
		series = append(series, ChartSeries{
			Name:   strings.TrimSpace(s.Name),
			Values: values,
		})
	}

	if len(series) == 0 || len(categories) == 0 {
		return ChartTemplateData{}
	}

	colors := []string{theme.AccentColor, theme.PrimaryColor, "#94A3B8", "#38BDF8", "#F59E0B"}

	data := ChartTemplateData{
		Type:       chartType,
		Title:      title,
		Categories: categories,
		Series:     series,
		Colors:     colors,
		Notes:      notes,
		Has:        true,
	}

	switch chartType {
	case "bar":
		data.Bars = buildBarChart(series, categories, colors)
	case "line":
		data.Points, data.LinePaths = buildLineChart(series, categories)
	case "pie":
		data.Slices = buildPieChart(series, colors)
	}

	return data
}

func buildBarChart(series []ChartSeries, categories []string, colors []string) []ChartBar {
	const (
		w = 1000.0
		h = 420.0
	)
	maxVal := 0.0
	for _, s := range series {
		for _, v := range s.Values {
			if v > maxVal {
				maxVal = v
			}
		}
	}
	if maxVal <= 0 {
		maxVal = 1
	}

	groupCount := float64(len(categories))
	seriesCount := float64(len(series))
	groupWidth := w / groupCount
	barWidth := groupWidth / (seriesCount + 1)

	var bars []ChartBar
	for i, cat := range categories {
		for j, s := range series {
			value := s.Values[i]
			barHeight := (value / maxVal) * h
			x := (float64(i) * groupWidth) + (float64(j) * barWidth) + (barWidth * 0.2)
			y := h - barHeight
			bars = append(bars, ChartBar{
				X:     x,
				Y:     y,
				W:     barWidth * 0.6,
				H:     barHeight,
				Value: value,
				Color: colors[j%len(colors)],
				Label: cat,
			})
		}
	}
	return bars
}

func buildLineChart(series []ChartSeries, categories []string) ([][]ChartPoint, []string) {
	const (
		w = 1000.0
		h = 420.0
	)
	maxVal := 0.0
	for _, s := range series {
		for _, v := range s.Values {
			if v > maxVal {
				maxVal = v
			}
		}
	}
	if maxVal <= 0 {
		maxVal = 1
	}
	step := 0.0
	if len(categories) > 1 {
		step = w / float64(len(categories)-1)
	}

	points := make([][]ChartPoint, 0, len(series))
	paths := make([]string, 0, len(series))

	for _, s := range series {
		pts := make([]ChartPoint, 0, len(categories))
		for i := range categories {
			x := float64(i) * step
			y := h - ((s.Values[i] / maxVal) * h)
			pts = append(pts, ChartPoint{X: x, Y: y, Value: s.Values[i]})
		}
		points = append(points, pts)
		path := buildLinePath(pts)
		paths = append(paths, path)
	}

	return points, paths
}

func buildLinePath(points []ChartPoint) string {
	if len(points) == 0 {
		return ""
	}
	var b strings.Builder
	for i, p := range points {
		if i == 0 {
			fmt.Fprintf(&b, "M %.1f %.1f", p.X, p.Y)
		} else {
			fmt.Fprintf(&b, " L %.1f %.1f", p.X, p.Y)
		}
	}
	return b.String()
}

func buildPieChart(series []ChartSeries, colors []string) []ChartSlice {
	if len(series) == 0 {
		return nil
	}
	values := series[0].Values
	total := 0.0
	for _, v := range values {
		if v > 0 {
			total += v
		}
	}
	if total == 0 {
		total = 1
	}

	cx, cy := 240.0, 240.0
	r := 200.0
	start := 0.0
	var slices []ChartSlice
	for i, v := range values {
		angle := (v / total) * 2 * math.Pi
		end := start + angle
		large := 0
		if angle > math.Pi {
			large = 1
		}
		x1 := cx + r*math.Cos(start)
		y1 := cy + r*math.Sin(start)
		x2 := cx + r*math.Cos(end)
		y2 := cy + r*math.Sin(end)
		path := fmt.Sprintf("M %.1f %.1f L %.1f %.1f A %.1f %.1f 0 %d 1 %.1f %.1f Z", cx, cy, x1, y1, r, r, large, x2, y2)
		slices = append(slices, ChartSlice{
			Path:  path,
			Color: colors[i%len(colors)],
			Label: fmt.Sprintf("%d", i+1),
		})
		start = end
	}
	return slices
}

func extractBulletsLabel(slide SlidePlan) string {
	// Try to extract a label from notes or use a default
	if strings.TrimSpace(slide.Notes) != "" {
		// Use first line of notes as label if short enough
		lines := strings.SplitN(slide.Notes, "\n", 2)
		if len(lines[0]) <= 30 {
			return strings.TrimSpace(lines[0])
		}
	}
	return "Overview"
}

func extractStats(slide SlidePlan) []StatItem {
	// Parse stats from slide data if available
	// Stats can be embedded in slide.Stats field or parsed from notes
	if slide.Stats != nil && len(slide.Stats) > 0 {
		stats := make([]StatItem, 0, len(slide.Stats))
		for _, s := range slide.Stats {
			if s.Value != "" {
				stats = append(stats, StatItem{
					Value: s.Value,
					Label: s.Label,
				})
			}
		}
		if len(stats) > 0 {
			return stats
		}
	}

	// Try to extract stats from bullets that look like metrics
	// e.g., "47% growth rate" -> Value: "47%", Label: "growth rate"
	return nil
}
