package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Load configuration (ignore error to use defaults)
	cfg, err := LoadConfig("theme.conf")
	if err != nil {
		// Just print a warning if we can't read it, but proceed with defaults
		// fmt.Printf("Warning: Could not load theme.conf: %v\n", err)
	}

	model := initialModel(cfg)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
