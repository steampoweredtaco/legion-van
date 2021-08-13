package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/Pallinder/go-randomdata"
	log "github.com/sirupsen/logrus"
	"github.com/steampoweredtaco/legion-van/bananoutils"
	legionImage "github.com/steampoweredtaco/legion-van/image"
	"github.com/ugorji/go/codec"
)

func GenerateAndFilterMonkees(ctx context.Context, monkeysPerRequest uint, monkeyChan chan<- MonkeyStats, deltaChan chan<- Stats, filter CmdLineFilter, wg *sync.WaitGroup) {
	defer wg.Done()
	wg.Add(1)
	go getMonkeyData(ctx, monkeysPerRequest, monkeyChan, deltaChan, filter, wg)
}

func getMonkeyData(ctx context.Context, monkeysPerRequest uint, monkeySendChan chan<- MonkeyStats, deltaChan chan<- Stats, filter CmdLineFilter, wg *sync.WaitGroup) {
	defer wg.Done()
	getStatsURL := bananoutils.GetMonkeyDescriptionURI()
	var totalCount uint64
	var survivorCount uint64
	raidName := strings.Title(randomdata.Adjective() + " " + randomdata.Noun())

	log.Infof("Raiding with %s clan", raidName)
main:
	for {

		// Exit early there are web errors and then the app shutsdown
		select {
		case <-ctx.Done():
			log.Debugf("stopping the %s raid.", raidName)
			break main
		default:
		}
		wallets := generateManyWallets(monkeysPerRequest)
		jsonBody := make(map[string][]string)
		jsonBody["addresses"] = wallets.getAccounts()
		jsonReader := wallets.encodeAccountsAsJSON()

		request, err := http.NewRequestWithContext(ctx, "POST", getStatsURL, jsonReader)
		if err != nil {
			select {
			case <-ctx.Done():
			default:
				log.Errorf("could not get monkey stats %s", err)
			}
			continue
		}
		request.Header.Set("Content-Type", "application/json")
		response, err := httpClient.Do(request)
		if err != nil {
			select {
			case <-ctx.Done():
			default:
				log.Errorf("could not get monkey stats %s", err)
			}
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
			// These are gonna be a coding error or caused by the context deadline so only have tese for debuging.
			log.Debugf("could not unmarshal response: %s %T", err, err)
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
				log.Debugf("stopping the %s raid.", raidName)
				break main
			default:
				totalCount++
				totalDelta++
				if matchFilters(monkey, filter) {
					survivorCount++
					survivorDelta++
					select {
					case monkeySendChan <- monkey:
					case <-ctx.Done():
					}
				}
			}
		}
		deltaChan <- Stats{Total: totalDelta, TotalRequests: 1, Found: survivorDelta}
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
			log.Debugf("Stopping some monKey's looting.")
			return
		}

		monkeySVG, err := bananoutils.GrabMonkey(ctx, bananoutils.Account(monkey.PublicAddress), legionImage.SVGFormat)
		if err != nil {
			log.Warnf("lost a monkey %s", err)
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
