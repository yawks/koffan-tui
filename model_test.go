package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
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

func TestToggleKeepsCursorInOriginalColumn(t *testing.T) {
	for _, completed := range []bool{false, true} {
		for _, row := range []int{0, 1, 2} {
			for _, count := range []int{1, 3} {
				if row >= count {
					continue
				}
				t.Run(fmt.Sprintf("completed=%v/row=%d/count=%d", completed, row, count), func(t *testing.T) {
					m := model{
						lists:    []List{{ID: 1}},
						sections: []Section{{ID: 10}},
						expanded: map[int]bool{10: true}, focus: focusContent,
						row: row, showCompleted: true,
					}
					if completed {
						m.column = 1
						m.lists[0].Stats.Completed = count
					}
					for i := 0; i < count; i++ {
						m.sections[0].Items = append(m.sections[0].Items, Item{ID: i + 1, Completed: completed, Order: i})
					}
					updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
					got := updated.(model)
					wantRow := min(row, count-2)
					if got.column != m.column || got.row != wantRow || got.loading || cmd == nil {
						t.Fatalf("état inattendu: column=%d row=%d loading=%v", got.column, got.row, got.loading)
					}
					if got.sections[0].Items[row].Completed == completed {
						t.Fatal("le toggle doit être appliqué avant la réponse API")
					}
					updated, cmd = got.Update(toggleMsg{listID: 1, itemID: row + 1, completed: !completed})
					got = updated.(model)
					if cmd != nil || got.row != wantRow || got.column != m.column || len(got.pendingToggles) != 0 {
						t.Fatal("la réponse API ne doit ni recharger ni déplacer le curseur")
					}
				})
			}
		}
	}
}

func TestOptimisticToggleAPIResult(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprintf("failure=%v", fail), func(t *testing.T) {
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				if r.Method != http.MethodPost || r.URL.Path != "/items/1/toggle" {
					t.Errorf("requête inattendue: %s %s", r.Method, r.URL.Path)
				}
				if fail {
					http.Error(w, "toggle failed", http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()
			m := model{
				api:      &client{baseURL: server.URL, http: server.Client()},
				lists:    []List{{ID: 1}},
				sections: []Section{{ID: 10, Items: []Item{{ID: 1}, {ID: 2}}}},
				expanded: map[int]bool{10: true}, focus: focusContent, row: 0, showCompleted: true,
			}
			updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
			got := updated.(model)
			if got.lists[0].Stats.Completed != 1 {
				t.Fatal("compteur optimiste non mis à jour")
			}
			// Navigation remains available while the request is in flight.
			updated, _ = got.Update(tea.KeyPressMsg{Code: tea.KeyDown})
			got = updated.(model)
			row := got.row
			updated, next := got.Update(cmd())
			got = updated.(model)
			if next != nil || requests != 1 || got.loading || got.row != row {
				t.Fatal("le toggle doit effectuer une seule requête sans bloquer ni déplacer le curseur")
			}
			if got.sections[0].Items[0].Completed == fail || (got.err != nil) != fail {
				t.Fatalf("résultat du toggle incorrect: completed=%v err=%v", got.sections[0].Items[0].Completed, got.err)
			}
			wantCompleted := 1
			if fail {
				wantCompleted = 0
			}
			if got.lists[0].Stats.Completed != wantCompleted {
				t.Fatalf("compteur incorrect: %d", got.lists[0].Stats.Completed)
			}
		})
	}
}

func TestScrollLinesKeepsSelectedButtonVisible(t *testing.T) {
	lines := []string{"section", "columns", "one", "two", "three", "add"}
	visible := scrollLines(lines, 3, 3)

	if got := visible[len(visible)-1]; got != "add" {
		t.Fatalf("dernière ligne visible = %q, attendu add", got)
	}
}

func TestPendingTogglesAllowOtherItemsButPreventDuplicateRequests(t *testing.T) {
	m := model{
		lists:    []List{{ID: 1}},
		sections: []Section{{ID: 10, Items: []Item{{ID: 1}, {ID: 2}}}},
		expanded: map[int]bool{10: true}, focus: focusContent, row: 0, showCompleted: true,
	}
	space := tea.KeyPressMsg{Code: tea.KeySpace}
	updated, _ := m.Update(space)
	m = updated.(model)
	updated, cmd := m.Update(space)
	m = updated.(model)
	if cmd == nil || len(m.pendingToggles) != 2 || m.lists[0].Stats.Completed != 2 || m.row != -1 {
		t.Fatal("les articles suivants doivent pouvoir être cochés sans attendre")
	}
	m.column, m.row = 1, 0
	updated, cmd = m.Update(space)
	m = updated.(model)
	if cmd != nil || !m.sections[0].Items[0].Completed {
		t.Fatal("un article déjà en cours de toggle ne doit pas envoyer une seconde requête")
	}
}

func TestToggleWithCompletedItemsHidden(t *testing.T) {
	m := model{
		lists:    []List{{ID: 1}},
		sections: []Section{{ID: 10, Items: []Item{{ID: 1}}}},
		expanded: map[int]bool{10: true}, focus: focusContent, row: 0,
	}
	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	got := updated.(model)
	if got.column != 0 || got.row != -1 || got.showCompleted || !got.expanded[10] {
		t.Fatal("le curseur doit rester sur la section d'origine sans afficher les articles achetés")
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
