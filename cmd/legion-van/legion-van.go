package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
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
	log "github.com/sirupsen/logrus"
	"github.com/steampoweredtaco/legion-van/bananoutils"
	"github.com/steampoweredtaco/legion-van/gui"
	legionImage "github.com/steampoweredtaco/legion-van/image"
	"github.com/ugorji/go/codec"
)

var (
	jsonHandler = new(codec.JsonHandle)
)

var (
	httpTransport *http.Transport
	httpClient    *http.Client
)

func (*targetFormat) String() string {
	return "svg"
}

func (targetFmt *targetFormat) Set(value string) error {
	switch format := strings.ToLower(value); format {
	case "svg":
		fallthrough
	case "png":
		*targetFmt = targetFormat(format)
	default:
		return fmt.Errorf("only svg or png supported")
	}
	return nil
}

var config struct {
	interval       *time.Duration
	max_requests   *uint
	disablePreview *bool
	format         targetFormat
	batch_size     *uint
	debug          *bool
	monkeyServer   *string
}

type targetFormat string

func setupFlags() {
	config.format = "svg"

	config.debug = flag.Bool("debug", false, "changes logging and makes terminal virtual for debugging issues.")
	config.interval = flag.Duration("interval", time.Minute, "Frequency to update represenitative data.")
	config.max_requests = flag.Uint("max_requests", 4, "Maxiumum outstanding requests to spyglass.")
	config.disablePreview = flag.Bool("disable_preview", false, "Disable preview of found monkey in terminal, when enabled press any key to continue with next preview.")
	config.batch_size = flag.Uint("batch_size", 2500, "Number of monkeys to test per API request, higher or lower may affect performance")
	config.monkeyServer = flag.String("monkey_api", "https://monkey.banano.cc", "To change the backend monkey server, defaults to official one.")
	flag.Var(&config.format, "format", "set the target image format for saving monkey found in options are svg or png. svg is faster")

}
func parseFlags() {
	flag.Parse()
}

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

func getMonkeyData(ctx context.Context, monkeysPerRequest uint, monkeySendChan chan<- MonkeyStats, wg *sync.WaitGroup) {
	defer wg.Done()
	getStatsURL := fmt.Sprintf("%s/api/v1/monkey/dtl", *config.monkeyServer)
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
			return
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

		for _, monkey := range monKeys {
			select {
			case <-ctx.Done():
				log.Infof("stopping the %s raid.", raidName)
				break main
			default:
				totalCount++
				if strings.HasPrefix(monkey.Misc, "flamethrower") {
					survivorCount++
					monkeySendChan <- monkey
				}
			}
		}
	}
	log.Infof("The %s raided with a total of %d monkeys and %d survivor monKeys!", raidName, totalCount, survivorCount)
}

func writeMonkeyData(ctx context.Context, targetDir string, targetFormat string, monkeyDataChan <-chan MonkeyStats, wg *sync.WaitGroup) {
	defer wg.Done()

	var convert func(svg io.Reader) (io.Reader, error)
	extension := "." + strings.ToLower(targetFormat)

	switch format := strings.ToUpper(targetFormat); format {
	case "PNG":
		convert = func(svg io.Reader) (io.Reader, error) {
			dataBytes, err := io.ReadAll(svg)
			if err != nil {
				return nil, err
			}
			data, err := legionImage.ConvertSvgToBinary(dataBytes, legionImage.PNGFormat, 1000)
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

	for monkey := range monkeyDataChan {
		select {
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
			previewChan <- gui.MonkeyPreview{Image: img, Title: monkey.SillyName}
		case <-ctx.Done():
			log.Debug("stopping the monKey preview")
			return
		}
	}

}

func processMonkeyData(ctx context.Context, targetDir string, targetFormat string, monkeyDataChan <-chan MonkeyStats, monkeyNameChan chan<- string, monkeyDisplayChan chan<- MonkeyStats, wg *sync.WaitGroup) {
	defer wg.Done()
	writeMonkeyChan := make(chan MonkeyStats, 100)
	defer close(writeMonkeyChan)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go writeMonkeyData(ctx, targetDir, targetFormat, writeMonkeyChan, wg)
	}

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
			writeMonkeyChan <- monkey
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

func generateFlamingMonkeys(ctx context.Context, monkeysPerRequest uint, monkeyChan chan<- MonkeyStats, wg *sync.WaitGroup) {
	defer wg.Done()
	wg.Add(1)
	go getMonkeyData(ctx, monkeysPerRequest, monkeyChan, wg)
}

func GrabMonkey(ctx context.Context, publicAddr bananoutils.Account) io.Reader {

	var addressBuilder strings.Builder
	addressBuilder.WriteString(*config.monkeyServer)
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

	bananoutils.ChangeMonkeyServer(*config.monkeyServer)
}

func setupLog() io.Closer {
	var writer io.Writer

	logFile, err := os.OpenFile("legion-van.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		log.Fatal("could not open log file: %s", err)
		os.Exit(-1)
	}

	stdOutWriter := os.Stderr
	if *config.debug {
		writer = io.MultiWriter(logFile, stdOutWriter)
		log.SetReportCaller(true)
		log.SetOutput(writer)
	} else {
		log.SetOutput(logFile)
	}
	return logFile
}

func setupGui(ctx context.Context, cleanupMain context.CancelFunc) *gui.MainApp {
	guiApp := gui.NewMainApp(ctx, cleanupMain, "legion-ban", log.StandardLogger())
	if *config.debug {
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
	runtime.SetBlockProfileRate(1)
	setupFlags()
	parseFlags()
	logFile := setupLog()
	defer logFile.Close()
	setupHttp()

	log.Infof("Using %d cpus", runtime.GOMAXPROCS(-1))
	legionImage.Init()
	defer legionImage.Destroy()

	targetDir := setupOutputDir()

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, *config.interval)
	gui := setupGui(ctx, cancel)

	finishedChan := make(chan struct{})
	defer close(finishedChan)

	go func() {
		monkeyNameChan := make(chan string, 100)
		monkeyDataChan := make(chan MonkeyStats, 1000)
		monkeyDisplayChan := make(chan MonkeyStats, 10)
		wg := new(sync.WaitGroup)
		wgProcessing := new(sync.WaitGroup)
		for i := uint(0); i < *config.max_requests; i++ {
			wg.Add(1)
			go generateFlamingMonkeys(ctx, *config.batch_size, monkeyDataChan, wg)
			wgProcessing.Add(1)
			go processMonkeyData(ctx, targetDir, string(config.format), monkeyDataChan, monkeyNameChan, monkeyDisplayChan, wgProcessing)
		}
		wgProcessing.Add(1)
		go previewMonkeys(ctx, gui.PNGPreviewChan(), monkeyDisplayChan, wgProcessing)
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
		wg.Wait()
		close(monkeyDataChan)
		wgProcessing.Wait()
		close(monkeyNameChan)
		gui.MainHasShutdown()
	}()

	go http.ListenAndServe(":8888", nil)

	gui.Run()
}
