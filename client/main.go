package main

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gorilla/websocket"
)

// --- Styles ---
var (
	focusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205")) // Pink focus
	blurredStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240")) // Grey blur
	cursorStyle  = focusedStyle.Copy()
	noStyle      = lipgloss.NewStyle()

	// Layout Styles
	docStyle      = lipgloss.NewStyle().Margin(1, 2)
	sidebarStyle  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62")).Width(20).Height(20).Padding(0, 1)
	chatViewStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62")).Width(50).Height(18).Padding(0, 1)
	inputStyle    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("205")).Width(72).Padding(0, 1)

	// Text Styles
	headerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Bold(true).Padding(1)
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
)

// --- Application State ---
type sessionState int

const (
	loginView sessionState = iota
	chatView
)

// --- Messages ---
type connectedMsg *websocket.Conn
type errMsg error
type incomingMsg string

// --- Main Model ---
type model struct {
	state  sessionState
	inputs []textinput.Model // 0: Server, 1: Username, 2: Password
	focus  int

	// Connection
	conn *websocket.Conn

	// Chat components
	viewport   viewport.Model
	chatInput  textinput.Model
	channels   []string
	activeChan int
	messages   []string
	err        error
}

func initialModel() model {
	// Initialize Login Inputs
	m := model{
		inputs:   make([]textinput.Model, 3),
		channels: []string{"# general", "# random", "# tech-talk", "# memes"},
		messages: []string{"Welcome to Echo!", "System: Ready to connect..."},
	}

	var t textinput.Model
	for i := range m.inputs {
		t = textinput.New()
		t.Cursor.Style = cursorStyle
		t.CharLimit = 32
		t.Width = 32 // Fix: Prevent truncation

		switch i {
		case 0:
			t.Placeholder = "Server URL (e.g. localhost:8080)"
			t.SetValue("localhost:8080") // Default convenience
			t.Focus()
			t.PromptStyle = focusedStyle
			t.TextStyle = focusedStyle
		case 1:
			t.Placeholder = "Username"
		case 2:
			t.Placeholder = "Password"
			t.EchoMode = textinput.EchoPassword
			t.EchoCharacter = '•'
		}
		m.inputs[i] = t
	}

	// Initialize Chat Input
	m.chatInput = textinput.New()
	m.chatInput.Placeholder = "Type a message..."
	m.chatInput.Focus()
	m.chatInput.CharLimit = 100
	m.chatInput.Width = 70

	// Initialize Viewport
	m.viewport = viewport.New(50, 18)
	m.viewport.SetContent(strings.Join(m.messages, "\n"))

	return m
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

// --- WebSocket Commands ---

func connectToSocket(host, username string) tea.Cmd {
	return func() tea.Msg {
		u := url.URL{Scheme: "ws", Host: host, Path: "/"}
		conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		if err != nil {
			return errMsg(err)
		}

		// Send the username immediately upon connection
		err = conn.WriteMessage(websocket.TextMessage, []byte(username))
		if err != nil {
			return errMsg(err)
		}

		return connectedMsg(conn)
	}
}

func waitForMessage(conn *websocket.Conn) tea.Cmd {
	return func() tea.Msg {
		if conn == nil {
			return nil
		}
		_, message, err := conn.ReadMessage()
		if err != nil {
			return errMsg(err)
		}
		return incomingMsg(string(message))
	}
}

// --- Update ---

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			if m.conn != nil {
				m.conn.Close()
			}
			return m, tea.Quit
		}

		// Handle Logic based on View
		if m.state == loginView {
			newModel, newCmd := updateLogin(msg, m)
			m = newModel.(model)
			cmds = append(cmds, newCmd)
		} else {
			// --- FIX: Add Navigation Logic Here ---
			switch msg.String() {
			case "up":
				m.activeChan--
				if m.activeChan < 0 {
					m.activeChan = len(m.channels) - 1 // Loop to bottom
				}
			case "down":
				m.activeChan++
				if m.activeChan >= len(m.channels) {
					m.activeChan = 0 // Loop to top
				}
			case "enter":
				// LOGIC SPLIT: Send Message vs. Switch Channel
				if m.chatInput.Value() != "" {
					// 1. SEND MESSAGE
					if m.conn != nil {
						text := m.chatInput.Value()
						m.conn.WriteMessage(websocket.TextMessage, []byte(text))
						
						timestamp := time.Now().Format("15:04:05")
						displayMsg := fmt.Sprintf("[%s] You: %s", timestamp, text)
						m.messages = append(m.messages, displayMsg)
						m.viewport.SetContent(strings.Join(m.messages, "\n"))
						m.viewport.GotoBottom()
					}
					m.chatInput.Reset()
				} else {
					// 2. SWITCH CHANNEL (Simulated)
					selectedChannel := m.channels[m.activeChan]
					
					// Optional: Clear previous messages to make it feel like a new room
					// m.messages = nil 
					
					// Add a system message indicating the switch
					infoMsg := fmt.Sprintf("--- Switched to %s ---", selectedChannel)
					m.messages = append(m.messages, infoMsg)
					m.viewport.SetContent(strings.Join(m.messages, "\n"))
					m.viewport.GotoBottom()
				}
			}
		}

	case connectedMsg:
		// Connection Successful!
		m.conn = msg
		m.state = chatView
		m.messages = append(m.messages, "System: ✓ Connected to server")
		m.viewport.SetContent(strings.Join(m.messages, "\n"))
		// Start listening for incoming messages
		cmds = append(cmds, waitForMessage(m.conn))

	case incomingMsg:
		// Received Message from Server
		timestamp := time.Now().Format("15:04:05")
		m.messages = append(m.messages, fmt.Sprintf("[%s] %s", timestamp, string(msg)))
		m.viewport.SetContent(strings.Join(m.messages, "\n"))
		m.viewport.GotoBottom()
		// Continue listening
		cmds = append(cmds, waitForMessage(m.conn))

	case errMsg:
		m.err = msg
		m.messages = append(m.messages, fmt.Sprintf("Error: %v", msg))
		m.viewport.SetContent(strings.Join(m.messages, "\n"))
	}

	// FIX: Always run updates for components (Blinking cursor, typing)
	// This was unreachable in your previous code due to early returns
	if m.state == loginView {
		cmd = m.updateInputs(msg)
		cmds = append(cmds, cmd)
	} else {
		m.chatInput, cmd = m.chatInput.Update(msg)
		cmds = append(cmds, cmd)
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func updateLogin(msg tea.KeyMsg, m model) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab", "shift+tab", "enter", "up", "down":
		s := msg.String()

		// Connect Action
		if s == "enter" && m.focus == len(m.inputs)-1 {
			serverHost := m.inputs[0].Value()
			username := m.inputs[1].Value()
			// Return the connection command
			return m, connectToSocket(serverHost, username)
		}

		// Navigation
		if s == "up" || s == "shift+tab" {
			m.focus--
		} else {
			m.focus++
		}

		if m.focus > len(m.inputs)-1 {
			m.focus = 0
		} else if m.focus < 0 {
			m.focus = len(m.inputs) - 1
		}

		// Update styles
		for i := 0; i <= len(m.inputs)-1; i++ {
			if i == m.focus {
				m.inputs[i].Focus()
				m.inputs[i].PromptStyle = focusedStyle
				m.inputs[i].TextStyle = focusedStyle
				continue
			}
			m.inputs[i].Blur()
			m.inputs[i].PromptStyle = noStyle
			m.inputs[i].TextStyle = noStyle
		}
		return m, nil
	}
	return m, nil
}

func (m *model) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return tea.Batch(cmds...)
}

// --- View ---
func (m model) View() string {
	if m.state == loginView {
		// Error display
		errView := ""
		if m.err != nil {
			errView = errorStyle.Render(fmt.Sprintf("\nError: %v", m.err))
		}

		return docStyle.Render(
			lipgloss.JoinVertical(
				lipgloss.Center,
				headerStyle.Render("ECHO CHAT LOGIN"),
				m.inputs[0].View(),
				m.inputs[1].View(),
				m.inputs[2].View(),
				lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("\n[Tab] to cycle • [Enter] to connect"),
				errView,
			),
		)
	}

	// Chat View
	var channelRows []string
	
	for i, ch := range m.channels {
		var cursorStr, nameStr string
		var style lipgloss.Style

		if i == m.activeChan {
			cursorStr = ">" // Just the char, padding handled by container
			nameStr = ch
			style = focusedStyle
		} else {
			cursorStr = ""
			nameStr = ch
			style = noStyle
		}

		// COLUMN 1: The Cursor
		// Fixed width of 2 cells. Aligned right so it points to the text.
		cursor := lipgloss.NewStyle().
			Width(2).
			Align(lipgloss.Right).
			PaddingRight(1). // Space between cursor and text
			Foreground(lipgloss.Color("205")). // Always pink cursor
			Render(cursorStr)

		// COLUMN 2: The Channel Name
		name := style.Render(nameStr)

		// Join them horizontally
		row := lipgloss.JoinHorizontal(lipgloss.Left, cursor, name)
		channelRows = append(channelRows, row)
	}

	// Join all rows vertically
	channelList := lipgloss.JoinVertical(lipgloss.Left, channelRows...)

	sidebar := sidebarStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, headerStyle.Render("Channels"), channelList),
	)

	chatWindow := chatViewStyle.Render(m.viewport.View())
	inputBox := inputStyle.Render(m.chatInput.View())

	mainUI := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, chatWindow)

	return docStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			headerStyle.Render("Echo Terminal"),
			mainUI,
			inputBox,
		),
	)
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}