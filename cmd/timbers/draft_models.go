package main

import (
	"slices"
	"strings"

	"github.com/gorewood/timbers/internal/llm"
	"github.com/gorewood/timbers/internal/output"
)

// runDraftModels lists providers, model aliases, and required API keys.
func runDraftModels(printer *output.Printer) error {
	infos := llm.ProviderInfos()

	if printer.IsJSON() {
		type jsonAlias struct {
			Alias string `json:"alias"`
			Model string `json:"model"`
		}
		type jsonProvider struct {
			Provider string      `json:"provider"`
			EnvVar   string      `json:"env_var,omitempty"`
			Aliases  []jsonAlias `json:"aliases"`
		}

		providers := make([]jsonProvider, 0, len(infos))
		for _, info := range infos {
			jp := jsonProvider{Provider: info.Name, EnvVar: info.EnvVar}
			for _, a := range sortedAliases(info.Aliases) {
				jp.Aliases = append(jp.Aliases, jsonAlias{Alias: a[0], Model: a[1]})
			}
			providers = append(providers, jp)
		}
		return printer.Success(map[string]any{"providers": providers})
	}

	printer.Print("Providers:\n")
	for _, info := range infos {
		auth := "(no API key needed)"
		if info.EnvVar != "" {
			auth = info.EnvVar
		}
		printer.Print("  %-12s %s\n", info.Name, auth)
		for _, a := range sortedAliases(info.Aliases) {
			printer.Print("    %-12s → %s\n", a[0], a[1])
		}
		printer.Print("\n")
	}
	return nil
}

// sortedAliases returns alias→model pairs sorted by alias name.
func sortedAliases(aliases map[string]string) [][2]string {
	pairs := make([][2]string, 0, len(aliases))
	for alias, model := range aliases {
		pairs = append(pairs, [2]string{alias, model})
	}
	slices.SortFunc(pairs, func(a, b [2]string) int {
		return strings.Compare(a[0], b[0])
	})
	return pairs
}
