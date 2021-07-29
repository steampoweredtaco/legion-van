package main

import (
	"flag"
	"net/http"
	"strings"
	"time"

	"github.com/bbedward/nano/address"
	"github.com/golang/glog"
	"github.com/steampoweredtaco/legion-van/bananoutils"
	"github.com/steampoweredtaco/legion-van/image"
)

var config struct {
	interval     *time.Duration
	max_requests *uint
}

func setupFlags() {
	config.interval = flag.Duration("interval", time.Minute, "Frequency to update represenitative data.")
	config.max_requests = flag.Uint("max_requests", 10, "Maxiumum outstanding requests to spyglass.")
}

func parseFlags() {
	flag.Parse()
}

func GenerateAddress() string {
	pub, _ := address.GenerateKey()
	return strings.Replace(string(address.PubKeyToAddress(pub)), "nano_", "ban_", -1)
}

func grabMonkey() {

	privateHex, publicAddr, err := bananoutils.GeneratePrivateKeyAndFirstPublicAddress()
	if err != nil {
		panic(err)
	}

	glog.Infof("Created key %s %s", privateHex, publicAddr)
	var addressBuilder strings.Builder
	addressBuilder.WriteString("https://monkey.banano.cc/api/v1/monkey/")
	addressBuilder.WriteString(string(publicAddr))
	response, err := http.Get(addressBuilder.String())
	if err != nil {
		glog.Fatalf("could not get monkey %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		glog.Fatalf("Non 200 error returned (%d %s)", response.StatusCode, response.Status)
	}
	image.DisplayImage(response.Body)
}

func main() {
	setupFlags()
	parseFlags()
	grabMonkey()

}
