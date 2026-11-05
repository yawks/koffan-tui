package main

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestHideCompletedCollapsesSectionsWithoutOpenItems(t *testing.T) {
	m := model{
		sections: []Section{
			{ID: 1, Items: []Item{{ID: 1, Completed: true}}},
			{ID: 2},
			{ID: 3, Items: []Item{{ID: 2}, {ID: 3, Completed: true}}},
			{ID: 4, Items: []Item{{ID: 4}}},
		},
		expanded:      map[int]bool{1: true, 2: true, 3: true, 4: false},
		focus:         focusContent,
		row:           0,
		column:        1,
		showCompleted: true,
	}
	key := tea.KeyPressMsg{Code: 'u', Text: "u"}

	updated, _ := m.updateContent(key)
	got := updated.(model)
	if got.showCompleted || got.column != 0 || got.row != -1 {
		t.Fatalf("état inattendu: showCompleted=%v column=%d row=%d", got.showCompleted, got.column, got.row)
	}
	if got.expanded[1] || got.expanded[2] || !got.expanded[3] || got.expanded[4] {
		t.Fatalf("état des sections inattendu: %#v", got.expanded)
	}
	if lines := got.renderSection(0); len(lines) != 1 {
		t.Fatalf("la section achetée doit être repliée: %#v", lines)
	}

	updated, _ = got.updateContent(key)
	got = updated.(model)
	if !got.showCompleted || !got.expanded[3] || got.expanded[4] {
		t.Fatalf("état inattendu après réaffichage: showCompleted=%v expanded=%#v", got.showCompleted, got.expanded)
	}
}

func TestCompletedHelpLabelTracksVisibility(t *testing.T) {
	for _, visible := range []bool{false, true} {
		m := model{showCompleted: visible}
		want, unwanted := "show bought", "hide bought"
		if visible {
			want, unwanted = unwanted, want
		}
		help := m.renderHelp(80)
		if !strings.Contains(help, want) || strings.Contains(help, unwanted) {
			t.Fatalf("aide incorrecte pour showCompleted=%v: %s", visible, help)
		}
	}
}

func TestBackspaceDeletesCompletedItem(t *testing.T) {
	m := model{
		sections:      []Section{{ID: 10, Items: []Item{{ID: 2, Name: "Bananas", Completed: true}}}},
		expanded:      map[int]bool{10: true},
		focus:         focusContent,
		row:           0,
		column:        1,
		showCompleted: true,
	}

	updated, _ := m.updateContent(tea.KeyPressMsg{Code: tea.KeyBackspace, Mod: tea.ModShift})
	got := updated.(model)
	if got.confirm == nil || got.confirm.kind != "items" || got.confirm.id != 2 {
		t.Fatalf("confirmation inattendue: %#v", got.confirm)
	}
}

func TestFollowItemAfterCompletedColumnChange(t *testing.T) {
	m := model{
		sections: []Section{{
			ID: 10,
			Items: []Item{
				{ID: 1, Name: "Apples", Completed: false},
				{ID: 2, Name: "Bananas", Completed: true},
			},
		}},
		expanded: make(map[int]bool),
	}

	m.followItem(2)

	if m.section != 0 || m.row != 0 || m.column != 1 || !m.expanded[10] {
		t.Fatalf("curseur inattendu: section=%d row=%d column=%d expanded=%v", m.section, m.row, m.column, m.expanded[10])
	}
}

func TestScrollLinesKeepsSelectedButtonVisible(t *testing.T) {
	lines := []string{"section", "columns", "one", "two", "three", "add"}
	visible := scrollLines(lines, 3, 3)

	if got := visible[len(visible)-1]; got != "add" {
		t.Fatalf("dernière ligne visible = %q, attendu add", got)
	}
}

func TestScrollMovesOnlyAtViewportEdges(t *testing.T) {
	items := make([]Item, 10)
	for i := range items {
		items[i].ID = i + 1
	}
	m := model{
		sections:      []Section{{ID: 10, Items: items}},
		expanded:      map[int]bool{10: true},
		row:           7,
		height:        17, // six visible content lines
		contentScroll: 4,
	}

	m.row = 6
	m.ensureContentCursorVisible()
	if m.contentScroll != 4 {
		t.Fatalf("scroll déplacé trop tôt: %d", m.contentScroll)
	}
	m.row = 1
	m.ensureContentCursorVisible()
	if m.contentScroll != 3 {
		t.Fatalf("scroll attendu au bord supérieur: %d", m.contentScroll)
	}
}

func TestMoveDownSkipsEmptyRowsInShortColumn(t *testing.T) {
	m := model{
		sections: []Section{{
			ID: 10,
			Items: []Item{
				{ID: 1},
				{ID: 2, Completed: true},
				{ID: 3, Completed: true},
				{ID: 4, Completed: true},
			},
		}},
		expanded:      map[int]bool{10: true},
		row:           0,
		column:        0,
		showCompleted: true,
	}

	m.moveDown()

	if m.row != 3 {
		t.Fatalf("ligne = %d, attendu le bouton ajouter à la ligne 3", m.row)
	}
	m.moveUp()
	if m.row != 0 {
		t.Fatalf("retour = %d, attendu le dernier article de gauche à la ligne 0", m.row)
	}
}

func TestToggleAllSections(t *testing.T) {
	m := model{
		sections: []Section{{ID: 1}, {ID: 2}},
		expanded: map[int]bool{1: true},
		row:      3,
	}

	m.toggleAllSections()
	if !m.expanded[1] || !m.expanded[2] || m.row != -1 {
		t.Fatalf("toutes les sections auraient dû être dépliées: %#v", m.expanded)
	}
	m.toggleAllSections()
	if m.expanded[1] || m.expanded[2] {
		t.Fatalf("toutes les sections auraient dû être repliées: %#v", m.expanded)
	}
}

func TestFoldHelpLabelTracksSectionState(t *testing.T) {
	for _, tt := range []struct {
		name     string
		sections []Section
		expanded map[int]bool
	}{
		{name: "empty"},
		{name: "collapsed", sections: []Section{{ID: 1}, {ID: 2}}, expanded: make(map[int]bool)},
		{name: "mixed", sections: []Section{{ID: 1}, {ID: 2}}, expanded: map[int]bool{1: true}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m := model{sections: tt.sections, expanded: tt.expanded}
			if help := m.renderHelp(80); !strings.Contains(help, "unfold all") {
				t.Fatalf("libellé unfold all attendu: %s", help)
			}
			if len(tt.sections) == 0 {
				return
			}
			key := tea.KeyPressMsg{Code: 'c', Text: "c"}
			updated, _ := m.updateContent(key)
			m = updated.(model)
			if help := m.renderHelp(80); !strings.Contains(help, "fold all") || strings.Contains(help, "unfold all") {
				t.Fatalf("libellé fold all attendu: %s", help)
			}
			updated, _ = m.updateContent(key)
			m = updated.(model)
			if help := m.renderHelp(80); !strings.Contains(help, "unfold all") {
				t.Fatalf("libellé unfold all attendu après repli: %s", help)
			}
		})
	}
}

func TestAutocompleteSelectsItemSection(t *testing.T) {
	m := model{
		sections: []Section{
			{ID: 1, Name: "Fruits"},
			{ID: 2, Name: "Épicerie", Items: []Item{{Name: "Pâtes"}}},
		},
		expanded: make(map[int]bool),
		focus:    focusContent,
	}
	m.openForm(formItem, 1)
	m.form.name.SetValue("Pât")
	m.updateMatches()

	updated, _ := m.updateForm(tea.KeyPressMsg{Code: tea.KeyEnter})
	got := updated.(model)
	if got.form.name.Value() != "Pâtes" || got.form.section != 1 {
		t.Fatalf("suggestion = %q, section = %d", got.form.name.Value(), got.form.section)
	}
}
