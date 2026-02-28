package color

import (
	"fmt"
	"math"
	"strings"
)

// ParseHex parses a hex color string like "#rrggbb" into r, g, b components (0–255).
func ParseHex(hex string) (r, g, b uint8, err error) {
	hex = strings.TrimSpace(hex)
	if len(hex) != 7 || hex[0] != '#' {
		return 0, 0, 0, fmt.Errorf("invalid hex color %q: must be #rrggbb", hex)
	}
	var ri, gi, bi int
	_, err = fmt.Sscanf(hex[1:], "%02x%02x%02x", &ri, &gi, &bi)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid hex color %q: %w", hex, err)
	}
	return uint8(ri), uint8(gi), uint8(bi), nil
}

// ToHex converts r, g, b (0–255) to a hex string "#rrggbb".
func ToHex(r, g, b uint8) string {
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

// rgbToHSL converts RGB (0–255) to HSL where h is [0,360), s and l are [0,1].
func rgbToHSL(r, g, b uint8) (h, s, l float64) {
	rf := float64(r) / 255.0
	gf := float64(g) / 255.0
	bf := float64(b) / 255.0

	max := math.Max(rf, math.Max(gf, bf))
	min := math.Min(rf, math.Min(gf, bf))
	l = (max + min) / 2.0

	if max == min {
		// Achromatic
		return 0, 0, l
	}

	d := max - min
	if l > 0.5 {
		s = d / (2.0 - max - min)
	} else {
		s = d / (max + min)
	}

	switch max {
	case rf:
		h = (gf - bf) / d
		if gf < bf {
			h += 6
		}
	case gf:
		h = (bf-rf)/d + 2
	case bf:
		h = (rf-gf)/d + 4
	}
	h *= 60

	return h, s, l
}

// hslToRGB converts HSL (h [0,360), s [0,1], l [0,1]) to RGB (0–255).
func hslToRGB(h, s, l float64) (r, g, b uint8) {
	if s == 0 {
		v := uint8(math.Round(l * 255))
		return v, v, v
	}

	var q float64
	if l < 0.5 {
		q = l * (1 + s)
	} else {
		q = l + s - l*s
	}
	p := 2*l - q

	hNorm := h / 360.0

	toRGB := func(t float64) uint8 {
		if t < 0 {
			t += 1
		}
		if t > 1 {
			t -= 1
		}
		var v float64
		switch {
		case t < 1.0/6.0:
			v = p + (q-p)*6*t
		case t < 1.0/2.0:
			v = q
		case t < 2.0/3.0:
			v = p + (q-p)*(2.0/3.0-t)*6
		default:
			v = p
		}
		return uint8(math.Round(v * 255))
	}

	r = toRGB(hNorm + 1.0/3.0)
	g = toRGB(hNorm)
	b = toRGB(hNorm - 1.0/3.0)
	return
}

// GenerateShades produces n shades of the given base hex color by varying
// HSL lightness from 25% to 85%, keeping hue and saturation constant.
func GenerateShades(baseHex string, n int) ([]string, error) {
	if n <= 0 {
		return nil, nil
	}

	r, g, b, err := ParseHex(baseHex)
	if err != nil {
		return nil, err
	}

	h, s, _ := rgbToHSL(r, g, b)

	shades := make([]string, n)
	for i := 0; i < n; i++ {
		var l float64
		if n == 1 {
			l = 0.55 // mid-range
		} else {
			l = 0.25 + (0.60 * float64(i) / float64(n-1)) // 25% to 85%
		}
		sr, sg, sb := hslToRGB(h, s, l)
		shades[i] = ToHex(sr, sg, sb)
	}
	return shades, nil
}

// PickUnusedShade generates n shades from baseHex and returns the first one
// not in usedColors. If all shades are taken, it cycles from the beginning.
func PickUnusedShade(baseHex string, usedColors []string, n int) (string, error) {
	shades, err := GenerateShades(baseHex, n)
	if err != nil {
		return "", err
	}
	if len(shades) == 0 {
		return "", fmt.Errorf("no shades generated")
	}

	used := make(map[string]bool, len(usedColors))
	for _, c := range usedColors {
		used[strings.ToLower(c)] = true
	}

	for _, shade := range shades {
		if !used[strings.ToLower(shade)] {
			return shade, nil
		}
	}

	// All shades taken — cycle: return the shade at index len(usedColors) % n.
	return shades[len(usedColors)%len(shades)], nil
}
