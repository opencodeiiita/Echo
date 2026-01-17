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
	conn   *websocket.Conn
	err    error
	width  int
	height int

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
		m.chatStartTime = time.Now() // Start tracking for adaptive animation

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

	// Logo with theme color
	logoLines := strings.Split(echoLogo, "\n")
	for _, line := range logoLines {
		if line == "" {
			continue
		}
		lineStyle := lipgloss.NewStyle().
			Foreground(m.styles.PrimaryColor).
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

	// Error message
	if m.err != nil {
		errStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF4757")).
			Bold(true).
			Width(50).
			Align(lipgloss.Center)
		b.WriteString(errStyle.Render("! " + m.err.Error()))
		b.WriteString("\n\n")
	}

	// Animated hints
	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Italic(true).
		Width(50).
		Align(lipgloss.Center)
	b.WriteString(hintStyle.Render("Tab: Navigate | Enter: Select | Esc: Quit"))

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

	// Animated connecting box
	frame := connectFrames[m.animFrame]

	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7D56F4")).
		Bold(true).
		Render("CONNECTING")

	animation := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00D9FF")).
		Bold(true).
		Render(frame)

	spinnerView := m.spinner.View()

	content := fmt.Sprintf(`
    %s %s

    %s

    Establishing secure connection...
    Server: %s
    User: %s
`,
		spinnerView,
		title,
		animation,
		m.serverInput.Value(),
		m.userInput.Value(),
	)

	box := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(m.styles.PrimaryColor).
		Padding(2, 4).
		Align(lipgloss.Center).
		Render(content)

	b.WriteString(box)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, b.String())
}

func (m mainModel) chatViewRender() string {
	var b strings.Builder

	// Header bar - uses theme primary color
	headerBg := m.styles.PrimaryColor
	headerFg := lipgloss.Color("#FFFFFF")

	// Pulsing online indicator (green dot)
	onlineIndicator := lipgloss.NewStyle().
		Background(headerBg).
		Foreground(lipgloss.Color("#00FF88")).
		Bold(true).
		Render(pulseFrames[m.pulseFrame])

	// ECHO label
	chatIcon := lipgloss.NewStyle().
		Background(headerBg).
		Foreground(headerFg).
		Bold(true).
		Render(" ECHO ")

	// Separator with purple background
	separator := lipgloss.NewStyle().
		Background(headerBg).
		Foreground(headerFg).
		Render("|")

	// Online indicator section
	onlineSection := lipgloss.NewStyle().
		Background(headerBg).
		Foreground(headerFg).
		Render(" " + onlineIndicator + " ")

	// Username - white on purple
	userSection := lipgloss.NewStyle().
		Background(headerBg).
		Foreground(headerFg).
		Bold(true).
		Render(m.userInput.Value() + " ")

	headerContent := chatIcon + separator + onlineSection + userSection

	// Full width header with purple background
	headerStyle := lipgloss.NewStyle().
		Background(headerBg).
		Foreground(headerFg).
		Width(m.width-2).
		Padding(0, 1)

	b.WriteString(headerStyle.Render(headerContent))
	b.WriteString("\n")

	// Chat viewport with styled border
	chatBorder := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#3B4252")).
		Width(m.width - 4).
		Height(m.viewport.Height + 2)

	b.WriteString(chatBorder.Render(m.viewport.View()))
	b.WriteString("\n")

	// Adaptive input border animation
	// Fast for first 10 seconds, then slow
	elapsedSeconds := time.Since(m.chatStartTime).Seconds()
	var inputBorderColor string
	if elapsedSeconds < 10 {
		// Fast animation - every frame
		if m.animFrame%2 == 0 {
			inputBorderColor = string(m.styles.SecondaryColor)
		} else {
			inputBorderColor = string(m.styles.PrimaryColor)
		}
	} else {
		// Slow animation - every 4 frames
		if m.animFrame%8 < 4 {
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

	// Footer hints with keyboard icons
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Italic(true)
	b.WriteString(footerStyle.Render(" [Enter] Send | [Up/Down] Scroll | [Esc] Quit"))

	return b.String()
}

func (m mainModel) getInputStyle(index int) lipgloss.Style {
	baseWidth := 50
	if m.focusIndex == index {
		// Animated glow effect for focused input
		glowColors := []string{"#00D9FF", "#00C8EE", "#00B7DD", "#00C8EE"}
		colorIdx := m.animFrame % len(glowColors)
		return lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(glowColors[colorIdx])).
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
			// System message with animation-like prefix
			prefix := ">"
			if i == len(m.messages)-1 {
				prefix = pulseFrames[m.pulseFrame]
			}
			sysStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#00FF88")).
				Italic(true)
			line := sysStyle.Render(fmt.Sprintf("  %s %s", prefix, msg.Content))
			lines = append(lines, line)
		} else {
			// User message with styled components
			timeStyle := m.styles.DateTime
			userStyle := m.styles.User
			msgStyle := m.styles.Msg

			timestamp := timeStyle.Render(fmt.Sprintf("[%s]", msg.Timestamp))
			user := userStyle.Render(msg.User)
			content := msgStyle.Render(msg.Content)

			line := fmt.Sprintf("%s %s: %s", timestamp, user, content)
			lines = append(lines, line)
		}
	}

	return strings.Join(lines, "\n")
}

func parseMessage(raw string) ChatMessage {
	timestamp := time.Now().Format("15:04")

	return ChatMessage{
		Timestamp: timestamp,
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
