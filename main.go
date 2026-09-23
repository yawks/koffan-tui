package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
)

func main() {
	cfg, path, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "configuration %s: %v\n", path, err)
		os.Exit(1)
	}

	if _, err := tea.NewProgram(newModel(newClient(cfg))).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
