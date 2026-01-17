package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gorilla/websocket"
)

// ASCII Art Logo with gradient effect
const echoLogo = `
███████╗ ██████╗██╗  ██╗ ██████╗ 
██╔════╝██╔════╝██║  ██║██╔═══██╗
█████╗  ██║     ███████║██║   ██║
██╔══╝  ██║     ██╔══██║██║   ██║
███████╗╚██████╗██║  ██║╚██████╔╝
╚══════╝ ╚═════╝╚═╝  ╚═╝ ╚═════╝ 
`

// Animated decorative frames
var borderFrames = []string{
	"╔══════════════════════════════════════════════════════╗",
	"╠══════════════════════════════════════════════════════╣",
	"║                                                      ║",
	"╚══════════════════════════════════════════════════════╝",
}

// Pulse animation frames for button
var pulseFrames = []string{"○", "◎", "●", "◉", "●", "◎"}

// Connection animation
var connectFrames = []string{
	"[    ]",
	"[=   ]",
	"[==  ]",
	"[=== ]",
	"[====]",
	"[ ===]",
	"[  ==]",
	"[   =]",
}

type sessionState int

const (
	loginView sessionState = iota
	connectingView
	chatView
)

type mainModel struct {
	state  sessionState
	styles Styles
	config Config

	// Login Inputs
	serverInput  textinput.Model
	userInput    textinput.Model
	passInput    textinput.Model
	focusIndex   int
	showPassword bool

	// Chat Components
	viewport viewport.Model
	msgInput textarea.Model
	messages []ChatMessage

	// Animation
	spinner       spinner.Model
	animFrame     int
	pulseFrame    int
	showCursor    bool
	chatStartTime time.Time // Track when chat started for adaptive animation

	// Connection
	conn     *websocket.Conn
	err      error
	width    int
	height   int
	username string // Store current username for message alignment

	// Status
	isConnecting bool
	statusMsg    string
}

// ChatMessage holds parsed message data for styled rendering
type ChatMessage struct {
	Timestamp string
	User      string
	Content   string
	IsSystem  bool
}

type errMsg error
type wsMsg string
type clearInputMsg struct{}
type tickMsg time.Time
type animTickMsg time.Time

func initialModel(cfg Config) mainModel {
	styles := InitStyles(cfg)

	// Server input
	s := textinput.New()
	s.Placeholder = "localhost:8080"
	s.Focus()
	s.Prompt = ""
	s.CharLimit = 64
	s.Width = 44
	s.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00D9FF"))
	s.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))

	// Username input
	u := textinput.New()
	u.Placeholder = "Enter username"
	u.Prompt = ""
	u.CharLimit = 32
	u.Width = 44
	u.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00D9FF"))
	u.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))

	// Password input
	p := textinput.New()
	p.Placeholder = "Enter password"
	p.Prompt = ""
	p.EchoMode = textinput.EchoPassword
	p.EchoCharacter = '●'
	p.CharLimit = 32
	p.Width = 44
	p.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00D9FF"))
	p.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))

	// Message input - textarea for multi-line support
	mi := textarea.New()
	mi.Placeholder = "Type your message..."
	mi.CharLimit = 500
	mi.SetWidth(60)
	mi.SetHeight(1) // Start with 1 line
	mi.ShowLineNumbers = false
	mi.FocusedStyle.CursorLine = lipgloss.NewStyle() // No line highlight
	mi.FocusedStyle.Base = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
	mi.BlurredStyle.Base = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))

	// Spinner for loading animation
	sp := spinner.New()
	sp.Spinner = spinner.MiniDot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))

	return mainModel{
		state:        loginView,
		styles:       styles,
		config:       cfg,
		serverInput:  s,
		userInput:    u,
		passInput:    p,
		msgInput:     mi,
		spinner:      sp,
		messages:     []ChatMessage{},
		viewport:     viewport.New(80, 20),
		showPassword: false,
		animFrame:    0,
		pulseFrame:   0,
	}
}

func (m mainModel) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.spinner.Tick, animTick())
}

// Animation tick command
func animTick() tea.Cmd {
	return tea.Tick(time.Millisecond*150, func(t time.Time) tea.Msg {
		return animTickMsg(t)
	})
}

func (m mainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			if m.conn != nil {
				m.conn.Close()
			}
			return m, tea.Quit

		case tea.KeyTab, tea.KeyShiftTab:
			if m.state == loginView {
				if msg.Type == tea.KeyShiftTab {
					m.focusIndex--
					if m.focusIndex < 0 {
						m.focusIndex = 4
					}
				} else {
					m.focusIndex = (m.focusIndex + 1) % 5
				}
				cmds = append(cmds, m.updateFocus())
			}

		case tea.KeySpace:
			// Space to toggle password visibility in login view
			if m.state == loginView && m.focusIndex == 3 {
				m.showPassword = !m.showPassword
				if m.showPassword {
					m.passInput.EchoMode = textinput.EchoNormal
				} else {
					m.passInput.EchoMode = textinput.EchoPassword
				}
				return m, nil
			}

		case tea.KeyEnter:
			// Check for Alt+Enter to insert newline in chat view
			// Note: Shift+Enter is not reliably detectable in terminals
			if m.state == chatView && msg.Alt {
				// Alt+Enter: insert newline
				m.msgInput.InsertString("\n")
				return m, nil
			}
			if m.state == loginView {
				if m.focusIndex == 3 {
					// Toggle password visibility
					m.showPassword = !m.showPassword
					if m.showPassword {
						m.passInput.EchoMode = textinput.EchoNormal
					} else {
						m.passInput.EchoMode = textinput.EchoPassword
					}
					return m, nil
				}
				if m.focusIndex == 4 {
					// Connect button pressed - switch to connecting view
					m.state = connectingView
					m.isConnecting = true
					return m, tea.Batch(m.connectCmd(), animTick())
				}
				// Move to next field
				m.focusIndex++
				if m.focusIndex > 4 {
					m.focusIndex = 0
				}
				cmds = append(cmds, m.updateFocus())
			} else if m.state == chatView {
				// Enter sends message in chat view
				if strings.TrimSpace(m.msgInput.Value()) != "" {
					msgToSend := m.msgInput.Value()
					m.msgInput.Reset()
					m.msgInput.SetHeight(1) // Reset to 1 line
					return m, m.sendMessageCmd(msgToSend)
				}
			}

		case tea.KeyCtrlU: // Ctrl+U to clear textarea (Unix-style)
			if m.state == chatView {
				m.msgInput.Reset()
				m.msgInput.SetHeight(1)
				return m, nil
			}

		case tea.KeyUp, tea.KeyDown:
			if m.state == chatView {
				// Let textarea handle up/down for cursor movement
				// Use PageUp/PageDown for scrolling viewport
			}
		case tea.KeyPgUp, tea.KeyPgDown:
			if m.state == chatView {
				m.viewport, cmd = m.viewport.Update(msg)
				cmds = append(cmds, cmd)
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		headerHeight := 3
		inputHeight := 6 // Allow up to 5 lines for input
		chatHeight := m.height - headerHeight - inputHeight - 4

		m.viewport.Width = msg.Width - 4
		m.viewport.Height = chatHeight
		m.msgInput.SetWidth(msg.Width - 10)

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case animTickMsg:
		// Update animation frames
		m.animFrame = (m.animFrame + 1) % len(connectFrames)
		m.pulseFrame = (m.pulseFrame + 1) % len(pulseFrames)
		cmds = append(cmds, animTick())

	case tickMsg:
		cmds = append(cmds, tickCmd())

	case errMsg:
		m.err = msg
		m.state = loginView
		m.isConnecting = false
		return m, nil

	case wsMsg:
		chatMsg := parseMessage(string(msg))
		m.messages = append(m.messages, chatMsg)
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		return m, waitForIncomingMessage(m.conn)

	case connectedMsg:
		m.state = chatView
		m.conn = msg.conn
		m.isConnecting = false
		m.err = nil
		m.username = m.userInput.Value() // Store username for message alignment
		m.chatStartTime = time.Now()     // Start tracking for adaptive animation

		// Add animated welcome message
		welcomeMsg := ChatMessage{
			Timestamp: time.Now().Format("15:04"),
			Content:   fmt.Sprintf("Successfully connected to %s", m.serverInput.Value()),
			IsSystem:  true,
		}
		m.messages = append(m.messages, welcomeMsg)

		userMsg := ChatMessage{
			Timestamp: time.Now().Format("15:04"),
			Content:   fmt.Sprintf("Welcome, %s! You can start chatting now.", m.userInput.Value()),
			IsSystem:  true,
		}
		m.messages = append(m.messages, userMsg)
		m.viewport.SetContent(m.renderMessages())

		m.msgInput.Focus()
		return m, tea.Batch(waitForIncomingMessage(m.conn), textarea.Blink, animTick())

	case clearInputMsg:
		m.msgInput.SetValue("")
	}

	// Handle inputs based on state
	if m.state == loginView {
		m.serverInput, cmd = m.serverInput.Update(msg)
		cmds = append(cmds, cmd)
		m.userInput, cmd = m.userInput.Update(msg)
		cmds = append(cmds, cmd)
		m.passInput, cmd = m.passInput.Update(msg)
		cmds = append(cmds, cmd)
	} else if m.state == chatView {
		m.msgInput, cmd = m.msgInput.Update(msg)
		cmds = append(cmds, cmd)
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)

		// Dynamic height adjustment for textarea (like WhatsApp)
		// Count lines in current input
		lines := strings.Count(m.msgInput.Value(), "\n") + 1
		if lines > 5 {
			lines = 5 // Max 5 lines
		}
		if lines < 1 {
			lines = 1
		}
		m.msgInput.SetHeight(lines)
	}

	return m, tea.Batch(cmds...)
}

func (m *mainModel) updateFocus() tea.Cmd {
	inputs := []*textinput.Model{&m.serverInput, &m.userInput, &m.passInput}

	for i, input := range inputs {
		if i == m.focusIndex {
			return input.Focus()
		}
		input.Blur()
	}
	return nil
}

func (m mainModel) View() string {
	switch m.state {
	case loginView:
		return m.loginView()
	case connectingView:
		return m.connectingView()
	default:
		return m.chatViewRender()
	}
}

func (m mainModel) loginView() string {
	var b strings.Builder

	// Animated decorative top border
	topDecor := lipgloss.NewStyle().
		Foreground(m.styles.PrimaryColor).
		Render("═══════════════════════════════════════════════")
	b.WriteString(topDecor + "\n")

	// Enhanced logo with gradient effect
	logoLines := strings.Split(echoLogo, "\n")
	gradientColors := []string{
		string(m.styles.PrimaryColor),
		string(m.styles.SecondaryColor),
		string(m.styles.PrimaryColor),
	}

	for idx, line := range logoLines {
		if line == "" {
			continue
		}
		// Create gradient effect by cycling colors
		colorIdx := (idx + m.animFrame) % len(gradientColors)
		lineStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(gradientColors[colorIdx])).
			Bold(true)
		b.WriteString(lineStyle.Render(line) + "\n")
	}

	// Animated subtitle with pulse
	pulse := pulseFrames[m.pulseFrame]
	subtitleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Italic(true).
		Align(lipgloss.Center).
		Width(50)
	b.WriteString(subtitleStyle.Render(pulse + " Terminal Chat Application " + pulse))
	b.WriteString("\n\n")

	// Input fields with animated focus indicator
	labelStyle := lipgloss.NewStyle().
		Foreground(m.styles.SecondaryColor).
		Bold(true).
		Width(50).
		Align(lipgloss.Center)

	// Server field
	serverIndicator := "  "
	if m.focusIndex == 0 {
		serverIndicator = "> "
	}
	serverLabel := labelStyle.Render(serverIndicator + IconServer + " Server")
	serverBorder := m.getInputStyle(0)
	b.WriteString(serverLabel + "\n")
	b.WriteString(serverBorder.Render(m.serverInput.View()))
	b.WriteString("\n\n")

	// Username field
	userIndicator := "  "
	if m.focusIndex == 1 {
		userIndicator = "> "
	}
	userLabel := labelStyle.Render(userIndicator + IconUser + " Username")
	userBorder := m.getInputStyle(1)
	b.WriteString(userLabel + "\n")
	b.WriteString(userBorder.Render(m.userInput.View()))
	b.WriteString("\n\n")

	// Password field
	passIndicator := "  "
	if m.focusIndex == 2 {
		passIndicator = "> "
	}
	passLabel := labelStyle.Render(passIndicator + IconLock + " Password")
	passBorder := m.getInputStyle(2)
	b.WriteString(passLabel + "\n")
	b.WriteString(passBorder.Render(m.passInput.View()))
	b.WriteString("\n")

	// Toggle button for password visibility
	toggleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Width(50).
		Align(lipgloss.Center)
	if m.focusIndex == 3 {
		toggleStyle = toggleStyle.
			Foreground(lipgloss.Color("#00D9FF")).
			Bold(true)
	}

	toggleText := "[ ] Show Password"
	if m.showPassword {
		toggleText = "[x] Hide Password"
	}
	b.WriteString(toggleStyle.Render(toggleText))
	b.WriteString("\n\n")

	// Fancy Connect Button with animation
	button := m.renderConnectButton()
	buttonContainer := lipgloss.NewStyle().
		Width(50).
		Align(lipgloss.Center)
	b.WriteString(buttonContainer.Render(button))
	b.WriteString("\n\n")

	// Enhanced error message with better styling
	if m.err != nil {
		errBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FF4757")).
			Foreground(lipgloss.Color("#FF4757")).
			Background(lipgloss.Color("#2D1B1B")).
			Bold(true).
			Width(50).
			Align(lipgloss.Center).
			Padding(0, 1)

		errIcon := "⚠"
		errText := fmt.Sprintf("%s %s", errIcon, m.err.Error())
		b.WriteString(errBox.Render(errText))
		b.WriteString("\n\n")
	}

	// Enhanced hints with better formatting
	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Italic(true).
		Width(50).
		Align(lipgloss.Center)
	hintText := fmt.Sprintf("Tab: Navigate | Enter: Select | Space: Toggle Password | Esc: Quit")
	b.WriteString(hintStyle.Render(hintText))

	// Bottom decorative border
	b.WriteString("\n" + topDecor)

	// Wrap in fancy login box
	loginBox := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(m.styles.PrimaryColor).
		Padding(1, 4).
		Align(lipgloss.Center)

	content := loginBox.Render(b.String())

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func (m mainModel) renderConnectButton() string {
	// Create fancy multi-line button with border art
	if m.focusIndex == 4 {
		// Focused state - animated and colorful
		pulse := pulseFrames[m.pulseFrame]

		topBorder := lipgloss.NewStyle().
			Foreground(m.styles.SecondaryColor).
			Render("╔════════════════════════╗")

		middleLine := lipgloss.NewStyle().
			Foreground(m.styles.SecondaryColor).
			Render("║") +
			lipgloss.NewStyle().
				Background(m.styles.PrimaryColor).
				Foreground(lipgloss.Color("#FFFFFF")).
				Bold(true).
				Padding(0, 1).
				Render(" "+pulse+" CONNECT "+pulse+" ") +
			lipgloss.NewStyle().
				Foreground(m.styles.SecondaryColor).
				Render("║")

		bottomBorder := lipgloss.NewStyle().
			Foreground(m.styles.SecondaryColor).
			Render("╚════════════════════════╝")

		return topBorder + "\n" + middleLine + "\n" + bottomBorder
	}

	// Unfocused state
	topBorder := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#3B4252")).
		Render("╔════════════════════════╗")

	middleLine := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#3B4252")).
		Render("║") +
		lipgloss.NewStyle().
			Background(lipgloss.Color("#3B4252")).
			Foreground(lipgloss.Color("#9CA3AF")).
			Padding(0, 1).
			Render("     CONNECT     ") +
		lipgloss.NewStyle().
			Foreground(lipgloss.Color("#3B4252")).
			Render("║")

	bottomBorder := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#3B4252")).
		Render("╚════════════════════════╝")

	return topBorder + "\n" + middleLine + "\n" + bottomBorder
}

func (m mainModel) connectingView() string {
	var b strings.Builder

	// Enhanced animated connecting view
	frame := connectFrames[m.animFrame]

	// Animated title with gradient
	titleStyle := lipgloss.NewStyle().
		Foreground(m.styles.PrimaryColor).
		Bold(true)
	title := titleStyle.Render("CONNECTING")

	// Enhanced animation with theme colors
	animationStyle := lipgloss.NewStyle().
		Foreground(m.styles.SecondaryColor).
		Bold(true)
	animation := animationStyle.Render(frame)

	spinnerView := m.spinner.View()

	// Enhanced connection info display
	infoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Italic(true)

	serverLabel := lipgloss.NewStyle().
		Foreground(m.styles.PrimaryColor).
		Bold(true).
		Render("Server:")
	serverValue := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#E5E7EB")).
		Render(m.serverInput.Value())

	userLabel := lipgloss.NewStyle().
		Foreground(m.styles.PrimaryColor).
		Bold(true).
		Render("User:")
	userValue := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#E5E7EB")).
		Render(m.userInput.Value())

	content := fmt.Sprintf(`
    %s %s

    %s

    %s
    %s %s
    %s %s
`,
		spinnerView,
		title,
		animation,
		infoStyle.Render("Establishing secure connection..."),
		serverLabel, serverValue,
		userLabel, userValue,
	)

	box := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(m.styles.PrimaryColor).
		Background(lipgloss.Color("#0D1117")).
		Padding(2, 4).
		Align(lipgloss.Center).
		Render(content)

	b.WriteString(box)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, b.String())
}

func (m mainModel) chatViewRender() string {
	var b strings.Builder

	// Clean, modern header with dark background
	accentColor := m.styles.PrimaryColor
	headerBg := lipgloss.Color("#0D1117") // Dark background
	headerFg := lipgloss.Color("#FFFFFF")
	dimFg := lipgloss.Color("#FFFFFFB0")

	// Calculate session duration
	sessionDuration := time.Since(m.chatStartTime)
	hours := int(sessionDuration.Hours())
	minutes := int(sessionDuration.Minutes()) % 60
	var sessionTime string
	if hours > 0 {
		sessionTime = fmt.Sprintf("%dh %dm", hours, minutes)
	} else {
		sessionTime = fmt.Sprintf("%dm", minutes)
	}

	// App name - ECHO in accent color (no background)
	appNameStyle := lipgloss.NewStyle().
		Foreground(accentColor).
		Bold(true)
	appName := " " + appNameStyle.Render("ECHO") + " "

	// Status indicator - center left
	statusDotStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00FF88")).
		Bold(true)
	statusTextStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00FF88"))

	onlineDot := statusDotStyle.Render(pulseFrames[m.pulseFrame])
	statusText := statusTextStyle.Render("ONLINE")
	statusSection := onlineDot + " " + statusText

	// User info - right side
	usernameStyle := lipgloss.NewStyle().
		Foreground(headerFg).
		Bold(true)
	username := usernameStyle.Render(m.userInput.Value())

	// Session info - far right
	sessionStyle := lipgloss.NewStyle().
		Foreground(dimFg)
	sessionInfo := " " + sessionStyle.Render("• "+sessionTime)

	// Build header with clean spacing
	leftPart := appName
	centerPart := statusSection
	rightPart := username + sessionInfo

	// Calculate widths
	leftWidth := lipgloss.Width(leftPart)
	centerWidth := lipgloss.Width(centerPart)
	rightWidth := lipgloss.Width(rightPart)

	// Calculate spacing for balanced layout
	totalContentWidth := leftWidth + centerWidth + rightWidth
	availableSpace := m.width - totalContentWidth - 4

	// Use fixed spacing between sections
	spacing1 := 3
	spacing2 := 3
	spacing3 := availableSpace - spacing1 - spacing2

	if spacing3 < 2 {
		spacing3 = 2
	}

	// Build header line
	headerLine := leftPart +
		strings.Repeat(" ", spacing1) +
		centerPart +
		strings.Repeat(" ", spacing2) +
		strings.Repeat(" ", spacing3) +
		rightPart

	// Apply header styling with dark background
	headerStyle := lipgloss.NewStyle().
		Background(headerBg).
		Foreground(headerFg).
		Width(m.width).
		Padding(0, 1)

	b.WriteString(headerStyle.Render(headerLine))
	b.WriteString("\n")

	// Accent color separator line
	separatorLine := lipgloss.NewStyle().
		Foreground(accentColor).
		Render(strings.Repeat("─", m.width))
	b.WriteString(separatorLine + "\n")

	// Chat viewport with enhanced styled border
	chatContent := m.viewport.View()

	// Add subtle scroll indicators
	var topIndicator string
	var bottomIndicator string

	// Check if scrolled from top
	if m.viewport.YOffset > 0 {
		topIndicator = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			Italic(true).
			Align(lipgloss.Center).
			Width(m.width - 4).
			Render("↑ More messages above")
	}

	// Check if not at bottom (simplified - if there's more content)
	totalLines := len(strings.Split(m.renderMessages(), "\n"))
	visibleLines := m.viewport.Height
	if m.viewport.YOffset+visibleLines < totalLines-2 {
		bottomIndicator = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			Italic(true).
			Align(lipgloss.Center).
			Width(m.width - 4).
			Render("↓ More messages below")
	}

	chatBorder := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#3B4252")).
		Width(m.width-4).
		Height(m.viewport.Height+2).
		Padding(0, 1)

	chatBox := chatBorder.Render(chatContent)

	// Add indicators if present
	if topIndicator != "" {
		b.WriteString(topIndicator + "\n")
	}
	b.WriteString(chatBox)
	if bottomIndicator != "" {
		b.WriteString("\n" + bottomIndicator)
	}
	b.WriteString("\n")

	// Enhanced adaptive input border animation with smoother transitions
	elapsedSeconds := time.Since(m.chatStartTime).Seconds()
	var inputBorderColor string
	if elapsedSeconds < 10 {
		// Fast animation - smooth color transition
		colors := []string{
			string(m.styles.SecondaryColor),
			string(m.styles.PrimaryColor),
			string(m.styles.SecondaryColor),
		}
		colorIdx := m.animFrame % len(colors)
		inputBorderColor = colors[colorIdx]
	} else {
		// Slow animation - subtle pulsing
		if m.animFrame%10 < 5 {
			inputBorderColor = string(m.styles.SecondaryColor)
		} else {
			inputBorderColor = "#3B4252"
		}
	}

	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(inputBorderColor)).
		Width(m.width-4).
		Padding(0, 1)

	b.WriteString(inputStyle.Render(m.msgInput.View()))
	b.WriteString("\n")

	// Enhanced footer with better styling
	footerContent := fmt.Sprintf(" [Enter] Send | [Alt+Enter] New Line | [PgUp/PgDn] Scroll | [Ctrl+U] Clear | [Esc] Quit")
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Italic(true).
		Background(lipgloss.Color("#0D1117")).
		Padding(0, 1)
	b.WriteString(footerStyle.Render(footerContent))

	return b.String()
}

func (m mainModel) getInputStyle(index int) lipgloss.Style {
	baseWidth := 50
	if m.focusIndex == index {
		// Enhanced animated glow effect with theme colors
		glowColors := []string{
			string(m.styles.SecondaryColor),
			string(m.styles.PrimaryColor),
			string(m.styles.SecondaryColor),
		}
		colorIdx := m.animFrame % len(glowColors)
		return lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(glowColors[colorIdx])).
			Background(lipgloss.Color("#0D1117")).
			Width(baseWidth).
			Padding(0, 1)
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#3B4252")).
		Width(baseWidth).
		Padding(0, 1)
}

func (m mainModel) renderMessages() string {
	var lines []string

	for i, msg := range m.messages {
		if msg.IsSystem {
			// Clean system message styling
			prefix := "◆"
			if i == len(m.messages)-1 {
				prefix = pulseFrames[m.pulseFrame]
			}

			var content string
			if msg.User != "" {
				userStyle := lipgloss.NewStyle().
					Foreground(m.styles.PrimaryColor).
					Bold(true)
				content = fmt.Sprintf("%s %s %s", prefix, userStyle.Render(msg.User), msg.Content)
			} else {
				content = fmt.Sprintf("%s %s", prefix, msg.Content)
			}

			sysStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#00FF88")).
				Italic(true)

			line := sysStyle.Render("  " + content)
			lines = append(lines, line)
		} else {
			// Clean chat message formatting without background blocks
			isOwnMessage := msg.User == m.username

			// Format components with proper styling
			timestamp := m.styles.DateTime.Render(fmt.Sprintf("[%s]", msg.Timestamp))
			user := m.styles.User.Render(msg.User + ":")
			content := m.styles.Msg.Render(msg.Content)

			// Create clean message line without background blocks
			if isOwnMessage {
				// Own message - use primary color for user, white for content
				userStyle := lipgloss.NewStyle().
					Foreground(m.styles.PrimaryColor).
					Bold(true)
				user = userStyle.Render(msg.User + ":")
				contentStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color("#E5E7EB"))
				content = contentStyle.Render(msg.Content)

				messageLine := fmt.Sprintf("%s  %s %s", timestamp, user, content)
				lines = append(lines, messageLine)
			} else {
				// Other user's message - use theme colors
				messageLine := fmt.Sprintf("%s  %s %s", timestamp, user, content)
				lines = append(lines, messageLine)
			}
		}
	}

	return strings.Join(lines, "\n")
}

func parseMessage(raw string) ChatMessage {
	// Parse system messages (user joined/left)
	if strings.Contains(raw, " has joined") {
		username := strings.TrimSuffix(raw, " has joined")
		return ChatMessage{
			Timestamp: time.Now().Format("15:04"),
			User:      username,
			Content:   "joined the chat",
			IsSystem:  true,
		}
	}
	if strings.Contains(raw, " has left") {
		username := strings.TrimSuffix(raw, " has left")
		return ChatMessage{
			Timestamp: time.Now().Format("15:04"),
			User:      username,
			Content:   "left the chat",
			IsSystem:  true,
		}
	}

	// Parse chat messages in format: "timestamp: username said: message"
	// Example: "17/1/2026, 10:03:56 pm: Krishna said: Hello"
	parts := strings.SplitN(raw, ": ", 3)
	if len(parts) == 3 {
		// Check if it matches the pattern "timestamp: username said: message"
		if strings.HasSuffix(parts[1], " said") {
			username := strings.TrimSuffix(parts[1], " said")
			// Extract time from full timestamp (format: "17/1/2026, 10:03:56 pm")
			fullTimestamp := parts[0]
			// Try to extract just the time part
			timeParts := strings.Split(fullTimestamp, ", ")
			var displayTime string
			if len(timeParts) > 1 {
				// Extract time from "10:03:56 pm"
				timeOnly := timeParts[1]
				timeOnlyParts := strings.Split(timeOnly, ":")
				if len(timeOnlyParts) >= 2 {
					displayTime = timeOnlyParts[0] + ":" + timeOnlyParts[1]
					if len(timeOnlyParts) == 3 {
						// Include AM/PM if present
						secondsAndAmPm := strings.TrimSpace(timeOnlyParts[2])
						amPmParts := strings.Fields(secondsAndAmPm)
						if len(amPmParts) > 1 {
							displayTime += " " + amPmParts[1]
						}
					}
				} else {
					displayTime = timeOnly
				}
			} else {
				displayTime = time.Now().Format("15:04")
			}

			return ChatMessage{
				Timestamp: displayTime,
				User:      username,
				Content:   parts[2],
				IsSystem:  false,
			}
		}
	}

	// Fallback: treat as plain message
	return ChatMessage{
		Timestamp: time.Now().Format("15:04"),
		User:      "",
		Content:   raw,
		IsSystem:  false,
	}
}

// Commands and Messages

type connectedMsg struct {
	conn *websocket.Conn
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*500, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m mainModel) connectCmd() tea.Cmd {
	return func() tea.Msg {
		server := m.serverInput.Value()
		if server == "" {
			server = "localhost:8080"
		}

		conn, err := connectWebsocket(server, m.userInput.Value(), m.passInput.Value())
		if err != nil {
			return errMsg(err)
		}

		return connectedMsg{conn: conn}
	}
}

func (m mainModel) sendMessageCmd(msg string) tea.Cmd {
	return func() tea.Msg {
		if m.conn == nil {
			return errMsg(fmt.Errorf("not connected"))
		}
		err := m.conn.WriteMessage(websocket.TextMessage, []byte(msg))
		if err != nil {
			return errMsg(err)
		}
		return nil
	}
}

func waitForIncomingMessage(conn *websocket.Conn) tea.Cmd {
	return func() tea.Msg {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return errMsg(err)
		}
		return wsMsg(string(data))
	}
}
