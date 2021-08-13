package main

import (
	"context"
	"fmt"
	"io"
	mrand "math/rand"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	_ "net/http/pprof"

	"github.com/gdamore/tcell/v2"
	"github.com/jessevdk/go-flags"
	log "github.com/sirupsen/logrus"
	"github.com/steampoweredtaco/legion-van/bananoutils"
	"github.com/steampoweredtaco/legion-van/engine"
	"github.com/steampoweredtaco/legion-van/gui"
	legionImage "github.com/steampoweredtaco/legion-van/image"
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
	VerboseLog     bool          `long:"verbose" description:"Changes logging to print debug."`
	MonkeyServer   string        `long:"monkey_api" description:"To change the backend monkey server, defaults to the official one." default:"https://monkey.banano.cc"`
	NumOfThreads   int           `long:"threads" description:"Changes number of threads to use, defaults to 2, with a decent machine this is probably all you need. Set to -1 for all hardware cpu threads available." default:"2"`
}

var filter engine.CmdLineFilter

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

func setupHttp() {
	httpTransport := http.DefaultTransport.(*http.Transport).Clone()
	httpTransport.MaxIdleConns = 100
	httpTransport.MaxConnsPerHost = 100
	httpTransport.MaxIdleConnsPerHost = 100

	httpClient := &http.Client{
		Timeout:   120 * time.Second,
		Transport: httpTransport,
	}

	engine.SetupHTTP(httpTransport, httpClient)
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
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetOutput(logFile)
	}
	if config.VerboseLog {
		log.SetLevel(log.DebugLevel)
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

	log.Infof("Using %d cpus", runtime.GOMAXPROCS(config.NumOfThreads))
	legionImage.Init()
	defer legionImage.Destroy()

	targetDir := setupOutputDir()

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, config.HowLongToRun)
	guiInstance := setupGui(ctx, cancel)

	finishedChan := make(chan struct{})
	defer close(finishedChan)

	go func() {
		defer guiInstance.MainHasShutdown()
		monkeyDataChan := make(chan engine.MonkeyStats, 1000)
		wgMonkeyDataChan := new(sync.WaitGroup)

		monkeyNameChan := make(chan string, 100)
		monkeyDisplayChan := make(chan engine.MonkeyStats, 10)
		wgProccessDisplayAndNameChan := new(sync.WaitGroup)

		// required to finish before closing out the main gui
		PNGPreviewChanWG := new(sync.WaitGroup)

		for i := uint(0); i < config.MaxRequests; i++ {
			wgMonkeyDataChan.Add(1)
			go engine.GenerateAndFilterMonkees(ctx, config.BatchSize, monkeyDataChan, guiInstance.TotalDeltaChan(), filter, wgMonkeyDataChan)
			wgProccessDisplayAndNameChan.Add(1)
			go engine.ProcessMonkeyData(ctx, targetDir, string(config.Format), monkeyDataChan, monkeyNameChan, monkeyDisplayChan, wgProccessDisplayAndNameChan)
		}
		PNGPreviewChanWG.Add(4)
		go gui.PreviewMonkeys(ctx, guiInstance.PNGPreviewChan(), monkeyDisplayChan, PNGPreviewChanWG)
		go gui.PreviewMonkeys(ctx, guiInstance.PNGPreviewChan(), monkeyDisplayChan, PNGPreviewChanWG)
		go gui.PreviewMonkeys(ctx, guiInstance.PNGPreviewChan(), monkeyDisplayChan, PNGPreviewChanWG)
		go gui.PreviewMonkeys(ctx, guiInstance.PNGPreviewChan(), monkeyDisplayChan, PNGPreviewChanWG)
		var monkeyHeadCount uint64
	main:
		for {
			select {
			case <-ctx.Done():
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
		// Logging to gui can be out of order but these lines should be serial
		log.Infof("Raiding time up for looting them vain monKeys\nFind your monKeys and their wallets at %s\nESC to quit.", targetDir)
		mrand.Seed(time.Now().UnixNano())
		if mrand.Intn(100) == 42 {
			log.Info("wen poem?")
		}
	}()

	go http.ListenAndServe(":8888", nil)
	guiInstance.Run()
}
