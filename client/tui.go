package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type state int

const (
	stateLogin state = iota
	stateConnecting
	stateChat
)

type tickMsg time.Time

type model struct {
	state       state
	theme       Theme
	serverInput textinput.Model
	userInput   textinput.Model
	passInput   textinput.Model
	msgInput    textinput.Model
	focusIndex  int
	err         string
	session     *Session
	chatHistory []Message
	width       int
	height      int
	animStep    int
}

func initialModel() model {
	s := textinput.New()
	s.Placeholder = "127.0.0.1:8080"
	s.Focus()

	u := textinput.New()
	u.Placeholder = "Username"

	p := textinput.New()
	p.Placeholder = "Password"
	p.EchoMode = textinput.EchoPassword
	p.EchoCharacter = '•'

	m := textinput.New()
	m.Placeholder = "Type a message..."

	return model{
		state:       stateLogin,
		theme:       LoadTheme("echo.theme"),
		serverInput: s,
		userInput:   u,
		passInput:   p,
		msgInput:    m,
		chatHistory: []Message{},
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.tick())
}

func (m model) tick() tea.Cmd {
	return tea.Tick(time.Millisecond*50, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tickMsg:
		m.animStep++
		return m, m.tick()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			if m.session != nil {
				m.session.Close()
			}
			return m, tea.Quit

		case "tab", "shift+tab", "up", "down":
			if m.state == stateLogin {
				s := msg.String()
				if s == "up" || s == "shift+tab" {
					m.focusIndex--
				} else {
					m.focusIndex++
				}

				if m.focusIndex > 3 {
					m.focusIndex = 0
				} else if m.focusIndex < 0 {
					m.focusIndex = 3
				}

				cmds := make([]tea.Cmd, 3)
				m.serverInput.Blur()
				m.userInput.Blur()
				m.passInput.Blur()

				switch m.focusIndex {
				case 0:
					cmds[0] = m.serverInput.Focus()
				case 1:
					cmds[1] = m.userInput.Focus()
				case 2:
					cmds[2] = m.passInput.Focus()
				}
				return m, tea.Batch(cmds...)
			}

		case "enter":
			if m.state == stateLogin {
				if m.focusIndex == 3 || m.passInput.Focused() {
					m.state = stateConnecting
					m.err = ""
					return m, m.connect()
				}
			} else {
				val := m.msgInput.Value()
				if val != "" {
					m.session.Send(val)
					m.msgInput.Reset()
				}
			}
		}

	case *Session:
		m.session = msg
		m.state = stateChat
		m.msgInput.Focus()
		return m, m.listenForMessages()

	case Message:
		m.chatHistory = append(m.chatHistory, msg)
		return m, m.listenForMessages()

	case error:
		m.err = msg.Error()
		m.state = stateLogin
		return m, nil
	}

	if m.state == stateLogin {
		switch m.focusIndex {
		case 0:
			m.serverInput, cmd = m.serverInput.Update(msg)
		case 1:
			m.userInput, cmd = m.userInput.Update(msg)
		case 2:
			m.passInput, cmd = m.passInput.Update(msg)
		}
	} else if m.state == stateChat {
		m.msgInput, cmd = m.msgInput.Update(msg)
	}

	return m, cmd
}

// connect handles the handshake with the backend.
// adding a small delay so the "scanning" animation doesn't 
// just flicker and disappear on fast connections.
func (m model) connect() tea.Cmd {
	return func() tea.Msg {
		server := m.serverInput.Value()
		if server == "" {
			server = "127.0.0.1:8080"
		}
		
		// let the scanner animation play for a bit
		time.Sleep(1200 * time.Millisecond)
		
		session, err := NewSession(server, m.userInput.Value(), m.passInput.Value())
		if err != nil {
			return err
		}
		return session
	}
}

func (m model) listenForMessages() tea.Cmd {
	return func() tea.Msg {
		select {
		case msg := <-m.session.Inbound:
			return msg
		case err := <-m.session.Errors:
			return err
		}
	}
}

func (m model) View() string {
	switch m.state {
	case stateConnecting:
		return m.connectingView()
	case stateChat:
		return m.chatView()
	default:
		return m.loginView()
	}
}

func (m model) connectingView() string {
	// A more unique "Scanning" animation
	frames := []string{
		"[    ]",
		"[=   ]",
		"[==  ]",
		"[=== ]",
		"[====]",
		"[ ===]",
		"[  ==]",
		"[   =]",
	}
	f := frames[(m.animStep/2)%len(frames)]
	
	style := lipgloss.NewStyle().Foreground(m.theme.Button).Bold(true)
	title := style.Render("ESTABLISHING ENCRYPTED LINK")
	bar := style.Render(f)
	
	content := fmt.Sprintf("%s\n\n%s\n\nTarget: %s", title, bar, m.serverInput.Value())
	
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func (m model) loginView() string {
	// Glitchy Title Effect
	title := "E C H O"
	var glitchedTitle strings.Builder
	for i, r := range title {
		if (m.animStep/10)%7 == i {
			glitchedTitle.WriteString(lipgloss.NewStyle().Foreground(m.theme.Error).Render(string(r)))
		} else {
			glitchedTitle.WriteString(lipgloss.NewStyle().Foreground(m.theme.Button).Bold(true).Render(string(r)))
		}
	}

	formStyle := lipgloss.NewStyle().
		Border(lipgloss.ThickBorder()).
		BorderForeground(m.theme.Button).
		Padding(1, 4)

	var s strings.Builder
	s.WriteString(glitchedTitle.String() + "\n\n")

	s.WriteString("SERVER_LINK>\n")
	s.WriteString(m.serverInput.View() + "\n\n")

	s.WriteString("USER_ID>\n")
	s.WriteString(m.userInput.View() + "\n\n")

	s.WriteString("ACCESS_KEY>\n")
	s.WriteString(m.passInput.View() + "\n\n")

	buttonStr := " [ INITIATE_CONNECTION ] "
	buttonStyle := lipgloss.NewStyle().Bold(true)
	if m.focusIndex == 3 {
		// Shimmer effect
		if (m.animStep/4)%4 == 0 {
			buttonStyle = buttonStyle.Background(lipgloss.Color("#ffffff")).Foreground(lipgloss.Color("#000000"))
		} else {
			buttonStyle = buttonStyle.Background(m.theme.Button).Foreground(lipgloss.Color("#000000"))
		}
	} else {
		buttonStyle = buttonStyle.
			Border(lipgloss.RoundedBorder()).
			BorderForeground(m.theme.Button).
			Foreground(m.theme.Button)
	}
	s.WriteString(buttonStyle.Render(buttonStr) + "\n\n")

	if m.err != "" {
		s.WriteString(lipgloss.NewStyle().Foreground(m.theme.Error).Render("ERR_LOG: "+m.err) + "\n\n")
	}

	hintColor := lipgloss.AdaptiveColor{Light: "#AAAAAA", Dark: "#555555"}
	s.WriteString(lipgloss.NewStyle().Foreground(hintColor).Italic(true).Render("SYS_NAV: [TAB] cycle | [ENTER] submit"))

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, formStyle.Render(s.String()))
}

func (m model) chatView() string {
	chatBorder := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(m.theme.Msg).
		Width(m.width - 4).
		Height(m.height - 7)

	header := lipgloss.NewStyle().
		Foreground(m.theme.Button).
		Bold(true).
		Border(lipgloss.NormalBorder(), false, false, true).
		BorderForeground(m.theme.Msg).
		Render(" TERMINAL_SESSION // USER:" + m.session.Username)

	var chatContent strings.Builder
	maxLines := m.height - 10
	start := 0
	if len(m.chatHistory) > maxLines {
		start = len(m.chatHistory) - maxLines
	}

	for i := start; i < len(m.chatHistory); i++ {
		msg := m.chatHistory[i]
		ts := lipgloss.NewStyle().Foreground(m.theme.DateTime).Render("[" + msg.Timestamp + "] ")
		
		var sender string
		if msg.Sender == "Server" {
			sender = lipgloss.NewStyle().Foreground(m.theme.DateTime).Italic(true).Render(msg.Sender + " >> ")
		} else {
			sender = lipgloss.NewStyle().Foreground(m.theme.User).Bold(true).Render(msg.Sender + " >> ")
		}
		
		content := lipgloss.NewStyle().Foreground(m.theme.Text).Render(msg.Content)
		chatContent.WriteString(fmt.Sprintf("%s%s%s\n", ts, sender, content))
	}

	footer := lipgloss.NewStyle().Foreground(m.theme.DateTime).Render(" COMMAND: [CTRL+C] TERMINATE")

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		chatBorder.Render(chatContent.String()),
		m.msgInput.View(),
		footer,
	)
}
