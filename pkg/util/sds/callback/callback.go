package sdscallback

import (
	"bytes"
	"crypto/tls"
	"github.com/sirupsen/logrus"
	"net/http"
	"time"
)

const (
	CallbackRetryLimit   = 3
	CallbackRetryTimeout = 10 * time.Minute
)

func DoCallback(URL string, message CallbackMessage) {
	messageJSON, err := message.ToJSON()
	if err != nil {
		logrus.Errorf("could not convert message to json")
		return
	}

	go SubmitMessage(URL, IsHTTPS(URL), messageJSON)
}

func IsHTTPS(input string) bool {
	const (
		HTTPSProtocol = "https"
	)
	if len(input) > len(HTTPSProtocol) && input[0:len(HTTPSProtocol)] == HTTPSProtocol {
		return true
	}
	return false
}

func SubmitMessage(URL string, https bool, message []byte) {
	var client *http.Client
	retries := 0
	for retries < CallbackRetryLimit {
		if https {
			client = &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
				},
			}
			logrus.Infof("Picking HTTPS for %s", URL)
		} else {
			client = &http.Client{}
			logrus.Infof("Picking HTTP for %s", URL)
		}
		request, err := http.NewRequest("PUT", URL, bytes.NewBuffer(message))
		request.Header.Set("Content-Type", "application/json")
		response, err := client.Do(request)
		if err == nil && response.StatusCode == 200 {
			logrus.Infof("successfully submitted message to %s", URL)
			return
		}
		logrus.Infof("unsuccessfully submitted message to %s, error was %s", URL, response.Status)
		retries++
		time.Sleep(CallbackRetryTimeout)
	}
	logrus.Infof("giving up on sending message to %s", URL)
}
