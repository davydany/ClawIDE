package color

import (
	"strings"
	"testing"
)

func TestParseHex(t *testing.T) {
	tests := []struct {
		input   string
		r, g, b uint8
		wantErr bool
	}{
		{"#ff0000", 255, 0, 0, false},
		{"#00ff00", 0, 255, 0, false},
		{"#0000ff", 0, 0, 255, false},
		{"#000000", 0, 0, 0, false},
		{"#ffffff", 255, 255, 255, false},
		{"#abcdef", 0xab, 0xcd, 0xef, false},
		{"invalid", 0, 0, 0, true},
		{"#fff", 0, 0, 0, true},
		{"", 0, 0, 0, true},
	}

	for _, tt := range tests {
		r, g, b, err := ParseHex(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseHex(%q) error = %v, wantErr = %v", tt.input, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && (r != tt.r || g != tt.g || b != tt.b) {
			t.Errorf("ParseHex(%q) = (%d,%d,%d), want (%d,%d,%d)", tt.input, r, g, b, tt.r, tt.g, tt.b)
		}
	}
}

func TestToHex(t *testing.T) {
	tests := []struct {
		r, g, b uint8
		want    string
	}{
		{255, 0, 0, "#ff0000"},
		{0, 255, 0, "#00ff00"},
		{0, 0, 255, "#0000ff"},
		{0, 0, 0, "#000000"},
		{255, 255, 255, "#ffffff"},
	}

	for _, tt := range tests {
		got := ToHex(tt.r, tt.g, tt.b)
		if got != tt.want {
			t.Errorf("ToHex(%d,%d,%d) = %q, want %q", tt.r, tt.g, tt.b, got, tt.want)
		}
	}
}

func TestRoundtrip(t *testing.T) {
	// Roundtrip: hex → RGB → HSL → RGB → hex should produce the same result.
	colors := []string{"#ff0000", "#00ff00", "#0000ff", "#3b82f6", "#a855f7", "#f97316"}
	for _, hex := range colors {
		r, g, b, err := ParseHex(hex)
		if err != nil {
			t.Fatalf("ParseHex(%q): %v", hex, err)
		}
		h, s, l := rgbToHSL(r, g, b)
		r2, g2, b2 := hslToRGB(h, s, l)
		got := ToHex(r2, g2, b2)
		if got != hex {
			t.Errorf("Roundtrip %q → RGB(%d,%d,%d) → HSL(%.2f,%.2f,%.2f) → RGB(%d,%d,%d) → %q",
				hex, r, g, b, h, s, l, r2, g2, b2, got)
		}
	}
}

func TestRoundtripGrays(t *testing.T) {
	// Grays have s=0, so hue is irrelevant. Roundtrip should preserve lightness.
	grays := []string{"#000000", "#808080", "#ffffff"}
	for _, hex := range grays {
		r, g, b, err := ParseHex(hex)
		if err != nil {
			t.Fatalf("ParseHex(%q): %v", hex, err)
		}
		h, s, l := rgbToHSL(r, g, b)
		r2, g2, b2 := hslToRGB(h, s, l)
		got := ToHex(r2, g2, b2)
		if got != hex {
			t.Errorf("Gray roundtrip %q → %q (HSL: %.2f,%.2f,%.2f)", hex, got, h, s, l)
		}
	}
}

func TestGenerateShades(t *testing.T) {
	shades, err := GenerateShades("#3b82f6", 8)
	if err != nil {
		t.Fatalf("GenerateShades: %v", err)
	}
	if len(shades) != 8 {
		t.Fatalf("expected 8 shades, got %d", len(shades))
	}

	// All shades should be valid hex colors
	seen := make(map[string]bool)
	for _, s := range shades {
		if len(s) != 7 || s[0] != '#' {
			t.Errorf("invalid shade format: %q", s)
		}
		seen[s] = true
	}

	// Should produce distinct shades
	if len(seen) < 6 {
		t.Errorf("expected mostly distinct shades, got %d unique out of 8: %v", len(seen), shades)
	}
}

func TestGenerateShadesEdgeCases(t *testing.T) {
	// Black
	shades, err := GenerateShades("#000000", 4)
	if err != nil {
		t.Fatalf("GenerateShades(black): %v", err)
	}
	if len(shades) != 4 {
		t.Fatalf("expected 4 shades, got %d", len(shades))
	}

	// White
	shades, err = GenerateShades("#ffffff", 4)
	if err != nil {
		t.Fatalf("GenerateShades(white): %v", err)
	}
	if len(shades) != 4 {
		t.Fatalf("expected 4 shades, got %d", len(shades))
	}

	// Zero shades
	shades, err = GenerateShades("#ff0000", 0)
	if err != nil {
		t.Fatalf("GenerateShades(n=0): %v", err)
	}
	if len(shades) != 0 {
		t.Fatalf("expected 0 shades, got %d", len(shades))
	}

	// Single shade
	shades, err = GenerateShades("#ff0000", 1)
	if err != nil {
		t.Fatalf("GenerateShades(n=1): %v", err)
	}
	if len(shades) != 1 {
		t.Fatalf("expected 1 shade, got %d", len(shades))
	}
}

func TestPickUnusedShade(t *testing.T) {
	base := "#3b82f6"

	// No used colors — should return first shade
	shade, err := PickUnusedShade(base, nil, 8)
	if err != nil {
		t.Fatalf("PickUnusedShade: %v", err)
	}
	if shade == "" || len(shade) != 7 {
		t.Errorf("invalid shade: %q", shade)
	}

	// Using some colors — should skip used ones
	shades, _ := GenerateShades(base, 8)
	used := []string{shades[0], shades[1]}
	picked, err := PickUnusedShade(base, used, 8)
	if err != nil {
		t.Fatalf("PickUnusedShade: %v", err)
	}
	if strings.ToLower(picked) == strings.ToLower(shades[0]) || strings.ToLower(picked) == strings.ToLower(shades[1]) {
		t.Errorf("PickUnusedShade returned a used shade: %q", picked)
	}

	// All shades used — should cycle
	picked, err = PickUnusedShade(base, shades, 8)
	if err != nil {
		t.Fatalf("PickUnusedShade (all used): %v", err)
	}
	if picked == "" {
		t.Error("PickUnusedShade should return a shade even when all are used")
	}
}
