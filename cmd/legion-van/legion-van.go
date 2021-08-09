package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	_ "net/http/pprof"

	"github.com/Pallinder/go-randomdata"
	"github.com/bbedward/nano/address"
	"github.com/gdamore/tcell/v2"
	"github.com/jessevdk/go-flags"
	log "github.com/sirupsen/logrus"
	"github.com/steampoweredtaco/legion-van/bananoutils"
	"github.com/steampoweredtaco/legion-van/gui"
	legionImage "github.com/steampoweredtaco/legion-van/image"
	"github.com/ugorji/go/codec"
)

func (*targetFormat) String() string {
	return "svg"
}

func (targetFmt *targetFormat) UnmarshalFlag(value string) error {
	*targetFmt = targetFormat(strings.ToLower(value))
	return nil
}

type targetFormat string

var config struct {
	HowLongToRun   time.Duration `long:"duration" description:"How long to run the search for" default:"1m"`
	MaxRequests    uint          `long:"max_requests" description:"Maxiumum outstanding parallel requests to monkeyapi" default:"4"`
	DisablePreview bool          `long:"disable_review" description:"Disable the gui and preview of monkeys"`
	Format         targetFormat  `long:"image_format" description:"Set the target image format for saving monkey found in options are svg or png. svg is faster" default:"png" choice:"png" choice:"svg"`
	BatchSize      uint          `long:"batch_size" description:"Number of monkeys to test per batch request, higher or lower may affect performance" default:"2500"`
	Debug          bool          `long:"debug" description:"Changes logging and makes terminal virtual for debugging issues."`
	MonkeyServer   string        `long:"monkey_api" description:"To change the backend monkey server, defaults to the official one." default:"https://monkey.banano.cc"`
}

var filter struct {
	HelpVanity bool     `long:"help-vanity" short:"V"`
	Hat        []string `short:"H" description:"hat option. See --help-vanity for list"`
	Glasses    []string `short:"G" description:"glasses option. See --help-vanity for list"`
	Mouth      []string `short:"O" description:"mouth option. See --help-vanity for list"`
	Cloths     []string `short:"C" description:"cloths option. See --help-vanity for list"`
	Feet       []string `short:"F" description:"feet option. See --help-vanity for list"`
	Tail       []string `short:"T" description:"tail option. See --help-vanity for list"`
	Misc       []string `short:"M" description:"misc  option. See --help-vanity for list"`
}

func printVanityFilterUsage() {
	usage := `
Vanity Filters Usage
--------------------
Each unique vanity -A,-C,-M (etc) if present requires at least one of
those options to be present in the found monKey.

WARNING: If you make a typo you will never find a monkey, validation and odds
coming to a monKey near you soon.

For example the following options would require a flamethrower and cap
./legion-van -M flamethrower -H cap
===
Every vanity filter option may be specified multiple times. If they are,
the monkey matched may have any of those options for that vanity.

For example, the following options would require a flamethrower or camera
./legion-van -M flamethrower -M camera
===
Mutliple unique vanity options and multiple instances of the same vanity
may be supplied and resutls in a match if both of those unique vanity options
are present but in any combination of the multiple choices supplied to each 
option.

For example the following requires a monkey with either a flamethrower or camera
AND a cap or beanie:
./legion-van -M flamethrower -M camera -C cap -C beanie

===
Each option choice maybe abbrevated to match the more general option.

For example, to get money's that match a pink tie:
./legion-van -M flamethrower -M tie-pink

But to get any color tie:
./legion-van -M flamethrower -M tie

Vanity Filter Choices
---------------------
Each option maybe supplied as -<option> <value> or -<option>=<value>
To select multiple choices that begin with the same starting letters just
use those starting letters (ie: -M tie matches -M tie-pink -M tie-cyan)

-H <Hat_option>:    values:
                        bandana
                        beanie-,beanie-banano,beanie-hippie,beanie-long
                        cap-,cap-backwards,cap-bebe,cap-carlos,cap-hng,
                        cap-hng-plus,cap-kappa,cap-pepe,cap-rick,camp-smug,
                        cap-smug-green,cap-thonk
                        crown
                        fedora-,fedora-long
                        hat-cowboy,hat-jester
                        helmet-viking

-G <Glasses>:       values:
                        eye-patch
                        glasses-nerd-cyan,glasses-nerd-green,glasses-nerd-pink
                        monocle,
                        sunglasses-aviator-cyan,sunglasses-aviator-green,
                        sunglasses-aviator-yellow,sunglasses-thug

-O <mOuths>:        values:
                        cigar
                        confused
                        joint
                        meh
                        pipe
                        smile-big-teeth,smile-normal,smile-tongue

-C <Cloths>         values:
                        overalls-blue,overalls-red
                        pants-buisness-blue,pants-flower
                        tshirt-long-stripes,tshirt-short-white

-F <Feet>           values:
                        sneakers-blue,sneakers-green,sneakers-red,
                        sneakers-swagger
                        socks-h-stripe,socks-v-stripe
                
-T <Tail>           values:
                        tail-sock

-M <Misc>           values:
                        banana-hands,banana-right-hand
                        bowtie
                        camera
                        club
                        flamethrower
                        gloves-white
                        guitar
                        microphone
                        necklace
                        tie-cyan,tie-pink
                        whisky
`

	fmt.Println(usage)
}

func makeLower(as []string) []string {
	res := make([]string, len(as))
	for i, value := range as {
		res[i] = strings.ToLower(value)
	}
	return res
}

func parseFlags() {
	parser := flags.NewParser(&config, flags.Default)
	parser.AddGroup("Vanity Filters", "These options allow for filtering of specific monKey features.", &filter)
	_, err := parser.Parse()

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if filter.HelpVanity {
		printVanityFilterUsage()
		os.Exit(1)
	}

	filter.Hat = makeLower(filter.Hat)
	filter.Glasses = makeLower(filter.Glasses)
	filter.Mouth = makeLower(filter.Mouth)
	filter.Cloths = makeLower(filter.Cloths)
	filter.Feet = makeLower(filter.Feet)
	filter.Tail = makeLower(filter.Tail)
	filter.Misc = makeLower(filter.Misc)

}

var (
	jsonHandler = new(codec.JsonHandle)
)

var (
	httpTransport *http.Transport
	httpClient    *http.Client
)

func GenerateAddress() string {
	pub, _ := address.GenerateKey()
	return strings.Replace(string(address.PubKeyToAddress(pub)), "nano_", "ban_", -1)
}

type MonkeyBase struct {
	PublicAddress   string
	PrivateKey      string
	BackgroundColor string `json:"background_color"`
	Glasses         string `json:"glasses"`
	Hat             string `json:"hat"`
	Misc            string `json:"misc"`
	Mouth           string `json:"mouth"`
	ShirtPants      string `json:"shirt_pants"`
	Shoes           string `json:"shoes"`
	Tail            string `json:"tail_accessory"`
	SillyName       string `json:"silly_name"`
}

type MonkeyStats struct {
	MonkeyBase
	Additional map[string]interface{}
}

func (monkey *MonkeyStats) UnmarshalJSON(data []byte) (err error) {
	monkey.Additional = make(map[string]interface{})
	err = codec.NewDecoderBytes(data, jsonHandler).Decode(&monkey.Additional)
	if err != nil {
		return fmt.Errorf("could not parse monkeystat: %w", err)
	}
	err = json.Unmarshal(data, &monkey.MonkeyBase)
	if err != nil {
		return fmt.Errorf("could not parse monkeystat main monkey: %w", err)
	}
	monkey.SillyName = randomdata.SillyName()
	return
}

func (monkey MonkeyStats) MarshalJSON() ([]byte, error) {
	monkey.Additional["public_address"] = monkey.PublicAddress
	monkey.Additional["private_key"] = monkey.PrivateKey
	data := make([]byte, 1000)
	err := codec.NewEncoderBytes(&data, jsonHandler).Encode(&monkey.Additional)
	if err != nil {
		return nil, err
	}
	return data, nil
}

type walletsDB struct {
	publicAccounts              []string
	publicAccountToWalletLookup map[string]string
}

func generateManyWallets(amount uint) walletsDB {
	var accountsToWalletKey = make(map[string]string, amount)
	accounts := make([]string, 0, amount)

	for i := uint(0); i < amount; i++ {
		privateWalletSeed, publicAccount, err := bananoutils.GeneratePrivateKeyAndFirstPublicAddress()
		if err != nil {
			panic(err)
		}
		publicAccountStr := string(publicAccount)

		accountsToWalletKey[publicAccountStr] = privateWalletSeed
		accounts = append(accounts, publicAccountStr)
	}
	return walletsDB{publicAccounts: accounts, publicAccountToWalletLookup: accountsToWalletKey}
}

func (db walletsDB) getAccounts() []string {
	return db.publicAccounts
}

func (db walletsDB) lookupWalletSeed(publicAddress string) string {
	return db.publicAccountToWalletLookup[publicAddress]
}

func (db walletsDB) encodeAccountsAsJSON() io.Reader {
	data := make([]byte, len(db.publicAccounts)*64)
	jsonStruct := make(map[string][]string)
	jsonStruct["addresses"] = db.publicAccounts
	err := codec.NewEncoderBytes(&data, jsonHandler).Encode(jsonStruct)
	if err != nil {
		log.Fatalf("could not marshal addresses for request %s", err)
	}
	return bytes.NewBuffer(data)
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

func matchFilters(monkey MonkeyStats) bool {
	return fitlerMatchAny(filter.Misc, monkey.Misc) &&
		fitlerMatchAny(filter.Cloths, monkey.ShirtPants) &&
		fitlerMatchAny(filter.Feet, monkey.Shoes) &&
		fitlerMatchAny(filter.Glasses, monkey.Glasses) &&
		fitlerMatchAny(filter.Hat, monkey.Hat) &&
		fitlerMatchAny(filter.Mouth, monkey.Mouth) &&
		fitlerMatchAny(filter.Tail, monkey.Tail)

}
func getMonkeyData(ctx context.Context, monkeysPerRequest uint, monkeySendChan chan<- MonkeyStats, deltaChan chan<- gui.Stats, wg *sync.WaitGroup) {
	defer wg.Done()
	getStatsURL := fmt.Sprintf("%s/api/v1/monkey/dtl", config.MonkeyServer)
	var totalCount uint64
	var survivorCount uint64
	raidName := strings.Title(randomdata.Adjective() + " " + randomdata.Noun())

	log.Infof("Raiding with %s clan", raidName)
main:
	for {

		// Exit early there are web errors and then the app shutsdown
		select {
		case <-ctx.Done():
			log.Infof("stopping the %s raid.", raidName)
			break main
		default:
		}
		wallets := generateManyWallets(monkeysPerRequest)
		jsonBody := make(map[string][]string)
		jsonBody["addresses"] = wallets.getAccounts()
		jsonReader := wallets.encodeAccountsAsJSON()

		request, err := http.NewRequestWithContext(ctx, "POST", getStatsURL, jsonReader)
		if err != nil {
			log.Errorf("Could not get monkey stats %s", err)
			continue
		}
		request.Header.Set("Content-Type", "application/json")
		response, err := httpClient.Do(request)
		if err != nil {
			log.Errorf("Could not get monkey stats %s", err)
			continue
		} else {
			defer response.Body.Close()
		}

		if response.StatusCode != 200 {
			log.Warningf("non 200 error returned (%d %s) sleeping 10 seconds cause server is probably loaded", response.StatusCode, response.Status)
			time.Sleep(time.Second * 10)
			continue
		}

		results := make(map[string]MonkeyStats)
		err = codec.NewDecoder(response.Body, jsonHandler).Decode(&results)
		if err != nil {
			log.Warningf("could not unmarshal response: %s", err)
			continue
		}

		monKeys := make([]MonkeyStats, 0, monkeysPerRequest)
		for address, monkey := range results {
			monKeys = append(monKeys, monkey)
			monKeys[len(monKeys)-1].PublicAddress = address
			monKeys[len(monKeys)-1].PrivateKey = wallets.lookupWalletSeed(address)
		}

		var totalDelta uint64
		var survivorDelta uint64
		for _, monkey := range monKeys {
			select {
			case <-ctx.Done():
				log.Infof("stopping the %s raid.", raidName)
				break main
			default:
				totalCount++
				totalDelta++
				if matchFilters(monkey) {
					survivorCount++
					survivorDelta++
					select {
					case monkeySendChan <- monkey:
					case <-ctx.Done():
					}
				}
			}
		}
		deltaChan <- gui.Stats{Total: totalDelta, TotalRequests: 1, Found: survivorDelta}
	}
	log.Infof("The %s raided with a total of %d monkeys and %d survivor monKeys!", raidName, totalCount, survivorCount)
}

func outputMonkeyData(ctx context.Context, targetDir string, targetFormat string, monkeyDataChan <-chan MonkeyStats) {

	var convert func(svg io.Reader) (io.Reader, error)
	extension := "." + strings.ToLower(targetFormat)

	switch format := strings.ToUpper(targetFormat); format {
	case "PNG":
		convert = func(svg io.Reader) (io.Reader, error) {
			dataBytes, err := io.ReadAll(svg)
			if err != nil {
				return nil, err
			}
			data, err := legionImage.ConvertSvgToBinary(dataBytes, legionImage.PNGFormat, 250)
			if err != nil {
				return nil, err
			}

			return bytes.NewReader(data), nil
		}
	case "SVG":
		convert = func(svg io.Reader) (io.Reader, error) {
			return svg, nil
		}
	default:
		log.Fatalf("cannot convert to format %s.", targetFormat)
	}

	for {

		var monkey MonkeyStats
		select {
		case monkey = <-monkeyDataChan:
		case <-ctx.Done():
			log.Infof("Stopping some monKey's looting.")
			return
		default:
		}

		monkeySVG, err := bananoutils.GrabMonkey(ctx, bananoutils.Account(monkey.PublicAddress), legionImage.SVGFormat)
		if err != nil {
			log.Warn("lost a monkey %s", err)
			continue
		}

		targetName := monkey.SillyName + "_" + monkey.PublicAddress
		targetJson := path.Join(targetDir, targetName) + ".json"

		var targetImgFile string = path.Join(targetDir, targetName) + extension

		jsonData, err := json.MarshalIndent(monkey, "", "  ")
		if err != nil {
			log.Fatalf("couldn't marshal monKey %s", monkey.SillyName)
		}
		err = ioutil.WriteFile(targetJson, jsonData, 0600)
		if err != nil {
			log.Fatalf("could now write monkey, sad %s: %s", monkey.SillyName, err)
		}
		monkeyConverted, err := convert(monkeySVG)
		if err != nil {
			log.Fatalf("could not convert monkey image, sad monkey %s: %s", monkey.SillyName, err)
		}
		monkeyData, err := io.ReadAll(monkeyConverted)
		if err != nil {
			log.Fatalf("could not write monkey image, sad monkey %s: %s", monkey.SillyName, err)
		}
		err = ioutil.WriteFile(targetImgFile, monkeyData, 0600)
		if err != nil {
			log.Fatalf("could now write monkey, sad monkey %s: %s", monkey.SillyName, err)
		}

	}
}
func previewMonkeys(ctx context.Context, previewChan chan<- gui.MonkeyPreview, monkeyDataChan <-chan MonkeyStats, wg *sync.WaitGroup) {
	defer wg.Done()
	if monkeyDataChan == nil {
		return
	}
	for {
		select {
		case monkey := <-monkeyDataChan:
			// start := time.Now()
			// grab as svg as it is nicer to the server and we can convert it locally
			monkeySVG, err := bananoutils.GrabMonkey(ctx, bananoutils.Account(monkey.PublicAddress), legionImage.SVGFormat)
			if err != nil {
				log.Warnf("could not convert monkey to preview: %s %s", monkey.SillyName, err)
				continue
			}
			data, err := io.ReadAll(monkeySVG)
			if err != nil {
				log.Warnf("could not read data for moneky to preview: %s %s", monkey.SillyName, err)
				continue
			}

			imagePNG, err := legionImage.ConvertSvgToBinary(data, legionImage.PNGFormat, 250)
			if err != nil {
				log.Warnf("could not convert svg monkey image to png data to display: %s %s", monkey.SillyName, err)
				continue
			}

			img, _, err := image.Decode(bytes.NewReader(imagePNG))
			if err != nil {
				log.Warnf("could not decode image data to display for monkey: %s %s", monkey.SillyName, err)
				continue
			}
			select {
			case previewChan <- gui.MonkeyPreview{Image: img, Title: monkey.SillyName}:
			case <-ctx.Done():
			}

		case <-ctx.Done():
			log.Info("stopping the monKey preview")
			return
		}
	}

}

func processMonkeyData(ctx context.Context, targetDir string, targetFormat string, monkeyDataChan <-chan MonkeyStats, monkeyNameChan chan<- string, monkeyDisplayChan chan<- MonkeyStats, wg *sync.WaitGroup) {
	defer wg.Done()
	outputMonkeyChan := make(chan MonkeyStats, 100)

	go func() {
		writeMonkeyWG := new(sync.WaitGroup)
		for i := 0; i < 10; i++ {
			writeMonkeyWG.Add(1)
			go func() {
				outputMonkeyData(ctx, targetDir, targetFormat, outputMonkeyChan)
				writeMonkeyWG.Done()
			}()
		}
		writeMonkeyWG.Wait()
		// All the writers are done, so chanel needs to close
		close(outputMonkeyChan)
	}()

main:
	for {
		select {
		case <-ctx.Done():
			break main
		case monkey, ok := <-monkeyDataChan:
			if !ok {
				break main
			}
			monkeyNameChan <- monkey.SillyName
			outputMonkeyChan <- monkey
			if monkeyDisplayChan != nil {
				wg.Add(1)
				go func() {
					defer wg.Done()
					select {
					case monkeyDisplayChan <- monkey:
					case <-ctx.Done():
						return
					}
				}()
			}

		}
	}
}

func generateFlamingMonkeys(ctx context.Context, monkeysPerRequest uint, monkeyChan chan<- MonkeyStats, deltaChan chan<- gui.Stats, wg *sync.WaitGroup) {
	defer wg.Done()
	wg.Add(1)
	go getMonkeyData(ctx, monkeysPerRequest, monkeyChan, deltaChan, wg)
}

func GrabMonkey(ctx context.Context, publicAddr bananoutils.Account) io.Reader {

	var addressBuilder strings.Builder
	addressBuilder.WriteString(config.MonkeyServer)
	addressBuilder.WriteString("/api/v1/monkey/")
	addressBuilder.WriteString(string(publicAddr))
	addressBuilder.WriteString("?format=svg")
	request, _ := http.NewRequestWithContext(ctx, "GET", addressBuilder.String(), nil)

	response, err := httpClient.Do(request)
	if err != nil {
		log.Fatalf("could not get monkey %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		log.Fatalf("Non 200 error returned (%d %s)", response.StatusCode, response.Status)
	}
	defer response.Body.Close()
	copy := new(bytes.Buffer)
	io.Copy(copy, response.Body)
	return copy
}

func setupHttp() {
	httpTransport = http.DefaultTransport.(*http.Transport).Clone()
	httpTransport.MaxIdleConns = 100
	httpTransport.MaxConnsPerHost = 100
	httpTransport.MaxIdleConnsPerHost = 100

	httpClient = &http.Client{
		Timeout:   120 * time.Second,
		Transport: httpTransport,
	}

	bananoutils.ChangeMonkeyServer(config.MonkeyServer)
}

func setupLog() io.Closer {
	var writer io.Writer

	logFile, err := os.OpenFile("legion-van.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		log.Fatal("could not open log file: %s", err)
		os.Exit(-1)
	}

	stdOutWriter := os.Stderr
	if config.Debug {
		writer = io.MultiWriter(logFile, stdOutWriter)
		log.SetOutput(writer)
	} else {
		log.SetOutput(logFile)
	}
	return logFile
}

func setupGui(ctx context.Context, cleanupMain context.CancelFunc) *gui.MainApp {
	guiApp := gui.NewMainApp(ctx, cleanupMain, "legion-ban", log.StandardLogger())
	if config.Debug {
		simScreen := tcell.NewSimulationScreen("")
		simScreen.SetSize(80, 25)
		simScreen.Init()
		guiApp.SetTerminalScreen(simScreen)

		// Workaround due to an issue with using a predefined screen with cview.
		guiApp.ForceDebugResize()
	}

	return guiApp

}

// setupOutputDir creates target output dir and returns the absolute path of the target directory.
func setupOutputDir() string {
	curdir, err := os.Getwd()
	if err != nil {
		log.Fatal("Can't get current directory.")
	}
	targetDir := path.Join(curdir, "foundMonKeys")
	targetDir, err = filepath.Abs(targetDir)
	if err != nil {
		log.Fatalf("could not resolve directory path: %s", err)
	}
	err = os.MkdirAll(targetDir, 0700)
	if err != nil {
		log.Fatalf("could not create directory %s", err)
	}
	stat, err := os.Stat(targetDir)
	if err != nil {
		log.Fatalf("could not verify permissions for %s: %s", targetDir, err)
	}
	if stat.Mode().Perm() != 0700 {
		log.Fatalf("will not run because %s is not set to 0700 permissions, you don't want anyone reading your keys do you?", targetDir)
	}
	return targetDir
}

func main() {
	//runtime.SetBlockProfileRate(1)
	parseFlags()
	logFile := setupLog()
	defer logFile.Close()
	setupHttp()

	log.Infof("Using %d cpus", runtime.GOMAXPROCS(-1))
	legionImage.Init()
	defer legionImage.Destroy()

	targetDir := setupOutputDir()

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, config.HowLongToRun)
	gui := setupGui(ctx, cancel)

	finishedChan := make(chan struct{})
	defer close(finishedChan)

	go func() {
		defer gui.MainHasShutdown()
		monkeyDataChan := make(chan MonkeyStats, 1000)
		wgMonkeyDataChan := new(sync.WaitGroup)

		monkeyNameChan := make(chan string, 100)
		monkeyDisplayChan := make(chan MonkeyStats, 10)
		wgProccessDisplayAndNameChan := new(sync.WaitGroup)

		// required to finish before closing out the main gui
		PNGPreviewChanWG := new(sync.WaitGroup)

		for i := uint(0); i < config.MaxRequests; i++ {
			wgMonkeyDataChan.Add(1)
			go generateFlamingMonkeys(ctx, config.BatchSize, monkeyDataChan, gui.TotalDeltaChan(), wgMonkeyDataChan)
			wgProccessDisplayAndNameChan.Add(1)
			go processMonkeyData(ctx, targetDir, string(config.Format), monkeyDataChan, monkeyNameChan, monkeyDisplayChan, wgProccessDisplayAndNameChan)
		}
		PNGPreviewChanWG.Add(4)
		go previewMonkeys(ctx, gui.PNGPreviewChan(), monkeyDisplayChan, PNGPreviewChanWG)
		go previewMonkeys(ctx, gui.PNGPreviewChan(), monkeyDisplayChan, PNGPreviewChanWG)
		go previewMonkeys(ctx, gui.PNGPreviewChan(), monkeyDisplayChan, PNGPreviewChanWG)
		go previewMonkeys(ctx, gui.PNGPreviewChan(), monkeyDisplayChan, PNGPreviewChanWG)
		var monkeyHeadCount uint64
	main:
		for {
			select {
			case <-ctx.Done():
				log.Info("Failing Monkeygedon")
				break main
			case monkeyName, ok := <-monkeyNameChan:
				if !ok {
					break main
				}
				monkeyHeadCount++
				log.Infof("Say hi to %s", monkeyName)
			}
		}
		log.Infof("Total monKeys confirmed alive %d", monkeyHeadCount)
		log.Info("Waiting for pending writes and network queries to close...there may be more survivors on the lifeboat")
		wgMonkeyDataChan.Wait()
		close(monkeyDataChan)
		wgProccessDisplayAndNameChan.Wait()
		close(monkeyNameChan)
		PNGPreviewChanWG.Wait()
	}()

	go http.ListenAndServe(":8888", nil)
	gui.Run()
}
