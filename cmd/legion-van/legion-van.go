package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/Pallinder/go-randomdata"
	"github.com/bbedward/nano/address"
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

func generateFlamingMonkeys(targetDir string) {

	getStatsURL := "https://monkey.banano.cc/api/v1/monkey/dtl"
	var addressesToWalletKey = make(map[bananoutils.Account]string, 1000)
	addresses := make([]string, 0, 1000)
	for i := 0; i < 10000; i++ {
		privateKey, publicAddr, err := bananoutils.GeneratePrivateKeyAndFirstPublicAddress()
		if err != nil {
			panic(err)
		}
		addressesToWalletKey[publicAddr] = privateKey
		addresses = append(addresses, string(publicAddr))
	}

	jsonBody := make(map[string][]string)
	jsonBody["addresses"] = addresses
	jsonData := make([]byte, 1000)
	err := codec.NewEncoderBytes(&jsonData, jsonHandler).Encode(jsonBody)
	if err != nil {
		glog.Errorf("could not marshal addresses for request %w", err)
	}
	response, err := http.Post(getStatsURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		glog.Errorf("Could not get monkey stats %w", err)
	}

	defer response.Body.Close()
	if response.StatusCode != 200 {
		glog.Fatalf("Non 200 error returned (%d %s)", response.StatusCode, response.Status)
	}

	results := make(map[string]MonkeyStats)
	glog.Infof("%+v", &results)
	if results == nil {
		glog.Info("yes nill?!")
	}
	err = codec.NewDecoder(response.Body, jsonHandler).Decode(&results)
	if err != nil {
		glog.Fatalf("could not unmarshal response: %s", err)
	}

	monKeys := make([]MonkeyStats, 0, 1000)
	for address, monkey := range results {
		monKeys = append(monKeys, monkey)
		monKeys[len(monKeys)-1].PublicAddress = address
		monKeys[len(monKeys)-1].PrivateKey = addressesToWalletKey[bananoutils.Account(address)]
	}

	monkeyCount := 0
	for _, monkey := range monKeys {
		if monkey.Misc != "none" && strings.HasPrefix(monkey.Misc, "flamethrower") {
			monkeyCount++
			monkeyPNG := grabMonkey(bananoutils.Account(monkey.PublicAddress))
			var monkeyPNGTee bytes.Buffer

			monkeyPNG = io.TeeReader(monkeyPNG, &monkeyPNGTee)
			if !*config.disablePreview {
				image.DisplayImage(monkeyPNG, monkey.PublicAddress+"\n"+addressesToWalletKey[bananoutils.Account(monkey.PublicAddress)])
			} else {
				// Have to read for the tee to work
				_, err = io.ReadAll(monkeyPNG)
				if err != nil {
					glog.Fatalf("could not read monkey, so sad: %s", err)
				}
			}
			monkeyName := randomdata.SillyName()
			targetName := monkeyName + "_" + monkey.PublicAddress
			targetJson := path.Join(targetDir, targetName) + ".json"
			targetPNG := path.Join(targetDir, targetName) + ".png"
			glog.Infof("Say hello to %s", monkeyName)

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
	glog.Infof("Found %d monKeys!", monkeyCount)
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
	//grabMonkey()
	generateFlamingMonkeys(targetDir)

}
