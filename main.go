package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"text/template"
	"time"
)

const (
	jsontpl = `
	{
	"blocks" : [
		{
			"type": "section",
			"text": {
				"text": "*Consul service(s) check(s) are failing!* :disappointed:",
				"type": "mrkdwn"
			}
		},
			{
				"type": "divider"  
			},  
			{
					"type": "section",
					"text" : {
							"text" : "*Summary:* _*{{len .}}* checks are failing_. Check below for more info.",
							"type" : "mrkdwn"
					}       
			},
			{{range .}}
			{
				"type" : "section",
				"text": {
					"text": "*Service name:* _{{.ServiceName}}_",
					"type": "mrkdwn"
				},
				"fields" : [
					{
						"type" : "mrkdwn",
						"text" : "*Host Name*: {{.Node}}"
					}
					,
				  {
						"type" : "mrkdwn",
						"text" : "*Check Name*: {{.Name}}"
					}
					,
					{
						"type" : "mrkdwn",
						"text" : "*Check ID*: {{.CheckID}}"
					}
					,
					{
						"type" : "mrkdwn",
						"text" : "*Check Output*: {{.Output}}"
					}
				]
			},
			{
				"type": "divider"
			},
			{{end}}
			{
				"type" : "section",
				"text" : {
					"text" : "May the force be with you!",
					"type" : "plain_text"
				}
			}
			
	]
}
 `
)

var (
	slackURLPtr = flag.String("slackurl", "", "A url to your slack incoming webhook. (Required)")
	portPtr     = flag.Int64("port", 9000, "A port for this app to listen on.")
)

func init() {

	flag.Usage = func() {
		fmt.Println("....")
		fmt.Println("Options:")
		flag.PrintDefaults()
	}

	flag.Parse()

	if flag.NFlag() == 0 || *slackURLPtr == "" {
		flag.Usage()
		os.Exit(0)
	}

}

//Check represnets the json data send by consul when a watch event occurs
type Check struct {
	Node        string
	CheckID     string
	Name        string
	Status      string
	Notes       string
	Output      string
	ServiceID   string
	ServiceName string
	ServiceTags []string
	Definition  map[string]interface{}
}

//Notifier provides an abstraction for a notification channel
type Notifier interface {
	Notify([]byte) error
}

type slackNotifier struct {
	slackURL     string
	slackDataTpl string
}

func (sn slackNotifier) sendToSlack(data []byte) error {
	timeout := time.Duration(4 * time.Second)
	client := http.Client{
		Timeout: timeout,
	}
	resp, err := client.Post(sn.slackURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New("http: the request was not succesful. Http response code: " + string(resp.StatusCode))
	}
	return nil
}

//Notify sends a slack mesasge using slack's incoming webhook
func (sn slackNotifier) Notify(data []byte) error {
	checks, err := parseResponseJSON(data)
	if err != nil {
		return err
	}
	if len(checks) > 0 {
		var tplbuffer bytes.Buffer
		tp, err := template.New("slackNotify").Parse(sn.slackDataTpl)
		if err != nil {
			return err
		}
		err = tp.Execute(&tplbuffer, checks)
		if err != nil {
			return err
		}
		err = sn.sendToSlack(tplbuffer.Bytes())
		if err != nil {
			return err
		}
	}
	return nil
}

func consulNotifyHandler(h http.HandlerFunc, n Notifier) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r)
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading request body",
				http.StatusInternalServerError)
		}
		err = n.Notify(body)
		if err != nil {
			//Log error
			http.Error(w, err.Error(),
				http.StatusInternalServerError)
		}
	})
}

func mainHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
}

func parseResponseJSON(data []byte) ([]Check, error) {
	var checks []Check
	err := json.Unmarshal(data, &checks)
	if err != nil {
		return nil, err
	}
	return checks, nil
}

func main() {
	sn := slackNotifier{slackURL: *slackURLPtr, slackDataTpl: jsontpl}
	http.HandleFunc("/watch/checks", consulNotifyHandler(mainHandler, sn))
	log.Fatal(http.ListenAndServe(":"+fmt.Sprint(*portPtr), nil))
}
