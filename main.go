package main

import (
	"bytes"
	"context"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/samwho/streamdeck"
	"github.com/wobsoriano/go-jxa"
)

var (
	pluginBaseName string = "com.onamish.streamdeck-plugins-jacobfg"
)

func main() {
	f, err := ioutil.TempFile("", pluginBaseName+".log")
	if err != nil {
		log.Fatalf("error creating temp file: %v", err)
	}
	defer f.Close()

	log.SetOutput(f)

	ctx := context.Background()
	if err := run(ctx); err != nil {
		log.Fatalf("%v\n", err)
	}
}

func run(ctx context.Context) error {
	params, err := streamdeck.ParseRegistrationParams(os.Args)
	if err != nil {
		return err
	}

	client := streamdeck.NewClient(ctx, params)
	setup(client)

	return client.Run()
}

func setup(client *streamdeck.Client) {

	actionsNames := []string{
		"left-half",
		"right-half",
		"top-left",
		"top-right",
		"bottom-left",
		"bottom-right",
		"first-third",
		"center-third",
		"last-third",
		"top-left-sixth",
		"top-center-sixth",
		"top-right-sixth",
		"bottom-left-sixth",
		"bottom-center-sixth",
		"bottom-right-sixth",
	}

	for _, actionName := range actionsNames {

		action := client.Action(pluginBaseName + ".rectangle." + actionName)
		action.RegisterHandler(streamdeck.KeyDown, func(ctx context.Context, client *streamdeck.Client, event streamdeck.Event) error {
			log.Default().Printf("KeyDown: %+v", event)

			callRectangle(event.Action[strings.LastIndex(event.Action, ".")+1:])
			//ignore errors

			// return fmt.Errorf("couldn't find settings for context %v", event.Context)
			return nil
		})
	}

	action := client.Action(pluginBaseName + ".google-meet.find-tab")
	action.RegisterHandler(streamdeck.KeyDown, func(ctx context.Context, client *streamdeck.Client, event streamdeck.Event) error {
		log.Default().Printf("KeyDown: %+v", event)

		code := `
		(function() {
			var chrome = Application('Google Chrome');
			if (chrome.running()) {
			  	for (win of chrome.windows()) {
					var tabIndex =
						win.tabs().findIndex(tab => tab.url().match(/meet.google.com/));
					if (tabIndex != -1) {
						chrome.activate();
						win.activeTabIndex = (tabIndex + 1);
						win.index = 1;
					}
			  	}
			}
		})();
		`
		v, err := jxa.RunJXA(code)

		if err != nil {
			log.Fatal(err.Error())
		}

		log.Default().Printf("Is dark mode: %s", v)

		return nil
	})
}

func callRectangle(name string) {
	log.Default().Printf("open -g rectangle://execute-action?name=" + name)
	cmd := exec.Command("open", "-g", "rectangle://execute-action?name="+name)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	_ = cmd.Run()
	// if err != nil {
	//     log.Fatalf("cmd.Run() failed with %s\n", err)
	// }
	outStr, errStr := string(stdout.Bytes()), string(stderr.Bytes())
	if len(outStr) > 0 {
		log.Default().Printf("out:\n%s\n", outStr)
	}
	if len(errStr) > 0 {
		log.Default().Printf("err:\n%s\n", errStr)
	}
}
