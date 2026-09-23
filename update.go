package main

import (
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.ensureContentCursorVisible()
		return m, nil
	case listsMsg:
		m.loading, m.err = false, msg.err
		if msg.err == nil {
			m.lists = msg.lists
			if m.listCursor >= len(m.lists) {
				m.listCursor = max(0, len(m.lists)-1)
			}
			if m.activeList >= len(m.lists) {
				m.activeList = max(0, len(m.lists)-1)
			}
			if len(m.lists) > 0 {
				m.loading = true
				return m, m.loadSections()
			}
			m.sections = nil
		}
		return m, nil
	case sectionsMsg:
		m.loading, m.err = false, msg.err
		if msg.err == nil {
			m.sections = msg.sections
			if m.activeList < len(m.lists) {
				total, completed := 0, 0
				for _, section := range m.sections {
					total += len(section.Items)
					for _, item := range section.Items {
						if item.Completed {
							completed++
						}
					}
				}
				m.lists[m.activeList].Stats.Total = total
				m.lists[m.activeList].Stats.Completed = completed
			}
			m.clampCursor()
			if m.pendingItem != 0 {
				m.followItem(m.pendingItem)
				m.pendingItem = 0
			}
			m.ensureContentCursorVisible()
		}
		return m, nil
	case actionMsg:
		m.loading, m.err = false, msg.err
		if msg.err != nil {
			return m, nil
		}
		switch msg.kind {
		case "list":
			return m, m.loadLists()
		default:
			return m, m.loadSections()
		}
	}

	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	if key.String() == "ctrl+c" || (key.String() == "q" && m.focus != focusForm) {
		return m, tea.Quit
	}
	if m.focus == focusForm {
		return m.updateForm(key)
	}
	if m.focus == focusConfirm {
		return m.updateConfirm(key)
	}
	if m.loading {
		return m, nil
	}
	m.err = nil
	switch m.focus {
	case focusLists:
		return m.updateLists(key)
	case focusContent:
		return m.updateContent(key)
	}
	return m, nil
}

func (m model) updateLists(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch keyName(key) {
	case "tab", "right", "l":
		if len(m.lists) > 0 {
			m.focus = focusContent
		}
	case "up", "k":
		m.listCursor = max(0, m.listCursor-1)
	case "down", "j":
		m.listCursor = min(len(m.lists)-1, m.listCursor+1)
	case "enter":
		if len(m.lists) > 0 {
			m.activeList = m.listCursor
			m.loading = true
			m.sections, m.section, m.row = nil, 0, -1
			m.contentScroll = 0
			return m, m.loadSections()
		}
	case "n":
		return m, m.openForm(formList, 0)
	case "e":
		if len(m.lists) > 0 {
			l := m.lists[m.listCursor]
			return m, m.openEdit(formList, l.ID, l.Name, 0, 0)
		}
	case "backspace", "delete":
		if len(m.lists) > 0 {
			l := m.lists[m.listCursor]
			m.confirm = &confirmation{"lists", l.ID, l.Name, l.Stats.Total}
			m.focus = focusConfirm
		}
	}
	return m, nil
}

func (m model) updateContent(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	s := m.selectedSection()
	switch keyName(key) {
	case "tab", "shift+tab":
		m.focus = focusLists
	case "left", "h":
		if m.row >= 0 {
			m.setColumn(0)
		} else if s != nil {
			m.expanded[s.ID] = false
		}
	case "right", "l":
		if m.row == -1 && s != nil {
			m.expanded[s.ID] = true
		} else if m.row >= 0 && m.showCompleted {
			m.setColumn(1)
		}
	case "up", "k":
		m.moveUp()
	case "down", "j":
		m.moveDown()
	case "enter", "space":
		if s == nil {
			break
		}
		if m.row == -1 {
			m.expanded[s.ID] = !m.expanded[s.ID]
		} else if m.row == m.rowCount() {
			return m, m.openForm(formItem, s.ID)
		} else if item := m.currentItem(); item != nil {
			m.pendingItem, m.loading = item.ID, true
			id := item.ID
			return m, func() tea.Msg { return actionMsg{"item", m.api.toggleItem(id)} }
		}
	case "a":
		if s != nil {
			return m, m.openForm(formItem, s.ID)
		}
	case "s":
		if len(m.lists) > 0 {
			return m, m.openForm(formSection, 0)
		}
	case "e":
		if s == nil {
			break
		}
		if m.row == -1 {
			return m, m.openEdit(formSection, s.ID, s.Name, 0, 0)
		}
		if item := m.currentItem(); item != nil {
			return m, m.openEdit(formItem, item.ID, item.Name, item.Quantity, item.SectionID)
		}
	case "backspace", "delete":
		if s == nil {
			break
		}
		if m.row == -1 {
			m.confirm = &confirmation{"sections", s.ID, s.Name, len(s.Items)}
			m.focus = focusConfirm
		} else if item := m.currentItem(); item != nil {
			m.confirm = &confirmation{"items", item.ID, item.Name, 0}
			m.focus = focusConfirm
		}
	case "r":
		m.loading = true
		return m, m.loadSections()
	case "u":
		m.showCompleted = !m.showCompleted
		if !m.showCompleted {
			m.column = 0
		}
	case "c":
		m.toggleAllSections()
	}
	m.clampCursor()
	m.ensureContentCursorVisible()
	return m, nil
}

func keyName(key tea.KeyPressMsg) string {
	if key.Key().Code == tea.KeyBackspace || key.Key().Code == tea.KeyDelete {
		return "delete"
	}
	return key.String()
}

func (m *model) toggleAllSections() {
	expand := false
	for _, section := range m.sections {
		if !m.expanded[section.ID] {
			expand = true
			break
		}
	}
	for _, section := range m.sections {
		m.expanded[section.ID] = expand
	}
	m.row = -1
}

func (m *model) moveDown() {
	if len(m.sections) == 0 {
		return
	}
	s := m.sections[m.section]
	if m.expanded[s.ID] {
		items := splitLeft(s)
		if m.column == 1 {
			items = splitRight(s)
		}
		switch {
		case m.row == -1 && len(items) > 0:
			m.row = 0
			return
		case m.row >= 0 && m.row < len(items)-1:
			m.row++
			return
		case m.row < m.rowCount():
			m.row = m.rowCount()
			return
		}
	}
	if m.section < len(m.sections)-1 {
		m.section++
		m.row = -1
	}
}

func (m *model) moveUp() {
	if len(m.sections) == 0 {
		return
	}
	if m.row == m.rowCount() {
		items := splitLeft(m.sections[m.section])
		if m.column == 1 {
			items = splitRight(m.sections[m.section])
		}
		m.row = len(items) - 1
	} else if m.row > -1 {
		m.row--
	} else if m.section > 0 {
		m.section--
		if s := m.sections[m.section]; m.expanded[s.ID] {
			m.row = max(len(splitLeft(s)), len(splitRight(s)))
		}
	}
}

func (m *model) setColumn(column int) {
	if column == 1 && !m.showCompleted {
		return
	}
	if m.row == m.rowCount() {
		m.column = column
		return
	}
	open, done := splitItems(m.selectedSection())
	items := open
	if column == 1 {
		items = done
	}
	if len(items) > 0 {
		m.column = column
		m.row = min(m.row, len(items)-1)
	}
}

func splitLeft(s Section) []Item  { a, _ := splitItems(&s); return a }
func splitRight(s Section) []Item { _, b := splitItems(&s); return b }

func (m *model) followItem(id int) {
	for si := range m.sections {
		open, done := splitItems(&m.sections[si])
		for col, items := range [][]Item{open, done} {
			for row := range items {
				if items[row].ID == id {
					m.section, m.row, m.column = si, row, col
					m.expanded[m.sections[si].ID] = true
					return
				}
			}
		}
	}
}

func (m model) updateForm(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if key.String() == "esc" {
		m.form, m.focus = nil, focusContent
		if len(m.lists) == 0 {
			m.focus = focusLists
		}
		return m, nil
	}
	f := m.form
	if f.kind == formItem && f.field == 2 {
		switch key.String() {
		case "left", "up", "h", "k":
			f.section = max(0, f.section-1)
			return m, nil
		case "right", "down", "l", "j":
			f.section = min(len(m.sections)-1, f.section+1)
			return m, nil
		case "shift+tab":
			f.field = 1
			return m, f.quantity.Focus()
		case "tab":
			f.field = 0
			return m, f.name.Focus()
		}
	}
	if f.kind == formItem && f.field == 0 {
		switch key.String() {
		case "down":
			if len(f.matches) > 0 {
				f.match = min(len(f.matches)-1, f.match+1)
				return m, nil
			}
		case "up":
			if len(f.matches) > 0 {
				f.match = max(0, f.match-1)
				return m, nil
			}
		case "tab":
			f.acceptMatch()
			f.field = 1
			f.name.Blur()
			return m, f.quantity.Focus()
		}
	}
	if f.kind == formItem && key.String() == "shift+tab" && f.field == 1 {
		f.field = 0
		f.quantity.Blur()
		return m, f.name.Focus()
	}
	if f.kind == formItem && key.String() == "tab" && f.field == 1 {
		f.field = 2
		f.quantity.Blur()
		return m, nil
	}
	if key.String() == "enter" {
		if f.kind == formItem && f.field == 0 {
			f.acceptMatch()
			f.field = 1
			f.name.Blur()
			return m, f.quantity.Focus()
		}
		if f.kind == formItem && f.field == 1 {
			f.field = 2
			f.quantity.Blur()
			return m, nil
		}
		name := strings.TrimSpace(f.name.Value())
		if name == "" {
			m.err = fmt.Errorf("le nom est obligatoire")
			return m, nil
		}
		var cmd tea.Cmd
		switch f.kind {
		case formList:
			if f.editing {
				cmd = func() tea.Msg { return actionMsg{"list", m.api.updateList(f.id, name)} }
			} else {
				cmd = func() tea.Msg { return actionMsg{"list", m.api.createList(name)} }
			}
		case formSection:
			if f.editing {
				cmd = func() tea.Msg { return actionMsg{"section", m.api.updateSection(f.id, name)} }
			} else {
				listID := m.lists[m.activeList].ID
				cmd = func() tea.Msg { return actionMsg{"section", m.api.createSection(listID, name)} }
			}
		case formItem:
			quantity := 1
			var err error
			if f.quantity.Value() != "" {
				quantity, err = strconv.Atoi(f.quantity.Value())
			}
			if err != nil || quantity < 1 {
				m.err = fmt.Errorf("la quantité doit être un entier supérieur ou égal à 1")
				return m, nil
			}
			if len(m.sections) == 0 {
				m.err = fmt.Errorf("une section est nécessaire")
				return m, nil
			}
			sectionID := m.sections[f.section].ID
			if f.editing {
				cmd = func() tea.Msg {
					return actionMsg{"item", m.api.updateItem(f.id, name, quantity, sectionID, f.originalSectionID)}
				}
			} else {
				cmd = func() tea.Msg { return actionMsg{"item", m.api.createItem(sectionID, name, quantity)} }
			}
		}
		m.form, m.focus, m.loading, m.err = nil, focusContent, true, nil
		return m, cmd
	}
	var cmd tea.Cmd
	if f.kind == formItem && f.field == 1 {
		f.quantity, cmd = f.quantity.Update(key)
	} else {
		f.name, cmd = f.name.Update(key)
		m.updateMatches()
	}
	return m, cmd
}

func (f *itemForm) acceptMatch() {
	if f.match >= 0 && f.match < len(f.matches) {
		f.name.SetValue(f.matches[f.match])
		f.section = f.matchSections[f.match]
	}
}

func (m model) updateConfirm(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc", "n":
		m.confirm = nil
		m.focus = focusContent
		if len(m.lists) == 0 {
			m.focus = focusLists
		}
	case "enter", "y", "o":
		c := *m.confirm
		m.confirm, m.loading = nil, true
		m.focus = focusContent
		kind := strings.TrimSuffix(c.kind, "s")
		if c.kind == "lists" {
			m.focus = focusLists
		}
		return m, func() tea.Msg { return actionMsg{kind, m.api.delete(c.kind, c.id)} }
	}
	return m, nil
}
