package main

import (
	"bytes"
	"io/ioutil"
	"net/http"

	"github.com/byuoitav/event-forwarding/logger"
	"github.com/labstack/echo"
)

var eventForwarding chan []byte

func init() {
	eventForwarding = make(chan []byte, 1000000)
	forwardingurl := "av-elk-rapidmaster1:9200"
	workers := 10

	logger.L.Info("Starting workers")

	for i := 0; i < workers; i++ {
		go func() {

			b := <-eventForwarding

			resp, err := http.Post(forwardingurl, "appliciation/json", bytes.NewBuffer(b))
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

func forwardEvent(context echo.Context) error {

	defer context.Request().Body.Close()
	b, err := ioutil.ReadAll(context.Request().Body)
	if err != nil {
		logger.L.Warn("Couldn't read body from: %v", context.Request().RemoteAddr)
		return context.JSON(http.StatusBadRequest, "")
	}

	logger.L.Debugf("Logging from %v", context.Request().RemoteAddr)
	eventForwarding <- b

	return context.JSON(http.StatusOK, "")
}

func main() {
	logger.L.Info("Setting up server...")

	port := ":80"
	router := echo.New()
	router.GET("/", forwardEvent)
	router.PUT("/", forwardEvent)
	router.POST("/", forwardEvent)

	server := http.Server{
		Addr:           port,
		MaxHeaderBytes: 1024 * 10,
	}

	logger.L.Info("Starting server...")
	err := router.StartServer(&server)
	logger.L.Errorf("%v", err.Error())
}
