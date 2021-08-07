/*
main gui command is just a demo app using the gui for prototyping and testing new ideas.
*/
package main

import (
	"flag"
	"image"

	"github.com/steampoweredtaco/legion-van/bananoutils"
	"github.com/steampoweredtaco/legion-van/gui"

	"github.com/gdamore/tcell/v2"
	log "github.com/sirupsen/logrus"
	legion "github.com/steampoweredtaco/legion-van/image"
)

func main() {
	log.SetReportCaller(true)
	debug := flag.Bool("debug", false, "set gui to virtual screen so debug can happen")
	flag.Parse()

	guiApp := gui.NewMainApp*"legion-ban", log.StandardLogger())
	if *debug {
		simScreen := tcell.NewSimulationScreen("")
		simScreen.SetSize(80, 25)
		guiApp.SetTerminalScreen(simScreen)
	}

	var accounts []bananoutils.Account
	for i := 0; i < 4; i++ {
		_, publicAccount, err := bananoutils.GeneratePrivateKeyAndFirstPublicAddress()
		if err != nil {
			panic(err)
		}
		accounts = append(accounts, publicAccount)

	}

	monKeys := make([]image.Image, 4)

	for i := 0; i < 4; i++ {
		r, err := bananoutils.GrabMonkey(accounts[i], legion.PNGFormat)
		if err != nil {
			log.Fatal(err)
		}
		img, _, err := image.Decode(r)
		if err != nil {
			log.Fatal(err)
		}
		monKeys[i] = img
	}

	guiApp.SetPreview1(monKeys[0])
	guiApp.SetPreview2(monKeys[1])
	guiApp.SetPreview3(monKeys[2])
	guiApp.SetPreview4(monKeys[3])
	defer func() {
		if r := recover(); r != nil {
			guiApp.ForceQuit()
			log.Error(r)
		}
	}()
	// go func() {
	// 	time.Sleep(10 * time.Second)
	// 	guiApp.Quit()
	// }()
	guiApp.Run()
}
