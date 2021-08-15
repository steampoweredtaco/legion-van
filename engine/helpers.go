package engine

import (
	"sort"
	"strings"
)

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
func glassesOdds(prefixes []string) (odds float64) {
	if len(prefixes) == 0 {
		odds = 1
		return
	}
	glassesOdds := 0
	for _, prefix := range prefixes {
		if strings.HasPrefix("glasses-nerd-green-[w-1].svg", prefix) {
			glassesOdds += 512
		}
		if strings.HasPrefix("sunglasses-aviator-yellow-[removes-eyes][w-1].svg", prefix) {
			glassesOdds += 512
		}
		if strings.HasPrefix("sunglasses-thug-[removes-eyes][w-1].svg", prefix) {
			glassesOdds += 520
		}
		if strings.HasPrefix("eye-patch-[w-0.5].svg", prefix) {
			glassesOdds += 256
		}
		if strings.HasPrefix("glasses-nerd-cyan-[w-1].svg", prefix) {
			glassesOdds += 512
		}
		if strings.HasPrefix("glasses-nerd-pink-[w-1].svg", prefix) {
			glassesOdds += 512
		}
		if strings.HasPrefix("monocle-[w-0.5].svg", prefix) {
			glassesOdds += 256
		}
		if strings.HasPrefix("sunglasses-aviator-cyan-[removes-eyes][w-1].svg", prefix) {
			glassesOdds += 512
		}
		if strings.HasPrefix("sunglasses-aviator-green-[removes-eyes][w-1].svg", prefix) {
			glassesOdds += 512
		}
	}
	odds = .25 * float64(glassesOdds) / 4096.0
	return
}

func hatsOdds(prefixes []string) (odds float64) {
	if len(prefixes) == 0 {
		odds = 1
		return
	}
	hatsOdds := 0
	for _, prefix := range prefixes {
		if strings.HasPrefix("beanie-long-[colorable-random][w-1].svg", prefix) {
			hatsOdds += 212
		}
		if strings.HasPrefix("cap-banano-[w-0.8].svg", prefix) {
			hatsOdds += 169
		}
		if strings.HasPrefix("cap-kappa-[w-0.8].svg", prefix) {
			hatsOdds += 169
		}
		if strings.HasPrefix("cap-smug-[w-0.8].svg", prefix) {
			hatsOdds += 169
		}
		if strings.HasPrefix("hat-jester-[unique][w-0.125].svg", prefix) {
			hatsOdds += 27
		}
		if strings.HasPrefix("bandana-[w-1].svg", prefix) {
			hatsOdds += 212
		}
		if strings.HasPrefix("cap-carlos-[w-0.8].svg", prefix) {
			hatsOdds += 169
		}
		if strings.HasPrefix("crown-[unique][w-0.225].svg", prefix) {
			hatsOdds += 48
		}
		if strings.HasPrefix("fedora-long-[w-1].svg", prefix) {
			hatsOdds += 212
		}
		if strings.HasPrefix("cap-hng-plus-[unique][w-0.125].svg", prefix) {
			hatsOdds += 27
		}
		if strings.HasPrefix("fedora-[w-1].svg", prefix) {
			hatsOdds += 212
		}
		if strings.HasPrefix("beanie-banano-[w-1].svg", prefix) {
			hatsOdds += 212
		}
		if strings.HasPrefix("beanie-hippie-[unique][w-0.125].svg", prefix) {
			hatsOdds += 27
		}
		if strings.HasPrefix("cap-[w-0.8].svg", prefix) {
			hatsOdds += 169
		}
		if strings.HasPrefix("cap-backwards-[w-1].svg", prefix) {
			hatsOdds += 212
		}
		if strings.HasPrefix("cap-bebe-[w-0.8].svg", prefix) {
			hatsOdds += 169
		}
		if strings.HasPrefix("cap-hng-[w-0.8].svg", prefix) {
			hatsOdds += 169
		}
		if strings.HasPrefix("hat-cowboy-[w-1].svg", prefix) {
			hatsOdds += 212
		}
		if strings.HasPrefix("helmet-viking-[w-1].svg", prefix) {
			hatsOdds += 224
		}
		if strings.HasPrefix("beanie-[w-1].svg", prefix) {
			hatsOdds += 212
		}
		if strings.HasPrefix("beanie-long-banano-[colorable-random][w-1].svg", prefix) {
			hatsOdds += 212
		}
		if strings.HasPrefix("cap-pepe-[w-0.8].svg", prefix) {
			hatsOdds += 169
		}
		if strings.HasPrefix("cap-rick-[w-0.8].svg", prefix) {
			hatsOdds += 169
		}
		if strings.HasPrefix("cap-smug-green-[w-0.8].svg", prefix) {
			hatsOdds += 169
		}
		if strings.HasPrefix("cap-thonk-[w-0.8].svg", prefix) {
			hatsOdds += 169
		}
	}
	odds = .35 * float64(hatsOdds) / 4096.0
	return
}
func miscOdds(prefixes []string) (odds float64) {
	if len(prefixes) == 0 {
		odds = 1
		return
	}
	miscOdds := 0
	for _, prefix := range prefixes {
		if strings.HasPrefix("banana-right-hand-[above-hands][removes-hand-right][w-1].svg", prefix) {
			miscOdds += 363
		}
		if strings.HasPrefix("camera-[above-shirts-pants][w-1].svg", prefix) {
			miscOdds += 363
		}
		if strings.HasPrefix("club-[above-hands][removes-hands][w-1].svg", prefix) {
			miscOdds += 363
		}
		if strings.HasPrefix("flamethrower-[removes-hands][above-hands][w-0.04].svg", prefix) {
			miscOdds += 15
		}
		if strings.HasPrefix("guitar-[above-hands][removes-left-hand][w-1].svg", prefix) {
			miscOdds += 363
		}
		if strings.HasPrefix("microphone-[above-hands][removes-hand-right][w-1].svg", prefix) {
			miscOdds += 363
		}
		if strings.HasPrefix("necklace-boss-[above-shirts-pants][w-0.75].svg", prefix) {
			miscOdds += 273
		}
		if strings.HasPrefix("tie-pink-[above-shirts-pants][w-1].svg", prefix) {
			miscOdds += 363
		}
		if strings.HasPrefix("whisky-right-[above-hands][removes-hand-right][w-0.5].svg", prefix) {
			miscOdds += 190
		}
		if strings.HasPrefix("banana-hands-[above-hands][removes-hands][w-1].svg", prefix) {
			miscOdds += 363
		}
		if strings.HasPrefix("bowtie-[above-hands][w-1].svg", prefix) {
			miscOdds += 363
		}
		if strings.HasPrefix("gloves-white-[above-hands][removes-hands][w-1].svg", prefix) {
			miscOdds += 363
		}
		if strings.HasPrefix("tie-cyan-[above-shirts-pants][w-1].svg", prefix) {
			miscOdds += 363
		}
	}
	odds = .3 * float64(miscOdds) / 4096.0
	return
}
func mouthOdds(prefixes []string) (odds float64) {
	if len(prefixes) == 0 {
		odds = 1
		return
	}
	mouthOdds := 0
	for _, prefix := range prefixes {

		if strings.HasPrefix("smile-tongue-[w-0.5].svg", prefix) {
			mouthOdds += 372
		}
		if strings.HasPrefix("cigar-[w-0.5].svg", prefix) {
			mouthOdds += 369
		}
		if strings.HasPrefix("confused-[w-1].svg", prefix) {
			mouthOdds += 737
		}
		if strings.HasPrefix("joint-[unique][w-0.06].svg", prefix) {
			mouthOdds += 45
		}
		if strings.HasPrefix("meh-[w-1].svg", prefix) {
			mouthOdds += 737
		}
		if strings.HasPrefix("pipe-[w-0.5].svg", prefix) {
			mouthOdds += 369
		}
		if strings.HasPrefix("smile-big-teeth-[w-1].svg", prefix) {
			mouthOdds += 737
		}
		if strings.HasPrefix("smile-normal-[w-1].svg", prefix) {
			mouthOdds += 737
		}
	}
	odds = float64(mouthOdds) / 4096.0
	return
}
func clothsOdds(prefixes []string) (odds float64) {
	if len(prefixes) == 0 {
		odds = 1
		return
	}
	shirtOdds := 0
	for _, prefix := range prefixes {

		if strings.HasPrefix("overalls-blue[w-1].svg", prefix) {
			shirtOdds += 683
		}
		if strings.HasPrefix("overalls-red[w-1].svg", prefix) {
			shirtOdds += 683
		}
		if strings.HasPrefix("pants-business-blue-[removes-legs][w-1].svg", prefix) {
			shirtOdds += 683
		}
		if strings.HasPrefix("pants-flower-[removes-legs][w-1].svg", prefix) {
			shirtOdds += 683
		}
		if strings.HasPrefix("tshirt-long-stripes-[colorable-random][w-1].svg", prefix) {
			shirtOdds += 683
		}
		if strings.HasPrefix("tshirt-short-white[w-1].svg", prefix) {
			shirtOdds += 686
		}
	}
	odds = .25 * float64(shirtOdds) / 4096.0
	return
}
func feetOdds(prefixes []string) (odds float64) {
	if len(prefixes) == 0 {
		odds = 1
		return
	}
	shoeOdds := 0
	for _, prefix := range prefixes {
		if strings.HasPrefix("socks-v-stripe-[colorable-random][removes-feet][w-1].svg", prefix) {
			shoeOdds += 686
		}
		if strings.HasPrefix("sneakers-blue-[removes-feet][w-1].svg", prefix) {
			shoeOdds += 683
		}
		if strings.HasPrefix("sneakers-green-[removes-feet][w-1].svg", prefix) {
			shoeOdds += 683
		}
		if strings.HasPrefix("sneakers-red-[removes-feet][w-1].svg", prefix) {
			shoeOdds += 683
		}
		if strings.HasPrefix("sneakers-swagger-[removes-feet][w-1].svg", prefix) {
			shoeOdds += 683
		}
		if strings.HasPrefix("socks-h-stripe-[removes-feet][w-1].svg", prefix) {
			shoeOdds += 683
		}
	}
	odds = .22 * float64(shoeOdds) / 4096.0
	return
}
func tailsOdds(prefixes []string) (odds float64) {
	if len(prefixes) == 0 {
		odds = 1
		return
	}
	tailsOdds := 0
	for _, prefix := range prefixes {
		if strings.HasPrefix("tail-sock-[colorable-random][w-1].svg", prefix) {
			tailsOdds += 4096
		}
	}
	// yes this could be just .2 but it is code generated, come on
	odds = .2 * float64(tailsOdds) / 4096.0
	return
}

func getSmallestPrefixes(filters []string) []string {
	sort.Strings(filters)
	newFilter := make([]string, 0)
	for _, filter := range filters {
		previous := ""
		if previous == "" {
			previous = filter
			newFilter = append(newFilter, filter)
			continue
		}
		if strings.HasPrefix(filter, previous) {
			continue
		}
		previous = filter
		newFilter = append(newFilter, filter)
	}
	return newFilter
}

func SimplifyFilters(filter *CmdLineFilter) {
	filter.Glasses = getSmallestPrefixes(filter.Glasses)
	filter.Hat = getSmallestPrefixes(filter.Hat)
	filter.Misc = getSmallestPrefixes(filter.Misc)
	filter.Mouth = getSmallestPrefixes(filter.Mouth)
	filter.Cloths = getSmallestPrefixes(filter.Cloths)
	filter.Feet = getSmallestPrefixes(filter.Feet)
	filter.Tail = getSmallestPrefixes(filter.Tail)

}

func GetOdds(filter CmdLineFilter) float64 {

	// most of this was generated from hacked version of the monkey server.
	gOdds := glassesOdds(filter.Glasses)
	hOdds := hatsOdds(filter.Hat)
	mOdds := miscOdds(filter.Misc)
	oOdds := mouthOdds(filter.Mouth)
	cOdds := clothsOdds(filter.Cloths)
	fOdds := feetOdds(filter.Feet)
	tOdds := tailsOdds(filter.Tail)

	totalOdds := gOdds * hOdds * mOdds * oOdds * cOdds * tOdds * fOdds
	return 1 / totalOdds
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
