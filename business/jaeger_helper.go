package business

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/kiali/kiali/config"
	"github.com/kiali/kiali/log"
)

type Trace struct {
	Id string `json:"traceID"`
}

type RequestTrace struct {
	Traces []Trace `json:"data"`
}

type JaegerServices struct {
	Services []string `json:"data"`
}

var (
	JaegerAvailable = true
)

func getErrorTracesFromJaeger(namespace string, service string) (errorTraces int, err error) {
	errorTraces = 0
	err = nil
	if !JaegerAvailable || !config.Get().ExternalServices.Tracing.Enabled {
		return -1, errors.New("jaeger is not available")
	}
	if config.Get().ExternalServices.Tracing.EnableJaeger {
		u, errParse := url.Parse(fmt.Sprintf("http://%s%s/api/traces", config.Get().ExternalServices.Tracing.Service, config.Get().ExternalServices.Tracing.Path))
		if errParse != nil {
			log.Errorf("Error parse Jaeger URL fetching Error Traces: %s", err)
			return -1, errParse
		} else {
			q := u.Query()
			q.Set("lookback", "1h")
			q.Set("service", fmt.Sprintf("%s.%s", service, namespace))
			t := time.Now().UnixNano() / 1000
			q.Set("start", fmt.Sprintf("%d", t-60*60*1000*1000))
			q.Set("end", fmt.Sprintf("%d", t))
			q.Set("tags", "{\"error\":\"true\"}")
			u.RawQuery = q.Encode()
			timeout := time.Duration(1000 * time.Millisecond)
			client := http.Client{
				Timeout: timeout,
			}
			resp, reqError := client.Get(u.String())
			if reqError != nil {
				log.Errorf("Error fetching Jaeger Error Traces: %s", reqError)
				return -1, reqError
			} else {
				defer resp.Body.Close()
				body, errRead := ioutil.ReadAll(resp.Body)
				if errRead != nil {
					log.Errorf("Error Reading Jaeger Response fetching Error Traces: %s", errRead)
					err = errRead
					return -1, err
				}
				var traces RequestTrace
				if errMarshal := json.Unmarshal([]byte(body), &traces); errMarshal != nil {
					log.Errorf("Error Unmarshal Jaeger Response fetching Error Traces: %s", errRead)
					err = errMarshal
					return -1, err
				}
				errorTraces = len(traces.Traces)
			}
		}
	}
	return errorTraces, err
}

func GetServices() (services JaegerServices, err error) {
	services = JaegerServices{Services: []string{}}
	err = nil
	u, err := url.Parse(fmt.Sprintf("http://%s%s/api/services", config.Get().ExternalServices.Tracing.Service, config.Get().ExternalServices.Tracing.Path))
	if err != nil {
		log.Errorf("Error parse Jaeger URL fetching Services: %s", err)
		return services, err
	}
	timeout := time.Duration(1000 * time.Millisecond)
	client := http.Client{
		Timeout: timeout,
	}
	resp, reqError := client.Get(u.String())
	if reqError != nil {
		err = reqError
	} else {
		defer resp.Body.Close()
		body, errRead := ioutil.ReadAll(resp.Body)
		if errRead != nil {
			log.Errorf("Error Reading Jaeger Response fetching Services: %s", errRead)
			err = errRead
			return services, err
		}
		if errMarshal := json.Unmarshal([]byte(body), &services); errMarshal != nil {
			log.Errorf("Error Unmarshal Jaeger Response fetching Services: %s", errRead)
			err = errMarshal
			return services, err
		}
	}
	return services, err
}
