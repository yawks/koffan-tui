package main

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var (
	accent         = lipgloss.Color("#F472B6")
	muted          = lipgloss.Color("#777777")
	danger         = lipgloss.Color("#FF4D4D")
	panelStyle     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(accent).Padding(0, 1)
	titleStyle     = lipgloss.NewStyle().Bold(true).Foreground(accent)
	listTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(accent).Align(lipgloss.Center).Border(lipgloss.NormalBorder(), false, false, true, false).BorderForeground(accent)
	selected       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#20111A")).Background(accent)
	doneStyle      = lipgloss.NewStyle().Foreground(muted).Strikethrough(true)
	buttonStyle    = lipgloss.NewStyle().Foreground(accent).Bold(true)
	errorStyle     = lipgloss.NewStyle().Foreground(danger).Bold(true)
	keyStyle       = lipgloss.NewStyle().Foreground(accent).Border(lipgloss.NormalBorder(), false, true, false, true).BorderForeground(accent)
)

func (m model) View() tea.View {
	width, height := m.screenSize()
	width, height = max(width, 50), max(height, 10)
	leftWidth := min(40, width*2/5)
	rightWidth := width - leftWidth - 1
	footer := renderHelp(width)
	panelHeight := max(3, height-lipgloss.Height(footer))

	left := panelStyle.Width(leftWidth - 4).Height(panelHeight).MaxHeight(panelHeight).Render(m.renderLists())
	right := panelStyle.Width(rightWidth - 4).Height(panelHeight).MaxHeight(panelHeight).Render(m.renderContent())
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	base := lipgloss.JoinVertical(lipgloss.Left, body, footer)

	if m.form != nil {
		base = lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, m.renderForm())
	} else if m.confirm != nil {
		base = lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, m.renderConfirm())
	}
	v := tea.NewView(base)
	v.AltScreen = true
	return v
}

func (m model) screenSize() (width, height int) {
	width, height = m.width, m.height
	if width == 0 {
		width = 80
	}
	if height == 0 {
		height = 24
	}
	return width, height
}

func (m model) contentViewportHeight() int {
	width, height := m.screenSize()
	return max(3, height-lipgloss.Height(renderHelp(width))-7)
}

func (m model) renderLists() string {
	lines := []string{titleStyle.Render("Listes"), ""}
	width, _ := m.screenSize()
	width = max(width, 50)
	contentWidth := min(40, width*2/5) - 8
	if len(m.lists) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(muted).Render("Aucune liste"))
	}
	for i, list := range m.lists {
		icon := list.Icon
		if icon == "" {
			icon = "•"
		}
		stats := fmt.Sprintf("%d/%d", list.Stats.Completed, list.Stats.Total)
		nameWidth := max(1, contentWidth-lipgloss.Width(icon)-lipgloss.Width(stats)-2)
		name := truncate(list.Name, nameWidth)
		line := fmt.Sprintf("%s %-*s %s", icon, nameWidth, name, stats)
		if m.focus == focusLists && i == m.listCursor {
			line = selected.Render(line)
		}
		lines = append(lines, line)
	}
	lines = append(lines, "", buttonStyle.Render("[n] + Nouvelle liste"))
	return strings.Join(lines, "\n")
}

func (m model) renderContent() string {
	if len(m.lists) == 0 {
		return titleStyle.Render("Koffan") + "\n\nCréez votre première liste avec n."
	}
	list := m.lists[m.activeList]
	width := m.width
	if width == 0 {
		width = 80
	}
	leftWidth := min(40, width*2/5)
	// The right panel consumes four cells for its border and horizontal padding.
	titleWidth := max(1, width-leftWidth-9)
	total, completed := list.Stats.Total, list.Stats.Completed
	if !m.loading {
		total, completed = 0, 0
		for _, section := range m.sections {
			total += len(section.Items)
			for _, item := range section.Items {
				if item.Completed {
					completed++
				}
			}
		}
	}
	header := []string{
		listTitleStyle.Width(titleWidth).Render(fmt.Sprintf("%s %s   %d/%d à acheter", list.Icon, list.Name, total-completed, total)),
		"",
		buttonStyle.Render("[s] + Section    [a] + Article"),
		"",
	}
	body := make([]string, 0)
	if m.loading {
		body = append(body, "Chargement…")
	}
	for i := range m.sections {
		body = append(body, m.renderSection(i)...)
	}
	if len(m.sections) == 0 && !m.loading {
		body = append(body, lipgloss.NewStyle().Foreground(muted).Render("Aucune section"))
	}
	if m.err != nil {
		body = append(body, "", errorStyle.Render("Erreur : "+m.err.Error()))
	}
	body = scrollLines(body, m.contentScroll, m.contentViewportHeight())
	return strings.Join(append(header, body...), "\n")
}

func (m model) renderSection(index int) []string {
	s := m.sections[index]
	done := 0
	for _, item := range s.Items {
		if item.Completed {
			done++
		}
	}
	marker := "▶"
	if m.expanded[s.ID] {
		marker = "▼"
	}
	header := fmt.Sprintf("%s %s  %d/%d", marker, s.Name, done, len(s.Items))
	headerStyle := lipgloss.NewStyle().Bold(true)
	if len(s.Items) > 0 && done == len(s.Items) {
		headerStyle = headerStyle.Inherit(doneStyle)
	}
	if m.focus == focusContent && m.section == index && m.row == -1 {
		headerStyle = headerStyle.Inherit(selected)
	}
	header = headerStyle.Render(header)
	lines := []string{header}
	if !m.expanded[s.ID] {
		return lines
	}

	open, completed := splitItems(&s)
	width := m.width
	if width == 0 {
		width = 80
	}
	columnWidth := max(8, (width-38)/2)
	if !m.showCompleted {
		columnWidth = max(18, width-36)
	}
	leftTitle := lipgloss.NewStyle().Bold(true).Width(columnWidth).Render("À acheter")
	if m.showCompleted {
		rightTitle := lipgloss.NewStyle().Bold(true).Width(columnWidth).Render("Terminés")
		lines = append(lines, "  "+leftTitle+"│ "+rightTitle)
	} else {
		lines = append(lines, "  "+leftTitle)
	}
	rows := len(open)
	if m.showCompleted {
		rows = max(rows, len(completed))
	}
	for row := 0; row < rows; row++ {
		left := renderItemCell(open, row, columnWidth, m.focus == focusContent && m.section == index && m.row == row && m.column == 0, false)
		line := "  " + left
		if m.showCompleted {
			right := renderItemCell(completed, row, columnWidth, m.focus == focusContent && m.section == index && m.row == row && m.column == 1, true)
			line += "│ " + right
		}
		lines = append(lines, line)
	}
	add := "＋ Ajouter un article"
	if m.focus == focusContent && m.section == index && m.row == rows {
		add = selected.Render(add)
	} else {
		add = buttonStyle.Render(add)
	}
	lines = append(lines, "  "+add, "")
	return lines
}

func renderItemCell(items []Item, row, width int, active, completed bool) string {
	if row >= len(items) {
		return strings.Repeat(" ", width)
	}
	item := items[row]
	quantity := ""
	if item.Quantity > 1 {
		quantity = fmt.Sprintf(" × %d", item.Quantity)
	}
	prefix := "○ "
	if completed {
		prefix = "✓ "
	}
	line := truncate(prefix+item.Name+quantity, width)
	style := lipgloss.NewStyle().Width(width)
	if completed {
		style = style.Inherit(doneStyle)
	}
	if active {
		style = style.Inherit(selected)
	}
	return style.Render(line)
}

func (m model) renderForm() string {
	f := m.form
	title := map[formKind]string{formList: "Nouvelle liste", formSection: "Nouvelle section", formItem: "Nouvel article"}[f.kind]
	if f.editing {
		title = map[formKind]string{formList: "Modifier la liste", formSection: "Modifier la section", formItem: "Modifier l’article"}[f.kind]
	}
	lines := []string{titleStyle.Render(title), "", "Nom", f.name.View()}
	if f.kind == formItem {
		for i, match := range f.matches {
			line := "  " + match
			if i == f.match {
				line = selected.Render(line)
			}
			lines = append(lines, line)
			if i == 4 {
				break
			}
		}
		lines = append(lines, "", "Quantité", f.quantity.View())
		if len(m.sections) > 0 {
			section := m.sections[f.section].Name
			line := "Section    ◀ " + section + " ▶"
			if f.field == 2 {
				line = selected.Render(line)
			}
			lines = append(lines, "", line)
		}
	}
	if m.err != nil {
		lines = append(lines, "", errorStyle.Render(m.err.Error()))
	}
	lines = append(lines, "", lipgloss.NewStyle().Foreground(muted).Render("Entrée valider · Échap annuler"))
	return panelStyle.BorderForeground(accent).Width(46).Render(strings.Join(lines, "\n"))
}

func (m model) renderConfirm() string {
	c := m.confirm
	var message string
	if c.kind == "sections" && c.count > 0 {
		message = errorStyle.Render("⚠ ATTENTION") + fmt.Sprintf("\n\nLa section « %s » contient %d article(s).\nTous ses articles seront également supprimés.", c.name, c.count)
	} else if c.kind == "lists" && c.count > 0 {
		message = errorStyle.Render("⚠ ATTENTION") + fmt.Sprintf("\n\nSupprimer « %s » et ses %d article(s) ?", c.name, c.count)
	} else {
		message = fmt.Sprintf("Supprimer « %s » ?", c.name)
	}
	message += "\n\n" + errorStyle.Render("[o/entrée] Supprimer") + "   [n/échap] Annuler"
	return panelStyle.BorderForeground(danger).Width(58).Render(message)
}

func truncate(s string, width int) string {
	runes := []rune(s)
	if len(runes) <= width {
		return s
	}
	if width < 2 {
		return string(runes[:width])
	}
	return string(runes[:width-1]) + "…"
}

func helpKey(key, label string) string {
	return keyStyle.Width(9).Align(lipgloss.Center).Render(key) + " " + lipgloss.NewStyle().Foreground(muted).Render(label)
}

func renderHelp(width int) string {
	entries := [][2]string{
		{"tab", "panneau"}, {"↑↓", "naviguer"}, {"←→", "plier/colonne"},
		{"↵/esp", "agir"}, {"e", "modifier"}, {"⌫", "supprimer"},
		{"u", "terminés"}, {"c", "tout plier"}, {"n/s/a", "ajouter"}, {"q", "quitter"},
	}
	const desiredCellWidth = 25
	columns := max(1, width/desiredCellWidth)
	cellWidth := width / columns
	rows := make([]string, 0, (len(entries)+columns-1)/columns)
	for start := 0; start < len(entries); start += columns {
		end := min(len(entries), start+columns)
		cells := make([]string, 0, columns)
		for _, entry := range entries[start:end] {
			cells = append(cells, lipgloss.NewStyle().Width(cellWidth).Render(helpKey(entry[0], entry[1])))
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, cells...))
	}
	return strings.Join(rows, "\n")
}

func scrollLines(lines []string, offset, height int) []string {
	if len(lines) <= height {
		return lines
	}
	offset = min(max(0, offset), len(lines)-height)
	return lines[offset : offset+height]
}
