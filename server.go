package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/byuoitav/event-forwarding/logger"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

var eventForwarding chan info

type info struct {
	body     []byte
	id       string
	evtype   string
	index    string
	endpoint string
}

func init() {
	eventForwarding = make(chan info, 1000000)
	workers := 10

	logger.L.Info("Starting workers")

	for i := 0; i < workers; i++ {
		go func() {
			for {

				b := <-eventForwarding

				logger.L.Debugf("event: %s", b.body)

				url := ""
				if len(b.id) < 1 && len(b.index) > 0 {
					url = fmt.Sprintf("http://av-elk-rapidmaster1:9200/%v/%v", b.index, b.evtype)
				} else if len(b.id) > 0 {
					url = fmt.Sprintf("http://av-elk-rapidmaster1:9200/%v/%v/%v", b.index, b.evtype, b.id)
				} else if len(b.endpoint) > 0 {
					url = fmt.Sprintf("http://av-elk-rapidmaster1:9200%v", b.endpoint)
				} else {
					logger.L.Infof("No endpoint information, discarding...")
					continue
				}

				resp, err := http.Post(url, "appliciation/json", bytes.NewBuffer(b.body))
				if err != nil {
					logger.L.Infof("[forwarder] There was a problem sending the event: %v", err.Error())
					continue
				}

				reBody, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					logger.L.Infof("[forwarder] could not read body: %v", err.Error())
					continue
				}
				if resp.StatusCode/100 != 2 {
					logger.L.Infof("[forwarder] Non-200 response recieved: %v. %s.", resp.StatusCode, reBody)
				} else {
					logger.L.Infof("[forwarder] good response: %s", reBody)
				}
				resp.Body.Close()
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

	info := info{index: "events"}
	info.body = b
	info.evtype = context.Param("type")

	logger.L.Debugf("Logging user from %v", context.Request().RemoteAddr)
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

	info := info{index: "events"}
	info.body = b
	info.evtype = context.Param("type")
	info.id = context.Param("id")

	logger.L.Debugf("Logging general from %v", context.Request().RemoteAddr)
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

	info := info{index: "events"}
	info.body = b
	info.evtype = "user"

	logger.L.Debugf("Logging baseline from %v", context.Request().RemoteAddr)
	eventForwarding <- info

	return context.JSON(http.StatusOK, "")
}

func forwardGenericEvent(context echo.Context) error {

	defer context.Request().Body.Close()
	b, err := ioutil.ReadAll(context.Request().Body)
	if err != nil {
		logger.L.Warn("Couldn't read body from: %v", context.Request().RemoteAddr)
		return context.JSON(http.StatusBadRequest, "")
	}

	var i info

	if len(context.Param("index")) > 0 {
		logger.L.Debugf("Logging generic from %v", context.Request().RemoteAddr)
		i = info{
			index:  context.Param("index"),
			body:   b,
			evtype: context.Param("type"),
		}
	} else {
		logger.L.Debugf("Logging root match %v from %v.", context.Request().URL.Path, context.Request().RemoteAddr)
		i = info{
			body:     b,
			endpoint: context.Request().URL.Path,
		}
	}

	eventForwarding <- i

	return context.JSON(http.StatusOK, "")
}

func main() {
	logger.L.Info("Setting up server...")

	port := ":80"

	router := echo.New()
	router.Pre(middleware.RemoveTrailingSlash())

	router.GET("/events/:type/:id", forwardEvent)
	router.PUT("/events/:type/:id", forwardEvent)

	router.POST("/", baselineUserEvents)
	router.POST("/events", baselineUserEvents)
	router.POST("/events/:type/:id", forwardEvent)
	router.POST("/events/:type", forwardUserEvent)

	router.POST("/:index/:type", forwardGenericEvent)

	router.POST("/user/*", forwardGenericEvent)
	router.POST("/events/*", forwardGenericEvent)
	router.POST("/*", forwardGenericEvent)

	server := http.Server{
		Addr:           port,
		MaxHeaderBytes: 1024 * 10,
	}

	logger.L.Info("Starting server...")

	err := router.StartServer(&server)

	logger.L.Errorf("%v", err.Error())
}
