package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/Pallinder/go-randomdata"
	"github.com/bbedward/nano/address"
	"github.com/eiannone/keyboard"
	"github.com/golang/glog"
	"github.com/steampoweredtaco/legion-van/bananoutils"
	"github.com/steampoweredtaco/legion-van/image"
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
}

type targetFormat string

func setupFlags() {
	config.format = "svg"

	config.interval = flag.Duration("interval", time.Minute, "Frequency to update represenitative data.")
	config.max_requests = flag.Uint("max_requests", 4, "Maxiumum outstanding requests to spyglass.")
	config.disablePreview = flag.Bool("disable_preview", false, "Disable preview of found monkey in terminal, when enabled press any key to continue with next preview.")
	config.batch_size = flag.Uint("batch_size", 2500, "Number of monkeys to test per API request, higher or lower may affect performance")
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
		glog.Fatalf("could not marshal addresses for request %s", err)
	}
	return bytes.NewBuffer(data)
}

func getMonkeyData(ctx context.Context, monkeysPerRequest uint, monkeySendChan chan<- MonkeyStats, wg *sync.WaitGroup) {
	defer wg.Done()
	defer close(monkeySendChan)
	getStatsURL := "https://monkey.banano.cc/api/v1/monkey/dtl"
	var totalCount uint64
	var survivorCount uint64
main:
	for {

		// Exit early there are web errors and then the app shutsdown
		select {
		case <-ctx.Done():
			glog.Infof("Stopping a monKey raid.")
			return
		default:
		}
		wallets := generateManyWallets(monkeysPerRequest)
		jsonBody := make(map[string][]string)
		jsonBody["addresses"] = wallets.getAccounts()
		jsonReader := wallets.encodeAccountsAsJSON()

		request, err := http.NewRequest("POST", getStatsURL, jsonReader)
		if err != nil {
			glog.Errorf("Could not get monkey stats %s", err)
			continue
		}
		request.Header.Set("Content-Type", "application/json")
		response, err := httpClient.Do(request)
		if err != nil {
			glog.Errorf("Could not get monkey stats %s", err)
			continue
		} else {
			defer response.Body.Close()
		}

		if response.StatusCode != 200 {
			glog.Warningf("Non 200 error returned (%d %s) sleeping 10 seconds cause server is probably loaded", response.StatusCode, response.Status)
			time.Sleep(time.Second * 10)
			continue
		}

		results := make(map[string]MonkeyStats)
		err = codec.NewDecoder(response.Body, jsonHandler).Decode(&results)
		if err != nil {
			glog.Warningf("could not unmarshal response: %s sleeping for a bit to not flood", err)
			time.Sleep(time.Second * 10)
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
				glog.Infof("Stopping a monKey raid.")
				break main
			default:
				totalCount++
				if strings.HasPrefix(monkey.Misc, "flamethrower") && strings.HasPrefix(monkey.Mouth, "joint") && strings.HasPrefix(monkey.Glasses, "mono") {
					survivorCount++
					monkeySendChan <- monkey
				}
			}
		}
	}
	glog.Infof("This banano republic raided with a total of %d monkeys and %d survivor monKeys!", totalCount, survivorCount)
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
			data, err := image.ConvertSvgToBinary(dataBytes, image.ImageFormat(format), 1000)
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
		glog.Fatalf("cannot convert to format %s.", targetFormat)
	}

	for monkey := range monkeyDataChan {
		select {
		case <-ctx.Done():
			glog.Infof("Stopping some monKey's looting.")
			return
		default:
		}

		monkeySVG := grabMonkey(bananoutils.Account(monkey.PublicAddress))
		targetName := monkey.SillyName + "_" + monkey.PublicAddress
		targetJson := path.Join(targetDir, targetName) + ".json"

		var targetImgFile string = path.Join(targetDir, targetName) + extension

		jsonData, err := json.MarshalIndent(monkey, "", "  ")
		if err != nil {
			glog.Fatalf("couldn't marshal monKey %s", monkey.SillyName)
		}
		err = ioutil.WriteFile(targetJson, jsonData, 0600)
		if err != nil {
			glog.Fatalf("could now write monkey, sad %s: %s", monkey.SillyName, err)
		}
		monkeyConverted, err := convert(monkeySVG)
		if err != nil {
			glog.Fatalf("could not convert monkey image, sad monkey %s: %s", monkey.SillyName, err)
		}
		monkeyData, err := io.ReadAll(monkeyConverted)
		if err != nil {
			glog.Fatalf("could not write monkey image, sad monkey %s: %s", monkey.SillyName, err)
		}
		err = ioutil.WriteFile(targetImgFile, monkeyData, 0600)
		if err != nil {
			glog.Fatalf("could now write monkey, sad monkey %s: %s", monkey.SillyName, err)
		}

	}
}
func displayMonkey(ctx context.Context, monkeyDataChan <-chan MonkeyStats, wg *sync.WaitGroup) {
	defer wg.Done()
	if monkeyDataChan == nil {
		return
	}
	for monkey := range monkeyDataChan {
		select {
		case <-ctx.Done():
			glog.Infof("Stopping the monKey to the moon.")
			return
		default:
		}
		monkeyPNG := grabMonkey(bananoutils.Account(monkey.PublicAddress))
		image.DisplayImage(monkeyPNG, monkey.PublicAddress+"\n"+monkey.PrivateKey)
	}

}

func processMonkeyData(ctx context.Context, shouldDisplayMonkey bool, targetDir string, targetFormat string, monkeyDataChan <-chan MonkeyStats, monkeyNameChan chan<- string, wg *sync.WaitGroup) {
	defer wg.Done()
	writeMonkeyChan := make(chan MonkeyStats, 100)
	defer close(writeMonkeyChan)

	var displayMonkeyChan chan MonkeyStats = nil
	if shouldDisplayMonkey {
		displayMonkeyChan = make(chan MonkeyStats, 100)
		defer close(displayMonkeyChan)
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go writeMonkeyData(ctx, targetDir, targetFormat, writeMonkeyChan, wg)
	}

	if displayMonkeyChan != nil {
		wg.Add(1)
		go displayMonkey(ctx, displayMonkeyChan, wg)
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
			if shouldDisplayMonkey {
				if displayMonkeyChan != nil {
					displayMonkeyChan <- monkey
				}
			}

		}
	}
}

func generateFlamingMonkeys(ctx context.Context, monkeysPerRequest uint, monkeyChan chan<- MonkeyStats, wg *sync.WaitGroup) {
	defer wg.Done()
	wg.Add(1)
	go getMonkeyData(ctx, monkeysPerRequest, monkeyChan, wg)
}

func grabMonkey(publicAddr bananoutils.Account) io.Reader {

	var addressBuilder strings.Builder
	addressBuilder.WriteString("https://monkey.banano.cc/api/v1/monkey/")
	addressBuilder.WriteString(string(publicAddr))
	addressBuilder.WriteString("?format=svg")
	request, _ := http.NewRequest("GET", addressBuilder.String(), nil)

	response, err := httpClient.Do(request)
	if err != nil {
		glog.Fatalf("could not get monkey %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		glog.Fatalf("Non 200 error returned (%d %s)", response.StatusCode, response.Status)
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
}

func main() {

	glog.Infof("Using %d cpus", runtime.GOMAXPROCS(-1))
	setupHttp()
	setupFlags()
	parseFlags()
	image.Init()
	defer image.Destroy()
	curdir, err := os.Getwd()
	if err != nil {
		glog.Fatal("Can't get current directory.")
	}
	targetDir := path.Join(curdir, "foundMonKeys")
	err = os.MkdirAll(targetDir, 0700)
	if err != nil {
		glog.Fatalf("could not create directory %s", err)
	}
	stat, err := os.Stat(targetDir)
	if err != nil {
		glog.Fatalf("could not verify permissions for %s: %s", targetDir, err)
	}
	if stat.Mode().Perm() != 0700 {
		glog.Fatalf("will not run because %s is not set to 0700 permissions, you don't want anyone reading your keys do you?", targetDir)
	}

	ctx := context.Background()
	//cancelCtx := context.WithCancel(ctx)
	deadlineCtx, cancel := context.WithTimeout(ctx, *config.interval)
	finishedChan := make(chan struct{})
	defer close(finishedChan)
	go func(done chan<- struct{}) {
		monkeyNameChan := make(chan string, 100)
		defer close(monkeyNameChan)
		monkeyDataChan := make(chan MonkeyStats, 1000)
		defer close(monkeyDataChan)

		wg := new(sync.WaitGroup)
		for i := uint(0); i < *config.max_requests; i++ {
			wg.Add(1)
			go generateFlamingMonkeys(deadlineCtx, *config.batch_size, monkeyDataChan, wg)
			go processMonkeyData(ctx, !*config.disablePreview, targetDir, string(config.format), monkeyDataChan, monkeyNameChan, wg)

		}
		var monkeyHeadCount uint64
	main:
		for {
			select {
			case <-deadlineCtx.Done():
				glog.Info("Failing Monkeygedon")
				break main
			case monkeyName, ok := <-monkeyNameChan:
				if !ok {
					break main
				}
				monkeyHeadCount++
				glog.Infof("Say hi to %s", monkeyName)
			}
		}
		glog.Infof("Total monKeys confirmed alive %d", monkeyHeadCount)
		wg.Wait()
		done <- struct{}{}
	}(finishedChan)

	keyEvent, _ := keyboard.GetKeys(1)
	defer keyboard.Close()
	if err != nil {
		glog.Fatal("cannot read keyboard input: ", err)
	}

	glog.Info("Push ESC or Q/q to quit.")
main:
	for {
		select {

		case key := <-keyEvent:
			if key.Key != keyboard.KeyEsc && key.Rune != 'q' && key.Rune != 'Q' {
				continue
			}
			cancel()
			<-finishedChan
			break main
		case <-finishedChan:
			break main
		}
	}
}
