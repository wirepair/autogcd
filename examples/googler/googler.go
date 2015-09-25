package main

import (
	"flag"
	"github.com/wirepair/autogcd"
	"io/ioutil"
	"log"
)

var (
	chromePath string
	userDir    string
	chromePort string
)
var done chan struct{}

// For an excellent list of command line switches see: http://peter.sh/experiments/chromium-command-line-switches/
var startupFlags = []string{"--disable-new-tab-first-run", "--no-first-run"}

func init() {
	flag.StringVar(&chromePath, "chrome", "C:\\Program Files (x86)\\Google\\Chrome\\Application\\chrome.exe", "path to Xvfb")
	flag.StringVar(&userDir, "dir", "C:\\temp\\", "user directory")
	flag.StringVar(&chromePort, "port", "9222", "Debugger port")
}

func main() {
	flag.Parse()
	settings := autogcd.NewSettings(chromePath, randUserDir())
	settings.RemoveUserDir(true)           // clean up user directory after exit
	settings.AddStartupFlags(startupFlags) // disable new tab junk

	auto := autogcd.NewAutoGcd(settings) // create our automation debugger
	auto.Start()                         // start it
	defer auto.Shutdown()
	done = make(chan struct{})

	tab, err := auto.GetTab() // get the first visual tab
	if err != nil {
		log.Fatalf("error getting visual tab to work with")
	}

	tab.SetNavigationTimeout(10) // give up after 10 seconds for navigating, default is 30 seconds
	tab.SetStabilityTime(450)
	if _, err := tab.Navigate("https://www.google.co.jp"); err != nil {
		log.Fatalf("error going to google!")
	}

	domHandlerFn := func(tab *autogcd.Tab, change *autogcd.NodeChangeEvent) {
		//log.Printf("change event %s occurred\n", change.EventType)
	}
	tab.SetDomChangeHandler(domHandlerFn)

	ele, ready, err := tab.GetElementById("lst-ib")
	if err != nil {
		log.Fatalf("error finding search element")
	}

	if !ready {
		ele.WaitForReady() // make sure this element is ready
	}

	log.Println("sending keys")
	err = ele.SendKeys("github gcd\n") // use \n to hit enter
	if err != nil {
		log.Fatalf("error sending keys to element: %s\n", err)
	}

	log.Println("waiting for stability")
	err = tab.WaitStable()
	if err != nil {
		log.Printf("transition failed: %s\n", err)
	}

	log.Println("getting search elements")
	eles, err := tab.GetElementsBySelector("a")
	for _, ele := range eles {
		log.Printf("got ele: %s\n", ele.NodeId())
		link := ele.GetAttribute("href")
		if link == "https://github.com/wirepair/gcd" {
			log.Println("found the best link there is")
			starGcd(ele, tab)
			return
		}
	}
	<-done

}

func starGcd(ele *autogcd.Element, tab *autogcd.Tab) {
	log.Printf("clicking google link\n")
	tab.DebugEvents(true)
	err := ele.Click()
	if err != nil {
		log.Fatalf("error clicking google link: %s\n", err)
	}
	err = tab.WaitStable()
	if err != nil {
		log.Printf("transition failed: %s\n", err)
	}
	log.Println("getting buttons")
	eles, err := tab.GetElementsBySelector("button")

	for _, ele := range eles {
		title := ele.GetAttribute("title")
		if title == "Star wirepair/gcd" {
			log.Printf("found the star button!")
			ele.Click()
			err = tab.WaitTransitioning(250, true)
			if err != nil {
				log.Printf("failed staring gcd: %s\n", err)

			}
			log.Printf("DONE")
			done <- struct{}{}
		}
	}

}

func randUserDir() string {
	dir, err := ioutil.TempDir(userDir, "autogcd")
	if err != nil {
		log.Fatalf("error getting temp dir: %s\n", err)
	}
	return dir
}
