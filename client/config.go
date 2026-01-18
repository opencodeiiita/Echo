package main

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

// Config holds the color configuration for the TUI
type Config struct {
	WindowColor   string
	UserColor     string
	DateTimeColor string
	MsgColor      string
	TextColor     string
}

// Preset themes - select by number in theme.conf
var themePresets = map[int]Config{
	// 1: Default (Purple/Cyan)
	1: {
		WindowColor:   "#7D56F4",
		UserColor:     "#00D9FF",
		DateTimeColor: "#6B7280",
		MsgColor:      "#E5E7EB",
		TextColor:     "#FFFFFF",
	},
	// 2: Cyberpunk (Magenta/Cyan)
	2: {
		WindowColor:   "#FF00FF",
		UserColor:     "#00FFFF",
		DateTimeColor: "#FF6B9D",
		MsgColor:      "#FFFFFF",
		TextColor:     "#E0E0E0",
	},
	// 3: Forest (Green)
	3: {
		WindowColor:   "#2ECC71",
		UserColor:     "#F1C40F",
		DateTimeColor: "#7F8C8D",
		MsgColor:      "#ECF0F1",
		TextColor:     "#BDC3C7",
	},
	// 4: Ocean (Blue)
	4: {
		WindowColor:   "#3498DB",
		UserColor:     "#2980B9",
		DateTimeColor: "#95A5A6",
		MsgColor:      "#ECF0F1",
		TextColor:     "#BDC3C7",
	},
	// 5: Sunset (Orange/Red)
	5: {
		WindowColor:   "#E74C3C",
		UserColor:     "#F39C12",
		DateTimeColor: "#95A5A6",
		MsgColor:      "#FFEAA7",
		TextColor:     "#DFE6E9",
	},
	// 6: Dracula (Popular dark theme)
	6: {
		WindowColor:   "#BD93F9",
		UserColor:     "#50FA7B",
		DateTimeColor: "#6272A4",
		MsgColor:      "#F8F8F2",
		TextColor:     "#F8F8F2",
	},
	// 7: Nord (Cool arctic theme)
	7: {
		WindowColor:   "#88C0D0",
		UserColor:     "#A3BE8C",
		DateTimeColor: "#4C566A",
		MsgColor:      "#ECEFF4",
		TextColor:     "#D8DEE9",
	},
	// 8: Monokai (Classic editor theme)
	8: {
		WindowColor:   "#F92672",
		UserColor:     "#A6E22E",
		DateTimeColor: "#75715E",
		MsgColor:      "#F8F8F2",
		TextColor:     "#F8F8F2",
	},
	// 9: Gruvbox (Retro warm theme)
	9: {
		WindowColor:   "#FE8019",
		UserColor:     "#B8BB26",
		DateTimeColor: "#928374",
		MsgColor:      "#EBDBB2",
		TextColor:     "#FBF1C7",
	},
	// 10: Tokyo Night (Modern VSCode theme)
	10: {
		WindowColor:   "#7AA2F7",
		UserColor:     "#9ECE6A",
		DateTimeColor: "#565F89",
		MsgColor:      "#C0CAF5",
		TextColor:     "#A9B1D6",
	},
	// 11: One Dark (Popular VS Code theme)
	11: {
		WindowColor:   "#61AFEF",
		UserColor:     "#98C379",
		DateTimeColor: "#5C6370",
		MsgColor:      "#ABB2BF",
		TextColor:     "#E5C07B",
	},
	// 12: Material Dark (Google Material Design)
	12: {
		WindowColor:   "#82AAFF",
		UserColor:     "#C792EA",
		DateTimeColor: "#546E7A",
		MsgColor:      "#EEFFFF",
		TextColor:     "#B2CCD6",
	},
	// 13: Catppuccin Mocha (Popular modern theme)
	13: {
		WindowColor:   "#89B4FA",
		UserColor:     "#A6E3A1",
		DateTimeColor: "#6C7086",
		MsgColor:      "#CDD6F4",
		TextColor:     "#F9E2AF",
	},
	// 14: Solarized Dark (Classic terminal theme)
	14: {
		WindowColor:   "#268BD2",
		UserColor:     "#859900",
		DateTimeColor: "#586E75",
		MsgColor:      "#EEE8D5",
		TextColor:     "#FDF6E3",
	},
	// 15: Ayu Dark (Elegant dark theme)
	15: {
		WindowColor:   "#FFCC66",
		UserColor:     "#5CCFE6",
		DateTimeColor: "#4D5566",
		MsgColor:      "#E6E1CF",
		TextColor:     "#F8F8F2",
	},
}

// DefaultConfig returns the default configuration
func DefaultConfig() Config {
	return themePresets[1] // Default theme
}

// LoadConfig reads the configuration from a file
func LoadConfig(path string) (Config, error) {
	config := DefaultConfig()

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return config, nil // Return default if file doesn't exist
		}
		return config, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "THEME":
			// Load preset theme by number
			themeNum, err := strconv.Atoi(value)
			if err == nil {
				if preset, exists := themePresets[themeNum]; exists {
					config = preset
				}
			}
		case "WINDOW":
			config.WindowColor = value
		case "USER":
			config.UserColor = value
		case "DATETIME":
			config.DateTimeColor = value
		case "MSG":
			config.MsgColor = value
		case "TEXT":
			config.TextColor = value
		}
	}

	if err := scanner.Err(); err != nil {
		return config, err
	}

	return config, nil
}
