package main

import "github.com/charmbracelet/lipgloss"

// Base colors that don't change with themes
var (
	successColor = lipgloss.Color("#00FF88") // Green
	errorColor   = lipgloss.Color("#FF4757") // Red
	dimColor     = lipgloss.Color("#6B7280") // Gray
	bgDark       = lipgloss.Color("#0D1117") // Dark background
	bgMedium     = lipgloss.Color("#161B22") // Medium background
)

type Styles struct {
	// Layout
	App           lipgloss.Style
	LoginBox      lipgloss.Style
	ChatContainer lipgloss.Style

	// Components
	Logo        lipgloss.Style
	Title       lipgloss.Style
	Subtitle    lipgloss.Style
	Window      lipgloss.Style
	Header      lipgloss.Style
	InputField  lipgloss.Style
	InputFocus  lipgloss.Style
	Button      lipgloss.Style
	ButtonFocus lipgloss.Style

	// Message elements
	User       lipgloss.Style
	DateTime   lipgloss.Style
	Msg        lipgloss.Style
	PrivMsg    lipgloss.Style // Private message style
	Text       lipgloss.Style
	Error      lipgloss.Style
	Success    lipgloss.Style
	Hint       lipgloss.Style
	Separator  lipgloss.Style
	StatusBar  lipgloss.Style
	OnlineUser lipgloss.Style

	// Theme colors for use in tui_model.go
	PrimaryColor   lipgloss.Color
	SecondaryColor lipgloss.Color
	PrivMsgColor   lipgloss.Color
}

func InitStyles(cfg Config) Styles {
	// Use theme colors from config
	primaryColor := lipgloss.Color(cfg.WindowColor)
	secondaryColor := lipgloss.Color(cfg.UserColor)

	return Styles{
		// Store colors for external use
		PrimaryColor:   primaryColor,
		SecondaryColor: secondaryColor,

		// Full app container
		App: lipgloss.NewStyle().
			Background(bgDark),

		// Login box - centered with double border
		LoginBox: lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(primaryColor).
			Padding(2, 4).
			Margin(1, 2).
			Width(60),

		// Chat container
		ChatContainer: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(0, 1),

		// ASCII Logo style
		Logo: lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			Align(lipgloss.Center),

		// Title
		Title: lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true).
			MarginBottom(1).
			Align(lipgloss.Center),

		// Subtitle
		Subtitle: lipgloss.NewStyle().
			Foreground(dimColor).
			Italic(true).
			Align(lipgloss.Center),

		// Window with gradient border
		Window: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(1, 2),

		// Header bar - uses theme primary color
		Header: lipgloss.NewStyle().
			Background(primaryColor).
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true).
			Padding(0, 2).
			MarginBottom(1),

		// Input field - unfocused
		InputField: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(dimColor).
			Padding(0, 1).
			MarginBottom(1),

		// Input field - focused with glow effect
		InputFocus: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(secondaryColor).
			Padding(0, 1).
			MarginBottom(1),

		// Button - normal
		Button: lipgloss.NewStyle().
			Background(dimColor).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 3).
			MarginTop(1).
			Bold(true),

		// Button - focused/hover
		ButtonFocus: lipgloss.NewStyle().
			Background(primaryColor).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 3).
			MarginTop(1).
			Bold(true),

		// User name in messages
		User: lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true),

		// Timestamp
		DateTime: lipgloss.NewStyle().
			Foreground(lipgloss.Color(cfg.DateTimeColor)).
			Italic(true),

		// Message content
		Msg: lipgloss.NewStyle().
			Foreground(lipgloss.Color(cfg.MsgColor)),

		// General text
		Text: lipgloss.NewStyle().
			Foreground(lipgloss.Color(cfg.TextColor)),

		// Error messages
		Error: lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true).
			Padding(0, 1),

		// Success messages
		Success: lipgloss.NewStyle().
			Foreground(successColor).
			Bold(true),

		// Hints/help text
		Hint: lipgloss.NewStyle().
			Foreground(dimColor).
			Italic(true).
			MarginTop(1),

		// Separator line
		Separator: lipgloss.NewStyle().
			Foreground(dimColor),

		// Status bar
		StatusBar: lipgloss.NewStyle().
			Background(bgMedium).
			Foreground(successColor).
			Padding(0, 1),

		// Online user indicator
		OnlineUser: lipgloss.NewStyle().
			Foreground(successColor).
			Bold(true),

		// Private message style
		PrivMsg: lipgloss.NewStyle().
			Foreground(lipgloss.Color(cfg.PrivMsgColor)).
			Italic(true),

		// Private message color for use in tui_model.go
		PrivMsgColor: lipgloss.Color(cfg.PrivMsgColor),
	}
}

// Animation frames for spinner
var SpinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Typing indicator frames
var TypingFrames = []string{".", "..", "..."}

// Connection status icons
const (
	IconConnected    = "●"
	IconDisconnected = "○"
	IconConnecting   = "◌"
	IconSend         = "➤"
	IconUser         = "👤"
	IconLock         = "🔒"
	IconServer       = "🌐"
	IconChat         = "💬"
	IconOnline       = "🟢"
	IconOffline      = "🔴"
)
