package main

import (
	"log"
	"os"

	"github.com/Nordgedanken/Morpheus/matrix"
	"github.com/Nordgedanken/Morpheus/util"
	"github.com/matrix-org/gomatrix"
	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/gui"
	"github.com/therecipe/qt/widgets"
	"github.com/tidwall/buntdb"
	"sync"
)

var window *widgets.QMainWindow
var localLog *log.Logger

func main() {
	var file *os.File
	var err error

	localLog = util.Logger()
	localLog, file, err = util.StartFileLog(localLog)
	if err != nil {
		localLog.Fatalln(err)
	}
	defer file.Close()

	db, DBOpenErr := matrix.OpenDB()
	if DBOpenErr != nil {
		localLog.Fatalln(DBOpenErr)
	}

	localLog.Println("Starting Morpheus")

	app := widgets.NewQApplication(len(os.Args), os.Args)

	app.SetAttribute(core.Qt__AA_UseHighDpiPixmaps, true)
	app.SetApplicationName("Morpheus")
	app.SetApplicationVersion("0.0.1")
	appIcon := gui.NewQIcon5(":/qml/resources/logos/MorpheusBig.png")
	app.SetWindowIcon(appIcon)

	window = widgets.NewQMainWindow(nil, 0)
	windowHeight := 600
	windowWidth := 950

	desktopApp := widgets.QApplication_Desktop()
	primaryScreen := desktopApp.PrimaryScreen()
	screen := desktopApp.Screen(primaryScreen)
	windowX := (screen.Width() - windowHeight) / 2
	windowY := (screen.Height() - windowWidth) / 2

	window.Move2(windowX, windowY)

	var accessToken string
	var homeserverURL string
	var userID string

	// Get cache
	DBErr := db.View(func(tx *buntdb.Tx) error {
		accessTokenErr := tx.AscendKeys("user:accessToken",
			func(key, value string) bool {
				accessToken = value
				return true
			})
		if accessTokenErr != nil {
			return accessTokenErr
		}
		homeserverURLErr := tx.AscendKeys("user:homeserverURL",
			func(key, value string) bool {
				homeserverURL = value
				return true
			})
		if homeserverURLErr != nil {
			return homeserverURLErr
		}
		userIDErr := tx.AscendKeys("user:userID",
			func(key, value string) bool {
				userID = value
				return true
			})
		return userIDErr
	})
	if DBErr != nil {
		localLog.Fatalln(DBErr)
	}
	db.Close()

	if accessToken != "" && homeserverURL != "" && userID != "" {
		var wg sync.WaitGroup
		localLog.Println("Starting Auto Login Sequenze in background")
		results := make(chan *gomatrix.Client)

		wg.Add(1)
		go matrix.DoLogin("", "", homeserverURL, userID, accessToken, localLog, results, &wg)

		go func() {
			wg.Wait()      // wait for each execTask to return
			close(results) // then close the results channel
		}()

		//Show MainUI
		for result := range results {
			//TODO Don't switch screen on wrong login data.
			mainUI, mainUIErr := NewMainUI(windowWidth, windowHeight, result, window)
			if mainUIErr != nil {
				localLog.Fatalln(mainUIErr)
				return
			}
			mainUI.Resize2(windowWidth, windowHeight)
			window.SetCentralWidget(mainUI)
		}
	} else {
		//Show loginUI
		loginUI, loginUIErr := NewLoginUI(windowWidth, windowHeight, window)
		if loginUIErr != nil {
			localLog.Fatalln(loginUIErr)
			return
		}
		loginUI.Resize2(windowWidth, windowHeight)
		window.SetCentralWidget(loginUI)
	}

	window.Resize2(windowWidth, windowHeight)
	window.Show()

	//enter the main event loop
	_ = widgets.QApplication_Exec()
	localLog.Println("Stopping Morpheus")
}
