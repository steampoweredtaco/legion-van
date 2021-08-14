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

func NewRingBuffer(inCh, outCh chan interface{}) *RingBuffer {
	return &RingBuffer{
		inCh:  inCh,
		outCh: outCh,
	}
}

// RingBuffer throttle buffer for implement async channel.
type RingBuffer struct {
	inCh  chan interface{}
	outCh chan interface{}
}

func (r *RingBuffer) Run() {
	for v := range r.inCh {
		select {
		case r.outCh <- v:
		default:
			<-r.outCh // pop one item from outchan
			r.outCh <- v
		}
	}
	close(r.outCh)
}
