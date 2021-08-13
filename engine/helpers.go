package engine

import "strings"

type CmdLineFilter struct {
	HelpVanity bool     `long:"help-vanity" short:"V"`
	Hat        []string `short:"H" description:"hat option. See --help-vanity for list"`
	Glasses    []string `short:"G" description:"glasses option. See --help-vanity for list"`
	Mouth      []string `short:"O" description:"mouth option. See --help-vanity for list"`
	Cloths     []string `short:"C" description:"cloths option. See --help-vanity for list"`
	Feet       []string `short:"F" description:"feet option. See --help-vanity for list"`
	Tail       []string `short:"T" description:"tail option. See --help-vanity for list"`
	Misc       []string `short:"M" description:"misc  option. See --help-vanity for list"`
}

func fitlerMatchAny(options []string, s string) bool {
	// filters always match empty options
	if len(options) == 0 {
		return true
	}
	for _, prefix := range options {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}

func matchFilters(monkey MonkeyStats, filter CmdLineFilter) bool {
	return fitlerMatchAny(filter.Misc, monkey.Misc) &&
		fitlerMatchAny(filter.Cloths, monkey.ShirtPants) &&
		fitlerMatchAny(filter.Feet, monkey.Shoes) &&
		fitlerMatchAny(filter.Glasses, monkey.Glasses) &&
		fitlerMatchAny(filter.Hat, monkey.Hat) &&
		fitlerMatchAny(filter.Mouth, monkey.Mouth) &&
		fitlerMatchAny(filter.Tail, monkey.Tail)

}
