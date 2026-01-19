package main

import (
	"bufio"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type Theme struct {
	Window   lipgloss.TerminalColor
	User     lipgloss.TerminalColor
	DateTime lipgloss.TerminalColor
	Msg      lipgloss.TerminalColor
	Text     lipgloss.TerminalColor
	Error    lipgloss.TerminalColor
	Button   lipgloss.TerminalColor
}

func DefaultTheme() Theme {
	return Theme{
		Window:   lipgloss.Color("#1a1a1a"),
		User:     lipgloss.Color("#00d7ff"),
		DateTime: lipgloss.Color("#5f5f5f"),
		Msg:      lipgloss.Color("#ffffff"),
		Text:     lipgloss.Color("#e4e4e4"),
		Error:    lipgloss.Color("#ff5f87"),
		Button:   lipgloss.Color("#00ff87"),
	}
}

func LoadTheme(path string) Theme {
	theme := DefaultTheme()
	file, err := os.Open(path)
	if err != nil {
		return theme
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

		key := strings.ToUpper(strings.TrimSpace(parts[0]))
		val := strings.TrimSpace(parts[1])

		switch key {
		case "WINDOW":
			theme.Window = lipgloss.Color(val)
		case "USER":
			theme.User = lipgloss.Color(val)
		case "DATETIME":
			theme.DateTime = lipgloss.Color(val)
		case "MSG":
			theme.Msg = lipgloss.Color(val)
		case "TEXT":
			theme.Text = lipgloss.Color(val)
		case "ERROR":
			theme.Error = lipgloss.Color(val)
		case "BUTTON":
			theme.Button = lipgloss.Color(val)
		}
	}

	return theme
}
