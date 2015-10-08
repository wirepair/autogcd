package main

import (
	"flag"
	"github.com/wirepair/autogcd"
	"io/ioutil"
	"log"
	"time"
)

var (
	chromePath string
	userDir    string
	chromePort string
	debug      bool
)

// For an excellent list of command line switches see: http://peter.sh/experiments/chromium-command-line-switches/
var startupFlags = []string{"--disable-new-tab-first-run", "--no-first-run", "--disable-translate"}
var waitForTimeout = time.Second * 5
var waitForRate = time.Millisecond * 25

var navigationTimeout = time.Second * 10

var stableAfter = time.Millisecond * 450
var stabilityTimeout = time.Second * 2

func init() {
	flag.StringVar(&chromePath, "chrome", "C:\\Program Files (x86)\\Google\\Chrome\\Application\\chrome.exe", "path to Xvfb")
	flag.StringVar(&userDir, "dir", "C:\\temp\\", "user directory")
	flag.StringVar(&chromePort, "port", "9222", "Debugger port")
	flag.BoolVar(&debug, "debug", false, "Show debug DOM node event changes")
}

func main() {
	flag.Parse()
	settings := autogcd.NewSettings(chromePath, randUserDir())
	settings.RemoveUserDir(true)           // clean up user directory after exit
	settings.AddStartupFlags(startupFlags) // disable new tab junk

	auto := autogcd.NewAutoGcd(settings) // create our automation debugger
	auto.Start()                         // start it
	defer auto.Shutdown()

	tab, err := auto.GetTab() // get the first visual tab
	if err != nil {
		log.Fatalf("error getting visual tab to work with")
	}
	configureTab(tab)

	if _, err := tab.Navigate("https://www.google.co.jp"); err != nil {
		log.Fatalf("error going to google: %s\n", err)
	}
	log.Printf("navigation complete")

	err = tab.WaitFor(waitForRate, waitForTimeout, autogcd.ElementByIdReady(tab, "lst-ib"))
	if err != nil {
		log.Println("timed out waiting for search field")
	}

	log.Printf("getting search element")
	ele, _, err := tab.GetElementById("lst-ib")
	if err != nil {
		log.Fatalf("error finding search element: %s\n", err)
	}

	log.Println("sending keys")
	err = ele.SendKeys("github gcd\n") // use \n to hit enter
	if err != nil {
		log.Fatalf("error sending keys to element: %s\n", err)
	}
	err = tab.WaitFor(waitForRate, waitForTimeout, autogcd.TitleContains(tab, "github gcd"))
	if err != nil {
		log.Println("timed out waiting for title to change to github gcd")
	}

	log.Println("waiting for stability")
	err = tab.WaitStable()
	if err != nil {
		log.Printf("stability timed out: %s\n", err)
	}

	log.Println("getting search elements")
	eles, err := tab.GetElementsBySelector("a")
	for _, ele := range eles {
		link := ele.GetAttribute("href")
		if link == "https://github.com/wirepair/gcd" {
			log.Println("found the best link there is")
			loadGcd(ele, tab)
			break
		}
	}
	log.Printf("Done")
}

// Set various timeouts
func configureTab(tab *autogcd.Tab) {
	tab.SetNavigationTimeout(navigationTimeout) // give up after 10 seconds for navigating, default is 30 seconds
	tab.SetStabilityTime(stableAfter)
	if debug {
		domHandlerFn := func(tab *autogcd.Tab, change *autogcd.NodeChangeEvent) {
			log.Printf("change event %s occurred\n", change.EventType)
		}
		tab.GetDOMChanges(domHandlerFn)
	}
}

func loadGcd(ele *autogcd.Element, tab *autogcd.Tab) {
	log.Printf("clicking google link\n")
	err := ele.Click()
	if err != nil {
		log.Fatalf("error clicking google link: %s\n", err)
	}
	tab.WaitFor(waitForRate, waitForTimeout, autogcd.TitleContains(tab, "wirepair/gcd"))
	log.Printf("here we are, bask in its glory")
	time.Sleep(5 * time.Second)
}

func randUserDir() string {
	dir, err := ioutil.TempDir(userDir, "autogcd")
	if err != nil {
		log.Fatalf("error getting temp dir: %s\n", err)
	}
	return dir
}
