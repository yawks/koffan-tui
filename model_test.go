package main

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestBackspaceDeletesCompletedItem(t *testing.T) {
	m := model{
		sections:      []Section{{ID: 10, Items: []Item{{ID: 2, Name: "Bananes", Completed: true}}}},
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
				{ID: 1, Name: "Pommes", Completed: false},
				{ID: 2, Name: "Bananes", Completed: true},
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
