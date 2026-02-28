package banner

import (
	"fmt"
	"net"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
)

// ANSI color codes
const (
	cyan   = "\033[36m"
	red    = "\033[31m"
	gray   = "\033[90m"
	bold   = "\033[1m"
	reset  = "\033[0m"
	white  = "\033[97m"
	yellow = "\033[33m"
)

// Happy crab — displayed at startup (Tamagotchi: alive and thriving)
var happyCrab = []string{
	`            ▗▖   ▗▖`,
	`     ▗▄▖   ▐▙▖ ▗▟▌   ▗▄▖`,
	`    ▐▛▀▜▖ ▗▟████▙▖  ▗▜▀▛▌`,
	`    ▐▌  ▜▟██████████▛▘  ▐▌`,
	`     ▜▖ ▐████████████▌ ▗▛`,
	`      ▀▄▐██ ▀▌▐▀ ██▌▄▀`,
	`        ▐██▌  ▄  ▐██▌`,
	`        ▐████▀▀▀████▌`,
	`        ▗▟██████████▙▖`,
	`       ▟▛▘ ▐██████▌ ▀▜▙`,
	`      ▟▘  ▗▛▘    ▀▜▖  ▝▙`,
}

// Dead crab — displayed at shutdown (Tamagotchi: game over)
var deadCrab = []string{
	`            ▗▖   ▗▖`,
	`     ▗▄▖   ▐▙▖ ▗▟▌   ▗▄▖`,
	`    ▐▛▀▜▖ ▗▟████▙▖  ▗▜▀▛▌`,
	`    ▐▌  ▜▟██████████▛▘  ▐▌`,
	`     ▜▖ ▐████████████▌ ▗▛`,
	`      ▀▄▐██ ╳▌▐╳ ██▌▄▀`,
	`        ▐██▌  ▄  ▐██▌`,
	`        ▐████▄▄▄████▌`,
	`        ▗▟██████████▙▖`,
	`       ▟▛▘ ▐██████▌ ▀▜▙`,
	`      ▟▘  ▗▛▘    ▀▜▖  ▝▙`,
}

// ASCII art generated using block characters.
// Each line is a row of the banner.
var asciiArt = []string{
	`  ██████╗██╗      █████╗ ██╗    ██╗██╗██████╗ ███████╗`,
	` ██╔════╝██║     ██╔══██╗██║    ██║██║██╔══██╗██╔════╝`,
	` ██║     ██║     ███████║██║ █╗ ██║██║██║  ██║█████╗  `,
	` ██║     ██║     ██╔══██║██║███╗██║██║██║  ██║██╔══╝  `,
	` ╚██████╗███████╗██║  ██║╚███╔███╔╝██║██████╔╝███████╗`,
	`  ╚═════╝╚══════╝╚═╝  ╚═╝ ╚══╝╚══╝ ╚═╝╚═════╝ ╚══════╝`,
}

// Print renders the startup banner with ASCII art, server URL, and QR code.
func Print(host string, port int, versionStr string) {
	// Determine the accessible URL
	displayHost := host
	if host == "0.0.0.0" || host == "::" || host == "" {
		displayHost = "localhost"
	}
	localURL := fmt.Sprintf("http://%s:%d", displayHost, port)

	// Find LAN IP for QR code (phones/tablets need the real IP, not localhost)
	lanIP := detectLANIP()
	lanURL := ""
	if lanIP != "" && (host == "0.0.0.0" || host == "::" || host == "") {
		lanURL = fmt.Sprintf("http://%s:%d", lanIP, port)
	}

	fmt.Println()

	// Print happy crab mascot
	for _, line := range happyCrab {
		fmt.Printf("  %s%s%s\n", cyan, line, reset)
	}
	fmt.Println()

	// Print ASCII art
	for _, line := range asciiArt {
		fmt.Printf("  %s%s%s\n", cyan, line, reset)
	}

	fmt.Println()

	// Version line
	fmt.Printf("  %s%s%s\n", gray, versionStr, reset)
	fmt.Println()

	// URL info
	fmt.Printf("  %s%sLocal:%s   %s%s\n", bold, white, reset, localURL, reset)
	if lanURL != "" {
		fmt.Printf("  %s%sNetwork:%s %s%s\n", bold, white, reset, lanURL, reset)
	}

	fmt.Println()

	// QR code — prefer LAN URL so mobile devices can reach it
	qrTarget := lanURL
	if qrTarget == "" {
		qrTarget = localURL
	}
	qrStr := renderTerminalQR(qrTarget)
	if qrStr != "" {
		fmt.Printf("  %s%sScan to open on your phone or tablet:%s\n\n", bold, yellow, reset)
		// Indent each QR line
		for _, line := range strings.Split(qrStr, "\n") {
			if line != "" {
				fmt.Printf("    %s\n", line)
			}
		}
		fmt.Println()
	}
}

// PrintShutdown renders the dead crab and a farewell message.
func PrintShutdown() {
	fmt.Println()
	for _, line := range deadCrab {
		fmt.Printf("  %s%s%s\n", red, line, reset)
	}
	fmt.Println()
	fmt.Printf("  %s%sEt tu, Brute?%s\n", red, bold, reset)
	fmt.Printf("  %sThe claw has been... declawed.%s\n", red, reset)
	fmt.Printf("  %sClawIDE signing off. Snip snip.%s\n", red, reset)
	fmt.Println()
}

// renderTerminalQR generates a compact QR code string using Unicode half-block
// characters. Each character cell represents 2 vertical modules, so the output
// is half the height of a naive renderer.
func renderTerminalQR(url string) string {
	qr, err := qrcode.New(url, qrcode.Medium)
	if err != nil {
		return ""
	}
	qr.DisableBorder = false
	bitmap := qr.Bitmap()
	rows := len(bitmap)
	if rows == 0 {
		return ""
	}
	cols := len(bitmap[0])

	var sb strings.Builder

	// Process two rows at a time using Unicode half-block technique:
	// ▀ = top filled, bottom empty
	// ▄ = top empty, bottom filled
	// █ = both filled
	// ' ' = both empty
	//
	// QR convention: true = black (dark module), false = white (light).
	// In terminal: dark modules → dark block, light modules → space.
	// We invert for dark-background terminals: dark module = █, light = ' '.
	for y := 0; y < rows; y += 2 {
		for x := 0; x < cols; x++ {
			top := bitmap[y][x]
			bot := false
			if y+1 < rows {
				bot = bitmap[y+1][x]
			}

			switch {
			case top && bot:
				sb.WriteRune('█')
			case top && !bot:
				sb.WriteRune('▀')
			case !top && bot:
				sb.WriteRune('▄')
			default:
				sb.WriteRune(' ')
			}
		}
		sb.WriteRune('\n')
	}

	return sb.String()
}

// detectLANIP returns the first non-loopback IPv4 address found on the machine,
// which is typically the address other devices on the same LAN can reach.
func detectLANIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		ip := ipNet.IP
		if ip.IsLoopback() || ip.To4() == nil {
			continue
		}
		return ip.String()
	}
	return ""
}
