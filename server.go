package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/byuoitav/event-forwarding/logger"
	"github.com/labstack/echo"
)

var eventForwarding chan info

type info struct {
	body   []byte
	id     string
	evtype string
}

func init() {
	eventForwarding = make(chan info, 1000000)
	workers := 10

	logger.L.Info("Starting workers")

	for i := 0; i < workers; i++ {
		go func() {

			b := <-eventForwarding
			url := ""
			if len(b.id) < 1 {
				url = fmt.Sprintf("http://av-elk-rapidmaster1:9200/events/%v", b.evtype)
			} else {
				url = fmt.Sprintf("http://av-elk-rapidmaster1:9200/events/%v/%v", b.evtype, b.id)
			}

			resp, err := http.Post(url, "appliciation/json", bytes.NewBuffer(b.body))
			if err != nil {
				logger.L.Infof("[forwarder] There was a problem sending the event: %v", err.Error())
			}
			defer resp.Body.Close()

			if resp.StatusCode/100 != 2 {
				logger.L.Infof("[forwarder] Non-200 response recieved: %v.", resp.StatusCode)

				b, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					logger.L.Infof("[forwarder] could not read body: %v", err.Error())
				}
				logger.L.Infof("[forwarder] response: %s", b)
			}

		}()
	}
	logger.L.Info("Done.")
}

func forwardUserEvent(context echo.Context) error {

	defer context.Request().Body.Close()
	b, err := ioutil.ReadAll(context.Request().Body)
	if err != nil {
		logger.L.Warn("Couldn't read body from: %v", context.Request().RemoteAddr)
		return context.JSON(http.StatusBadRequest, "")
	}

	info := info{}
	info.body = b
	info.evtype = context.Param("type")

	logger.L.Debugf("Logging from %v", context.Request().RemoteAddr)
	eventForwarding <- info

	return context.JSON(http.StatusOK, "")
}

func forwardEvent(context echo.Context) error {

	defer context.Request().Body.Close()
	b, err := ioutil.ReadAll(context.Request().Body)
	if err != nil {
		logger.L.Warn("Couldn't read body from: %v", context.Request().RemoteAddr)
		return context.JSON(http.StatusBadRequest, "")
	}

	info := info{}
	info.body = b
	info.evtype = context.Param("type")
	info.id = context.Param("id")

	logger.L.Debugf("Logging from %v", context.Request().RemoteAddr)
	eventForwarding <- info

	return context.JSON(http.StatusOK, "")
}

func baselineUserEvents(context echo.Context) error {
	defer context.Request().Body.Close()
	b, err := ioutil.ReadAll(context.Request().Body)
	if err != nil {
		logger.L.Warn("Couldn't read body from: %v", context.Request().RemoteAddr)
		return context.JSON(http.StatusBadRequest, "")
	}

	info := info{}
	info.body = b
	info.evtype = "user"

	logger.L.Debugf("Logging from %v", context.Request().RemoteAddr)
	eventForwarding <- info

	return context.JSON(http.StatusOK, "")
}

func justTellMe(context echo.Context) error {
	logger.L.Infof("url: %v", context.Request().URL)
	return context.JSON(http.StatusOK, "")
}

func main() {
	logger.L.Info("Setting up server...")

	port := ":80"
	router := echo.New()
	router.GET("/events/:type/:id", forwardEvent)
	router.PUT("/events/:type/:id", forwardEvent)

	router.POST("/", baselineUserEvents)
	router.POST("/events", baselineUserEvents)
	router.POST("/events/:type/:id", forwardEvent)
	router.POST("/events/:type", forwardUserEvent)

	server := http.Server{
		Addr:           port,
		MaxHeaderBytes: 1024 * 10,
	}

	logger.L.Info("Starting server...")
	err := router.StartServer(&server)
	logger.L.Errorf("%v", err.Error())
}
