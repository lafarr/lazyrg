package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Styles
var (
	// Color scheme
	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(highlight).
			Padding(1, 2).
			Bold(true)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(subtle).
			PaddingLeft(1).
			PaddingRight(1).
			Height(1)

	statusMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Render

	activeTabStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 4).
			MarginRight(2).
			Bold(true).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(highlight)

	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(subtle).
				Padding(0, 4).
				MarginRight(2).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(subtle)

	docStyle = lipgloss.NewStyle().
			Margin(1, 2).
			Padding(1, 2).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(subtle)

	highlightStyle = lipgloss.NewStyle().
			Foreground(highlight).
			Bold(true)

	inputBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1).
			MarginTop(1).
			MarginBottom(1).
			Align(lipgloss.Center)

	inputStyle = lipgloss.NewStyle().
			PaddingBottom(1).
			Align(lipgloss.Center)

	containerStyle = lipgloss.NewStyle().
			Align(lipgloss.Center)

	searchPromptStyle = lipgloss.NewStyle().
				Foreground(special).
				Bold(true)

	currentDirStyle = lipgloss.NewStyle().
			Foreground(subtle).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(subtle).
			Padding(0, 1).
			MarginTop(1).
			MarginBottom(1)

	dirIconStyle = lipgloss.NewStyle().
			Foreground(special).
			Bold(true)
)

// Custom item for search results
type Item struct {
	fileName string
	lineNum  string
	content  string
	fullPath string
}

func (i Item) Title() string       { return i.fileName + ":" + i.lineNum }
func (i Item) Description() string { return i.content }
func (i Item) FilterValue() string { return i.fileName + i.content }

// Key mappings
type keyMap struct {
	Search    key.Binding
	Search2   key.Binding
	Enter     key.Binding
	Back      key.Binding
	Quit      key.Binding
	Help      key.Binding
	Tab       key.Binding
	InputNext key.Binding
	InputPrev key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Search, k.Search2, k.Enter},
		{k.Back, k.Tab, k.Quit},
		{k.InputNext, k.InputPrev},
	}
}

var keys = keyMap{
	Search: key.NewBinding(

		key.WithKeys("ctrl+f"),
		key.WithHelp("ctrl+f", "search"),
	),
	Search2: key.NewBinding(
		key.WithKeys("ctrl+s"),
		key.WithHelp("ctrl+s", "search"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c", "q"),
		key.WithHelp("ctrl+c/q", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
	Tab: key.NewBinding(
		key.WithKeys("ctrl+t"),
		key.WithHelp("ctrl+t", "next tab"),
	),
	InputNext: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next input"),
	),
	InputPrev: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("shift+tab", "previous input"),
	),
}

// The tabs available in the UI
type tab int

const (
	searchTab tab = iota
	resultsTab
	fileTab
)

// Main application model
type model struct {
	tabs                 []string
	activeTab            tab
	searchInput          textinput.Model
	directoryInput       textinput.Model
	searchResults        list.Model
	fileViewer           viewport.Model
	statusMessage        string
	statusMessageType    string // "info", "error"
	width                int
	height               int
	showStatusBar        bool
	ready                bool
	help                 help.Model
	currentPath          string
	currentSearchPattern string
	keymap               keyMap
}

func initialModel() model {
	searchInput := textinput.New()
	searchInput.Placeholder = "Enter search pattern..."
	searchInput.Focus()
	searchInput.Width = 80
	searchInput.Prompt = "❯ "
	searchInput.PromptStyle = searchPromptStyle
	searchInput.TextStyle = lipgloss.NewStyle().Foreground(highlight)
	searchInput.Cursor.Style = lipgloss.NewStyle().Foreground(special)

	currentPath, err := os.Getwd()
	if err != nil {
		currentPath = "."
	}

	directoryInput := textinput.New()
	directoryInput.Placeholder = "Enter directory path (leave empty for current directory)..."
	directoryInput.Width = 80
	directoryInput.Prompt = "❯ "
	directoryInput.PromptStyle = searchPromptStyle
	directoryInput.TextStyle = lipgloss.NewStyle().Foreground(highlight)
	directoryInput.Cursor.Style = lipgloss.NewStyle().Foreground(special)

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		BorderForeground(highlight).
		BorderStyle(lipgloss.RoundedBorder()).
		Padding(0, 1)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		BorderForeground(highlight).
		BorderStyle(lipgloss.RoundedBorder()).
		Padding(0, 1)

	resultsList := list.New([]list.Item{}, delegate, 0, 0)
	resultsList.Title = "Search Results"
	resultsList.SetShowHelp(false)
	resultsList.Styles.Title = lipgloss.NewStyle().
		Foreground(special).
		Bold(true).
		MarginLeft(2)
	resultsList.Styles.FilterPrompt = lipgloss.NewStyle().
		Foreground(special)
	resultsList.Styles.FilterCursor = lipgloss.NewStyle().
		Foreground(highlight)

	fileViewer := viewport.New(0, 0)
	fileViewer.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#25A065")).
		Padding(0, 1)

	help := help.New()

	return model{
		tabs:              []string{"Search", "Results", "File View"},
		activeTab:         searchTab,
		searchInput:       searchInput,
		directoryInput:    directoryInput,
		searchResults:     resultsList,
		fileViewer:        fileViewer,
		statusMessage:     "Welcome to LazyRG! Press Ctrl+F to search",
		statusMessageType: "info",
		showStatusBar:     true,
		help:              help,
		currentPath:       currentPath,
		keymap:            keys,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

// Message types
type searchFinishedMsg struct {
	results []Item
	err     error
}

type fileLoadedMsg struct {
	content string
	err     error
}

// Run ripgrep
func executeRipgrep(pattern string, path string) tea.Cmd {
	return func() tea.Msg {
		if pattern == "" {
			return searchFinishedMsg{
				results: []Item{},
				err:     fmt.Errorf("empty search pattern"),
			}
		}

		cmd := exec.Command("rg", "--line-number", "--color", "never", "--no-heading", "--with-filename", pattern, path)
		output, err := cmd.CombinedOutput()
		if err != nil {
			if strings.Contains(string(output), "No such file or directory") {
				return searchFinishedMsg{
					results: []Item{},
					err:     fmt.Errorf("directory not found: %s", path),
				}
			}
			// rg returns exit code 1 when no matches were found, which is not an error for us
			if strings.TrimSpace(string(output)) == "" {
				return searchFinishedMsg{
					results: []Item{},
					err:     nil,
				}
			}
		}

		results := []Item{}
		lines := strings.Split(string(output), "\n")

		for _, line := range lines {
			if strings.TrimSpace(line) == "" {
				continue
			}

			parts := strings.SplitN(line, ":", 3)
			if len(parts) < 3 {
				continue
			}

			results = append(results, Item{
				fileName: strings.TrimSpace(parts[0]),
				lineNum:  strings.TrimSpace(parts[1]),
				content:  strings.TrimSpace(parts[2]),
				fullPath: strings.TrimSpace(parts[0]),
			})
		}

		return searchFinishedMsg{
			results: results,
			err:     nil,
		}
	}
}

// Load file content for viewing
func loadFile(filepath string, lineNum string) tea.Cmd {
	return func() tea.Msg {
		// Try using bat with line highlighting
		lineNumInt := 0
		if _, err := fmt.Sscanf(lineNum, "%d", &lineNumInt); err != nil {
			return fileLoadedMsg{err: fmt.Errorf("invalid line number: %s", lineNum)}
		}

		cmd := exec.Command("bat", "--color=always", "--style=full", "--highlight-line", lineNum, filepath)
		output, err := cmd.CombinedOutput()

		// Fallback to regular cat if bat is not installed
		if err != nil && strings.Contains(err.Error(), "executable file not found") {
			file, err := os.Open(filepath)
			if err != nil {
				return fileLoadedMsg{err: err}
			}
			defer file.Close()

			content, err := io.ReadAll(file)
			if err != nil {
				return fileLoadedMsg{err: err}
			}

			lines := strings.Split(string(content), "\n")

			// Simple highlighting
			highlightedContent := ""
			for i, line := range lines {
				lineNumberStr := fmt.Sprintf("%4d | ", i+1)
				if i+1 == lineNumInt {
					highlightedContent += "→ " + lineNumberStr + highlightStyle.Render(line) + "\n"
				} else {
					highlightedContent += "  " + lineNumberStr + line + "\n"
				}
			}

			return fileLoadedMsg{content: highlightedContent}
		}

		// If bat was successful, return its output
		if err == nil {
			return fileLoadedMsg{content: string(output)}
		}

		// If bat failed for any other reason, try without line highlighting
		cmd = exec.Command("bat", "--color=always", "--style=full", filepath)
		output, err = cmd.CombinedOutput()
		if err != nil {
			return fileLoadedMsg{err: err}
		}

		return fileLoadedMsg{content: string(output)}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keymap.Help):
			m.help.ShowAll = !m.help.ShowAll
			return m, nil

		case key.Matches(msg, m.keymap.Tab):
			m.activeTab = (m.activeTab + 1) % 3
			switch m.activeTab {
			case searchTab:
				m.searchInput.Focus()
			case resultsTab:
				if m.searchResults.Items() != nil && len(m.searchResults.Items()) > 0 {
					cmds = append(cmds, m.searchResults.StartSpinner())
				}
			}
			return m, tea.Batch(cmds...)

		case key.Matches(msg, m.keymap.Search) || key.Matches(msg, m.keymap.Search2):
			if m.activeTab != searchTab {
				m.activeTab = searchTab
				m.searchInput.Focus()
			}
			return m, nil

		case key.Matches(msg, m.keymap.Back):
			switch m.activeTab {
			case fileTab:
				m.activeTab = resultsTab
			case resultsTab:
				m.activeTab = searchTab
				m.searchInput.Focus()
			}
			return m, nil

		case key.Matches(msg, m.keymap.Enter):
			switch m.activeTab {
			case searchTab:
				if m.searchInput.Value() != "" {
					m.currentSearchPattern = m.searchInput.Value()
					searchPath := m.currentPath
					if m.directoryInput.Value() != "" {
						searchPath = m.directoryInput.Value()
					}
					m.activeTab = resultsTab
					m.statusMessage = fmt.Sprintf("Searching for: %s in %s", m.currentSearchPattern, searchPath)
					m.statusMessageType = "info"
					return m, executeRipgrep(m.currentSearchPattern, searchPath)
				}
			case resultsTab:
				if len(m.searchResults.Items()) > 0 {
					item, ok := m.searchResults.SelectedItem().(Item)
					if ok {
						m.activeTab = fileTab
						m.statusMessage = fmt.Sprintf("Viewing file: %s", item.fullPath)
						m.statusMessageType = "info"
						return m, loadFile(item.fullPath, item.lineNum)
					}
				}
			}
		}

	case searchFinishedMsg:
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Error: %s", msg.err)
			m.statusMessageType = "error"
			return m, nil
		}

		items := []list.Item{}
		for _, result := range msg.results {
			items = append(items, result)
		}

		if len(items) == 0 {
			m.statusMessage = "No results found"
			m.statusMessageType = "info"
		} else {
			m.statusMessage = fmt.Sprintf("Found %d results", len(items))
			m.statusMessageType = "info"
		}

		m.searchResults.SetItems(items)
		return m, nil

	case fileLoadedMsg:
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Error loading file: %s", msg.err)
			m.statusMessageType = "error"
			m.activeTab = resultsTab
			return m, nil
		}

		m.fileViewer.SetContent(msg.content)
		// Reset viewport to top when loading new file
		m.fileViewer.GotoTop()
		return m, nil

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.ready = true

		topHeight := 6  // For title and tabs with margins
		helpHeight := 2 // For help view at the bottom
		statusHeight := 1
		borderHeight := 2 // Account for top and bottom borders

		availableHeight := m.height - topHeight - helpHeight - statusHeight - borderHeight

		// Make input boxes fit within the container while accounting for padding and borders
		m.searchInput.Width = msg.Width - 30
		m.directoryInput.Width = msg.Width - 30

		h := availableHeight
		if m.help.ShowAll {
			h -= 4 // Make room for help
		}

		m.searchResults.SetSize(msg.Width-4, h)
		m.fileViewer.Width = msg.Width - 8 // Account for left/right borders and padding
		m.fileViewer.Height = h

		// Set viewport to start at the top
		m.fileViewer.GotoTop()

		m.help.Width = msg.Width

		return m, nil
	}

	// Handle updates for different components based on active tab
	switch m.activeTab {
	case searchTab:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if key.Matches(msg, m.keymap.InputNext) {
				if m.searchInput.Focused() {
					m.searchInput.Blur()
					m.directoryInput.Focus()
				} else {
					m.directoryInput.Blur()
					m.searchInput.Focus()
				}
				return m, nil
			} else if key.Matches(msg, m.keymap.InputPrev) {
				if m.searchInput.Focused() {
					m.searchInput.Blur()
					m.directoryInput.Focus()
				} else {
					m.directoryInput.Blur()
					m.searchInput.Focus()
				}
				return m, nil
			}
		}

		var cmd tea.Cmd
		if m.searchInput.Focused() {
			m.searchInput, cmd = m.searchInput.Update(msg)
		} else {
			m.directoryInput, cmd = m.directoryInput.Update(msg)
		}
		cmds = append(cmds, cmd)
	case resultsTab:
		var cmd tea.Cmd
		m.searchResults, cmd = m.searchResults.Update(msg)
		cmds = append(cmds, cmd)
	case fileTab:
		var cmd tea.Cmd
		m.fileViewer, cmd = m.fileViewer.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	var content string

	// Render tabs
	var renderedTabs []string
	for i, t := range m.tabs {
		if tab(i) == m.activeTab {
			renderedTabs = append(renderedTabs, activeTabStyle.Render(t))
		} else {
			renderedTabs = append(renderedTabs, inactiveTabStyle.Render(t))
		}
	}
	tabsView := lipgloss.JoinHorizontal(lipgloss.Center, renderedTabs...)

	// Status bar
	var statusBar string
	if m.showStatusBar {
		statusMsg := statusMessageStyle(m.statusMessage)
		statusBar = statusBarStyle.Width(m.width - 2).Render(statusMsg)
	}

	// Different content based on the active tab
	switch m.activeTab {
	case searchTab:
		searchBox := inputBoxStyle.Render(
			lipgloss.JoinVertical(
				lipgloss.Center,
				"Search Pattern",
				inputStyle.Render(m.searchInput.View()),
			),
		)

		directoryBox := inputBoxStyle.Render(
			lipgloss.JoinVertical(
				lipgloss.Center,
				"Directory Path",
				inputStyle.Render(m.directoryInput.View()),
			),
		)

		currentDirInfo := currentDirStyle.Render(
			fmt.Sprintf("%s %s",
				dirIconStyle.Render("📂"),
				m.currentPath,
			),
		)

		content = containerStyle.Width(m.width - 4).Render(
			lipgloss.JoinVertical(
				lipgloss.Center,
				tabsView,
				searchBox,
				directoryBox,
				currentDirInfo,
			),
		)
	case resultsTab:
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			tabsView,
			m.searchResults.View(),
		)
	case fileTab:
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			tabsView,
			m.fileViewer.View(),
		)
	}

	// Help view
	helpView := m.help.View(m.keymap)

	return fmt.Sprintf(
		"%s\n%s\n%s\n%s",
		titleStyle.Width(m.width-2).Render("LazyRG - Interactive Ripgrep TUI"),

		docStyle.Width(m.width-4).Height(m.height-7).Render(content),
		statusBar,
		helpView,
	)
}

func main() {
	logFile, err := os.OpenFile("lazyrg.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening log file: %v\n", err)
		os.Exit(1)
	}
	defer logFile.Close()

	log.SetOutput(logFile)
	log.Println("Starting LazyRG")

	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running program: %v", err)
		os.Exit(1)
	}
}
