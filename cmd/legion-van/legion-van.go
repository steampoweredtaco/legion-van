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

var config struct {
	interval       *time.Duration
	max_requests   *uint
	disablePreview *bool
}

func setupFlags() {
	config.interval = flag.Duration("interval", time.Minute, "Frequency to update represenitative data.")
	config.max_requests = flag.Uint("max_requests", 10, "Maxiumum outstanding requests to spyglass.")
	config.disablePreview = flag.Bool("disable_preview", false, "Disable preview of found monkey in terminal, when enabled press any key to continue with next preview.")
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
		glog.Fatalf("could not marshal addresses for request %w", err)
	}
	return bytes.NewBuffer(data)
}

func getMonkeyData(ctx context.Context, monkeysPerRequest uint, monkeySendChan chan<- MonkeyStats) {
	defer close(monkeySendChan)

	getStatsURL := "https://monkey.banano.cc/api/v1/monkey/dtl"
	for {

		wallets := generateManyWallets(monkeysPerRequest)
		jsonBody := make(map[string][]string)
		jsonBody["addresses"] = wallets.getAccounts()
		jsonReader := wallets.encodeAccountsAsJSON()

		response, err := http.Post(getStatsURL, "application/json", jsonReader)
		if err != nil {
			glog.Errorf("Could not get monkey stats %w", err)
		}

		defer response.Body.Close()
		if response.StatusCode != 200 {
			glog.Fatalf("Non 200 error returned (%d %s)", response.StatusCode, response.Status)
		}

		results := make(map[string]MonkeyStats)
		err = codec.NewDecoder(response.Body, jsonHandler).Decode(&results)
		if err != nil {
			glog.Fatalf("could not unmarshal response: %s", err)
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
				glog.Info("Stopping the monKey horde %s", ctx.Err())
				return
			default:
				monkeySendChan <- monkey
			}
		}
	}
}

func writeMonkeyData(ctx context.Context, targetDir string, monkeyDataChan <-chan MonkeyStats, monkeyNameChan chan<- string, wg *sync.WaitGroup) {
	defer wg.Done()
	for monkey := range monkeyDataChan {
		select {
		case <-ctx.Done():
			glog.Info("Stopping the monKey looting %s", ctx.Err())
			return
		default:
		}

		if monkey.Misc != "none" && strings.HasPrefix(monkey.Misc, "flamethrower") {
			monkeyPNG := grabMonkey(bananoutils.Account(monkey.PublicAddress))
			var monkeyPNGTee bytes.Buffer

			monkeyPNG = io.TeeReader(monkeyPNG, &monkeyPNGTee)
			if !*config.disablePreview {
				image.DisplayImage(monkeyPNG, monkey.PublicAddress+"\n"+monkey.PrivateKey)
			} else {
				// Have to read for the tee to work
				_, err := io.ReadAll(monkeyPNG)
				if err != nil {
					glog.Fatalf("could not read monkey, so sad: %s", err)
				}
			}
			monkeyName := randomdata.SillyName()
			targetName := monkeyName + "_" + monkey.PublicAddress
			targetJson := path.Join(targetDir, targetName) + ".json"
			targetPNG := path.Join(targetDir, targetName) + ".png"
			monkeyNameChan <- monkeyName

			jsonData, err := json.MarshalIndent(monkey, "", "  ")
			if err != nil {
				glog.Fatal("couldn't marshal monKey!")
			}
			err = ioutil.WriteFile(targetJson, jsonData, 0600)
			if err != nil {
				glog.Fatal("could now write monkey, sad monkey: %s", err)
			}
			monkeyData, err := io.ReadAll(&monkeyPNGTee)
			if err != nil {
				glog.Fatal("could not write monkey image, sad monkey: %s", err)
			}
			err = ioutil.WriteFile(targetPNG, monkeyData, 0600)
			if err != nil {
				glog.Fatal("could now write monkey, sad monkey: %s", err)
			}
		}
	}
}

func processMonkeyData(ctx context.Context, targetDir string, monkeyDataChan <-chan MonkeyStats, monkeyNameChan chan<- string) {
	defer close(monkeyNameChan)
	wg := new(sync.WaitGroup)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go writeMonkeyData(ctx, targetDir, monkeyDataChan, monkeyNameChan, wg)
	}
	wg.Wait()

}
func generateFlamingMonkeys(ctx context.Context, monkeysPerRequest uint, targetDir string) <-chan string {
	monkeyChan := make(chan MonkeyStats, monkeysPerRequest)
	monkeyNameChan := make(chan string, 10)

	go getMonkeyData(ctx, monkeysPerRequest, monkeyChan)
	go processMonkeyData(ctx, targetDir, monkeyChan, monkeyNameChan)
	return monkeyNameChan
}

func grabMonkey(publicAddr bananoutils.Account) io.Reader {

	var addressBuilder strings.Builder
	addressBuilder.WriteString("https://monkey.banano.cc/api/v1/monkey/")
	addressBuilder.WriteString(string(publicAddr))
	addressBuilder.WriteString("?format=png&size=1000")
	response, err := http.Get(addressBuilder.String())
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

func main() {
	setupFlags()
	parseFlags()
	curdir, err := os.Getwd()
	if err != nil {
		glog.Fatal("Can't get current directory.")
	}
	targetDir := path.Join(curdir, "foundMonKeys")
	err = os.MkdirAll(targetDir, 0700)
	if err != nil {
		glog.Fatal("could not create directory %s", err)
	}
	stat, err := os.Stat(targetDir)
	if err != nil {
		glog.Fatal("could not verify permissions for %s: %s", targetDir, err)
	}
	if stat.Mode().Perm() != 0700 {
		glog.Fatal("will not run because %s is not set to 0700 permissions, you don't want anyone reading your keys do you?")
	}

	ctx := context.Background()
	//cancelCtx := context.WithCancel(ctx)
	deadlineCtx, cancel := context.WithTimeout(ctx, time.Minute)
	finishedChan := make(chan struct{})
	defer close(finishedChan)
	go func(done chan<- struct{}) {
		monkeyNameChan := generateFlamingMonkeys(deadlineCtx, 10000, targetDir)
		var monkeyHeadCount uint64
	main:
		for {
			select {
			case <-deadlineCtx.Done():
				glog.Info("Ended Monkeygedon")
				glog.Infof("Found a total of %d monKeys", monkeyHeadCount)
				break main
			case monkeyName, more := <-monkeyNameChan:
				if !more {
					break main
				}
				monkeyHeadCount++
				glog.Infof("Say hi to %s", monkeyName)
			}
		}
		done <- struct{}{}
	}(finishedChan)

	keyEvent, _ := keyboard.GetKeys(1)
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
			break main
		case <-finishedChan:
			cancel()
			break main
		}
	}
}
