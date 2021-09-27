package interceptingBard

import (
	"errors"
	"log"
	"time"

	"github.com/mxschmitt/playwright-go"
)

type pageContextSet struct {
	pw      *playwright.Playwright
	browser playwright.Browser
	page    playwright.Page
}

// NewCtx creates, starts and returns a scraper object
func NewCtx() *pageContextSet {
	p := new(pageContextSet)
	p.spinUpPage()
	return p
}

func (ctx *pageContextSet) spinUpPage() {
	var err error
	ctx.pw, err = playwright.Run()
	if err != nil {
		log.Fatalf("could not launch playwright: %+v", err)
	}
	browser, err := ctx.pw.Firefox.Launch()
	if err != nil {
		log.Fatalf("could not launch Browser: %+v", err)
	}
	ctx.browser = browser
	newPage, err := browser.NewPage()
	ctx.page = newPage
	if err != nil {
		log.Fatalf("could not create page: %+v", err)
	}
}

// Close stops the browser and stops playwright
func (ctx *pageContextSet) Close() {
	ctx.spinDownPage()
}

func (ctx *pageContextSet) spinDownPage() {
	err := ctx.page.Close()
	if err != nil {
		log.Fatalf("page could not be closed: %+v", err)
	}
	err = ctx.browser.Close()
	if err != nil {
		log.Fatalf("could not close browser: %+v", err)
	}
	err = (*ctx.pw).Stop()
	if err != nil {
		log.Fatalf("could not stop Playwright: %+v", err)
	}
}

func (ctx *pageContextSet) GetResponseByClick(selector string, route interface{}) (playwright.Response, error) {
	// get the next button
	nextButton, err := ctx.page.QuerySelector(selector)
	if err != nil {
		log.Fatalf("could not get '"+selector+"' element: %+v", err)
	}

	// intercept the route
	intercepted := make(chan playwright.Response, 1)
	log.Println("attempting intercept")
	browserContext := ctx.page.Context()
	err = browserContext.Route(route,
		func(route playwright.Route, request playwright.Request) {
			// route.Continue here is need here - otherwise execution of the request may stop.
			err = route.Continue()
			if err != nil {
				log.Fatalf("route continue failed: %+v", err)
			}
			time.Sleep(time.Millisecond * 500)
			response, err := request.Response()
			if err != nil {
				log.Fatalf("could not get response object after click: %+v", err)
			}
			intercepted <- response
		},
	)
	if err != nil {
		log.Fatalf("setting page route failed: %+v", err)
	}

	err = nextButton.Click()
	if err != nil {
		log.Fatalf("could not click button element!: %+v", err)
	}
	log.Println("clicked!")

	// TODO this works but the pw api has some options structures with timeouts
	var response playwright.Response
	timeout := 30000 * time.Millisecond
	select {
	case response = <-intercepted:
		err = nil
	case <-time.After(timeout):
		response = nil
		err = errors.New("response timeout")
	}

	browserContext.Unroute(route)

	return response, err
}

func (ctx *pageContextSet) GetPage(url string) playwright.Page {
	_, err := ctx.page.Goto(url)
	if err != nil {
		log.Fatalf("could not get link (%v): %v", url, err)
	}
	return ctx.page
}