package ui

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// TitleAnimations is the ordered list of available title animations.
// The last entry is "random" which picks a concrete animation each session.
var TitleAnimations = []string{
	"kitt", "rainbow", "wave", "typing", "glitch",
	"pulse", "spotlight", "border", "matrix", "bounce",
	"random",
}

// concreteAnimations excludes "random".
func concreteAnimations() []string {
	return TitleAnimations[:len(TitleAnimations)-1]
}

// ResolveAnimation turns a raw animation name into a concrete one.
// If raw is "random" it picks a random concrete animation.
// Unknown names fall back to "kitt".
func ResolveAnimation(raw string) string {
	switch raw {
	case "random":
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		list := concreteAnimations()
		return list[r.Intn(len(list))]
	case "kitt", "rainbow", "wave", "typing", "glitch",
		"pulse", "spotlight", "border", "matrix", "bounce":
		return raw
	default:
		return "kitt"
	}
}

// renderTitleAnimation dispatches to the requested animation.
func renderTitleAnimation(name string, frame int, text string) string {
	switch name {
	case "rainbow":
		return animRainbow(frame, text)
	case "wave":
		return animWave(frame, text)
	case "typing":
		return animTyping(frame, text)
	case "glitch":
		return animGlitch(frame, text)
	case "pulse":
		return animPulse(frame, text)
	case "spotlight":
		return animSpotlight(frame, text)
	case "border":
		return animBorder(frame, text)
	case "matrix":
		return animMatrix(frame, text)
	case "bounce":
		return animBounce(frame, text)
	case "kitt":
		return animKitt(frame, text)
	default:
		return animKitt(frame, text)
	}
}

// ── Helpers ──────────────────────────────────────────────────────────────────

var animDim = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

func animFg(c string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(c))
}

// rainbow palette (shifting)
func rainbowColor(i, frame int) string {
	colors := []string{
		"#ff0000", "#ff7f00", "#ffff00", "#00ff00",
		"#0000ff", "#4b0082", "#9400d3",
	}
	idx := (i + frame/2) % len(colors)
	if idx < 0 {
		idx += len(colors)
	}
	return colors[idx]
}

// ── 1. K.I.T.T. ──────────────────────────────────────────────────────────────

func animKitt(frame int, text string) string {
	runes := []rune(text)
	n := len(runes)
	if n == 0 {
		return ""
	}
	bright := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff4444")).Bold(true)
	tail1 := lipgloss.NewStyle().Foreground(lipgloss.Color("#aa1111"))
	tail2 := lipgloss.NewStyle().Foreground(lipgloss.Color("#661111"))

	// Defaults matching old behaviour: speed=80ms, pause=1000ms => ~13 ticks pause.
	pause := 13
	cycle := 2*n - 2 // forward + backward excluding duplicate end points
	total := cycle + pause
	f := frame % total

	var pos, dir int
	if f >= cycle {
		pos = 0
		dir = 1
	} else if f < n {
		pos = f
		dir = 1
	} else {
		pos = cycle - f
		dir = -1
	}

	var s strings.Builder
	for i, ch := range runes {
		c := string(ch)
		isTail1 := (dir > 0 && i == pos-1) || (dir < 0 && i == pos+1)
		isTail2 := (dir > 0 && i == pos-2) || (dir < 0 && i == pos+2)
		switch {
		case f >= cycle:
			s.WriteString(animDim.Render(c))
		case i == pos:
			s.WriteString(bright.Render(c))
		case isTail1:
			s.WriteString(tail1.Render(c))
		case isTail2:
			s.WriteString(tail2.Render(c))
		default:
			s.WriteString(animDim.Render(c))
		}
	}
	return s.String()
}

// ── 2. Rainbow ───────────────────────────────────────────────────────────────

func animRainbow(frame int, text string) string {
	runes := []rune(text)
	if len(runes) == 0 {
		return ""
	}
	var s strings.Builder
	for i, ch := range runes {
		c := rainbowColor(i, frame)
		s.WriteString(animFg(c).Render(string(ch)))
	}
	return s.String()
}

// ── 3. Wave ──────────────────────────────────────────────────────────────────

func animWave(frame int, text string) string {
	runes := []rune(text)
	n := len(runes)
	if n == 0 {
		return ""
	}
	var s strings.Builder
	for i, ch := range runes {
		phase := math.Sin(float64(i)*0.5 + float64(frame)*0.15)
		intensity := (phase + 1.0) / 2.0 // 0..1
		// Interpolate between dim grey and bright cyan.
		v := int(100 + intensity*155)
		c := lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", 0, v, v))
		s.WriteString(lipgloss.NewStyle().Foreground(c).Render(string(ch)))
	}
	return s.String()
}

// ── 4. Typing ────────────────────────────────────────────────────────────────

func animTyping(frame int, text string) string {
	runes := []rune(text)
	n := len(runes)
	if n == 0 {
		return ""
	}
	cycle := n + 6 // type all + brief pause
	f := frame % cycle
	var s strings.Builder
	for i, ch := range runes {
		c := string(ch)
		if i < f && i < n {
			if i == f-1 {
				s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff")).Bold(true).Render(c))
			} else {
				s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#bbbbbb")).Render(c))
			}
		} else {
			s.WriteString(animDim.Render(c))
		}
	}
	return s.String()
}

// ── 5. Glitch ────────────────────────────────────────────────────────────────

func animGlitch(frame int, text string) string {
	runes := []rune(text)
	n := len(runes)
	if n == 0 {
		return ""
	}
	// Deterministic pseudo-random based on frame so it doesn't flicker madly.
	seed := int64(frame/3) * 12345
	r := rand.New(rand.NewSource(seed))
	var s strings.Builder
	for _, ch := range runes {
		c := string(ch)
		v := r.Intn(100)
		switch {
		case v > 92:
			s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff")).Bold(true).Render(c))
		case v > 80:
			s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#ff5555")).Render(c))
		case v > 65:
			s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#ffaa00")).Render(c))
		default:
			s.WriteString(animDim.Render(c))
		}
	}
	return s.String()
}

// ── 6. Pulse ─────────────────────────────────────────────────────────────────

func animPulse(frame int, text string) string {
	runes := []rune(text)
	if len(runes) == 0 {
		return ""
	}
	phase := math.Sin(float64(frame) * 0.1)
	intensity := (phase + 1.0) / 2.0
	v := int(80 + intensity*175)
	c := lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", v, v, v))
	style := lipgloss.NewStyle().Foreground(c).Bold(intensity > 0.5)
	return style.Render(text)
}

// ── 7. Spotlight ─────────────────────────────────────────────────────────────

func animSpotlight(frame int, text string) string {
	runes := []rune(text)
	n := len(runes)
	if n == 0 {
		return ""
	}
	cycle := 2*n - 2
	f := frame % cycle
	pos := f
	if f >= n {
		pos = cycle - f
	}
	bright := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff")).Bold(true)
	var s strings.Builder
	for i, ch := range runes {
		c := string(ch)
		if i == pos {
			s.WriteString(bright.Render(c))
		} else {
			s.WriteString(animDim.Render(c))
		}
	}
	return s.String()
}

// ── 8. Border ────────────────────────────────────────────────────────────────

func animBorder(frame int, text string) string {
	runes := []rune(text)
	n := len(runes)
	if n == 0 {
		return ""
	}
	cycle := n
	f := frame % cycle
	// Two spots moving from edges toward center.
	left := f
	right := n - 1 - f
	bright := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffaa00")).Bold(true)
	mid := lipgloss.NewStyle().Foreground(lipgloss.Color("#cc7700"))
	var s strings.Builder
	for i, ch := range runes {
		c := string(ch)
		switch {
		case i == left || i == right:
			s.WriteString(bright.Render(c))
		case (i == left+1 && left+1 < right) || (i == right-1 && right-1 > left):
			s.WriteString(mid.Render(c))
		default:
			s.WriteString(animDim.Render(c))
		}
	}
	return s.String()
}

// ── 9. Matrix ────────────────────────────────────────────────────────────────

func animMatrix(frame int, text string) string {
	runes := []rune(text)
	n := len(runes)
	if n == 0 {
		return ""
	}
	// Deterministic flashes.
	seed := int64(frame/4) * 99991
	r := rand.New(rand.NewSource(seed))
	var s strings.Builder
	for i, ch := range runes {
		c := string(ch)
		v := r.Intn(100)
		// Also a travelling bright spot.
		spot := frame % (n * 2)
		spotPos := spot
		if spot >= n {
			spotPos = n*2 - 1 - spot
		}
		switch {
		case i == spotPos:
			s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff00")).Bold(true).Render(c))
		case v > 85:
			s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#00cc00")).Render(c))
		case v > 60:
			s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#008800")).Render(c))
		default:
			s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#003300")).Render(c))
		}
	}
	return s.String()
}

// ── 10. Bounce ───────────────────────────────────────────────────────────────

func animBounce(frame int, text string) string {
	runes := []rune(text)
	n := len(runes)
	if n == 0 {
		return ""
	}
	// A bright spot with a small trail, bounces off both ends immediately.
	cycle := 2*n - 2
	f := frame % cycle
	pos := f
	if f >= n {
		pos = cycle - f
	}
	bright := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff44ff")).Bold(true)
	tail1 := lipgloss.NewStyle().Foreground(lipgloss.Color("#aa11aa"))
	var s strings.Builder
	for i, ch := range runes {
		c := string(ch)
		dir := 1
		if f >= n {
			dir = -1
		}
		isTail1 := (dir > 0 && i == pos-1) || (dir < 0 && i == pos+1)
		switch {
		case i == pos:
			s.WriteString(bright.Render(c))
		case isTail1:
			s.WriteString(tail1.Render(c))
		default:
			s.WriteString(animDim.Render(c))
		}
	}
	return s.String()
}
