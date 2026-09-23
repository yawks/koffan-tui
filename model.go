package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

type focusArea int

const (
	focusLists focusArea = iota
	focusContent
	focusForm
	focusConfirm
)

type formKind int

const (
	formList formKind = iota
	formSection
	formItem
)

type itemForm struct {
	kind              formKind
	editing           bool
	id                int
	originalSectionID int
	section           int
	field             int
	name              textinput.Model
	quantity          textinput.Model
	matches           []string
	matchSections     []int
	match             int
}

type confirmation struct {
	kind  string
	id    int
	name  string
	count int
}

type model struct {
	api           *client
	lists         []List
	listCursor    int
	activeList    int
	sections      []Section
	section       int
	row           int // -1: section header, len(rows): add button
	column        int
	showCompleted bool
	expanded      map[int]bool
	focus         focusArea
	form          *itemForm
	confirm       *confirmation
	loading       bool
	err           error
	width         int
	height        int
	contentScroll int
	pendingItem   int
}

func (m model) contentCursorLine() (line, total int) {
	if m.loading {
		line++
		total++
	}
	for i, section := range m.sections {
		rows := 1
		if m.expanded[section.ID] {
			rows = m.sectionRowCount(&section) + 4
		}
		if i == m.section {
			cursor := total
			if m.expanded[section.ID] && m.row >= 0 {
				cursor += 2 + m.row
			}
			line = cursor
		}
		total += rows
	}
	return line, total
}

func (m *model) ensureContentCursorVisible() {
	height := m.contentViewportHeight()
	line, total := m.contentCursorLine()
	if line < m.contentScroll {
		m.contentScroll = line
	} else if line >= m.contentScroll+height {
		m.contentScroll = line - height + 1
	}
	m.contentScroll = min(max(0, m.contentScroll), max(0, total-height))
}

func newModel(api *client) model {
	return model{api: api, expanded: make(map[int]bool), showCompleted: true, focus: focusLists, row: -1, loading: true}
}

func (m model) Init() tea.Cmd { return m.loadLists() }

type listsMsg struct {
	lists []List
	err   error
}

type sectionsMsg struct {
	sections []Section
	err      error
}

type actionMsg struct {
	kind string
	err  error
}

func (m model) loadLists() tea.Cmd {
	return func() tea.Msg {
		lists, err := m.api.lists()
		return listsMsg{lists, err}
	}
}

func (m model) loadSections() tea.Cmd {
	if len(m.lists) == 0 {
		return nil
	}
	id := m.lists[m.activeList].ID
	return func() tea.Msg {
		sections, err := m.api.sections(id)
		return sectionsMsg{sections, err}
	}
}

func (m model) selectedSection() *Section {
	if m.section < 0 || m.section >= len(m.sections) {
		return nil
	}
	return &m.sections[m.section]
}

func splitItems(s *Section) (open, done []Item) {
	if s == nil {
		return nil, nil
	}
	for _, item := range s.Items {
		if item.Completed {
			done = append(done, item)
		} else {
			open = append(open, item)
		}
	}
	sort.SliceStable(open, func(i, j int) bool { return open[i].Order < open[j].Order })
	sort.SliceStable(done, func(i, j int) bool { return done[i].Order < done[j].Order })
	return open, done
}

func (m model) currentItem() *Item {
	s := m.selectedSection()
	if s == nil || m.row < 0 {
		return nil
	}
	open, done := splitItems(s)
	items := open
	if m.column == 1 {
		items = done
	}
	if m.row >= len(items) {
		return nil
	}
	item := items[m.row]
	return &item
}

func (m model) rowCount() int {
	open, done := splitItems(m.selectedSection())
	if !m.showCompleted {
		return len(open)
	}
	return max(len(open), len(done))
}

func (m model) sectionRowCount(section *Section) int {
	open, done := splitItems(section)
	if !m.showCompleted {
		return len(open)
	}
	return max(len(open), len(done))
}

func (m *model) clampCursor() {
	if len(m.sections) == 0 {
		m.section, m.row = 0, -1
		return
	}
	m.section = min(max(m.section, 0), len(m.sections)-1)
	if !m.expanded[m.sections[m.section].ID] {
		m.row = -1
		return
	}
	m.row = min(max(m.row, -1), m.rowCount())
}

func newTextInput(placeholder string, limit int) textinput.Model {
	in := textinput.New()
	in.Placeholder = placeholder
	in.CharLimit = limit
	in.SetWidth(36)
	return in
}

func (m *model) openForm(kind formKind, sectionID int) tea.Cmd {
	limit := 100
	if kind == formItem {
		limit = 200
	}
	f := &itemForm{kind: kind, name: newTextInput("Nom", limit), match: -1}
	if kind == formItem {
		for i := range m.sections {
			if m.sections[i].ID == sectionID {
				f.section = i
			}
		}
		f.quantity = newTextInput("1", 9)
		f.quantity.Validate = func(value string) error {
			if value == "" {
				return nil
			}
			n, err := strconv.Atoi(value)
			if err != nil || n < 1 {
				return fmt.Errorf("entier supérieur ou égal à 1 attendu")
			}
			return nil
		}
	}
	m.form, m.focus = f, focusForm
	return m.form.name.Focus()
}

func (m *model) openEdit(kind formKind, id int, name string, quantity, sectionID int) tea.Cmd {
	cmd := m.openForm(kind, sectionID)
	m.form.editing = true
	m.form.id = id
	m.form.originalSectionID = sectionID
	m.form.name.SetValue(name)
	m.form.name.CursorEnd()
	if kind == formItem {
		m.form.quantity.SetValue(strconv.Itoa(max(1, quantity)))
		m.form.quantity.CursorEnd()
	}
	return cmd
}

func (m *model) updateMatches() {
	if m.form == nil || m.form.kind != formItem {
		return
	}
	needle := strings.ToLower(strings.TrimSpace(m.form.name.Value()))
	seen := make(map[string]bool)
	m.form.matches = nil
	m.form.matchSections = nil
	if needle != "" {
		for sectionIndex, section := range m.sections {
			for _, item := range section.Items {
				key := strings.ToLower(item.Name)
				if strings.Contains(key, needle) && !seen[key] {
					seen[key] = true
					m.form.matches = append(m.form.matches, item.Name)
					m.form.matchSections = append(m.form.matchSections, sectionIndex)
				}
			}
		}
	}
	if len(m.form.matches) == 0 {
		m.form.match = -1
	} else {
		m.form.match = min(max(m.form.match, 0), len(m.form.matches)-1)
	}
}
