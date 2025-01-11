package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/samwho/streamdeck"
	"github.com/wobsoriano/go-jxa"
)

var (
	pluginBaseName       string        = "com.onamish.streamdeck-plugins-jacobfg"
	neoShadeOpenCmd      string        = "-up!"
	neoShadeCloseCmd     string        = "-dn!"
	neoShadeStopCmd      string        = "-sp!"
	neoShadeFavouriteCmd string        = "-gp!"
	neoShadeTimeout      time.Duration = time.Second * 5
)

type AudioSettings struct {
	Settings struct {
		Input  string `json:"inputDevice,omitempty"`
		Output string `json:"outputDevice,omitempty"`
	} `json:"settings"`
}

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
		_, err := jxa.RunJXA(code)

		if err != nil {
			log.Fatal(err.Error())
		}

		// log.Default().Printf("Is dark mode: %s", v)

		return nil
	})

	actionAudio := client.Action(pluginBaseName + ".audio")
	actionAudio.RegisterHandler(streamdeck.KeyDown, func(ctx context.Context, client *streamdeck.Client, event streamdeck.Event) error {
		log.Default().Printf("KeyDown: %+v", event)

		log.Default().Printf("Payload: %s", event.Payload)
		payload := &AudioSettings{}
		err := json.Unmarshal(event.Payload, payload)
		if err != nil {
			log.Fatal(err.Error())
			return err
		}

		if payload.Settings.Input != "" {
			cmd := exec.Command("/opt/homebrew/bin/SwitchAudioSource", "-t", "input", "-s", payload.Settings.Input)
			if err := cmd.Run(); err != nil {
				log.Fatal(err)
			}
		}

		if payload.Settings.Output != "" {
			cmd := exec.Command("/opt/homebrew/bin/SwitchAudioSource", "-t", "output", "-s", payload.Settings.Output)
			if err := cmd.Run(); err != nil {
				log.Fatal(err)
			}
		}

		return nil
	})

	shutterOpenAction := client.Action(pluginBaseName + ".shutter-open")
	shutterOpenAction.RegisterHandler(streamdeck.KeyDown, func(ctx context.Context, client *streamdeck.Client, event streamdeck.Event) error {
		log.Default().Printf("KeyDown: %+v", event)
		return shutterAction(event, neoShadeOpenCmd)
	})

	shutterCloseAction := client.Action(pluginBaseName + ".shutter-close")
	shutterCloseAction.RegisterHandler(streamdeck.KeyDown, func(ctx context.Context, client *streamdeck.Client, event streamdeck.Event) error {
		log.Default().Printf("KeyDown: %+v", event)
		return shutterAction(event, neoShadeCloseCmd)
	})

	shutterStopAction := client.Action(pluginBaseName + ".shutter-stop")
	shutterStopAction.RegisterHandler(streamdeck.KeyDown, func(ctx context.Context, client *streamdeck.Client, event streamdeck.Event) error {
		log.Default().Printf("KeyDown: %+v", event)
		return shutterAction(event, neoShadeStopCmd)
	})

	shutterFavouriteAction := client.Action(pluginBaseName + ".shutter-favourite")
	shutterFavouriteAction.RegisterHandler(streamdeck.KeyDown, func(ctx context.Context, client *streamdeck.Client, event streamdeck.Event) error {
		log.Default().Printf("KeyDown: %+v", event)
		return shutterAction(event, neoShadeFavouriteCmd)
	})

	sleepAction := client.Action(pluginBaseName + ".sleep")
	sleepAction.RegisterHandler(streamdeck.KeyDown, func(ctx context.Context, client *streamdeck.Client, event streamdeck.Event) error {
		log.Default().Printf("KeyDown: %+v", event)
		log.Default().Printf("Payload: %s", event.Payload)

		payload := &SleepPropertyInspector{Settings: SleepPropertyInspectorSettings{Duration: "5"}}
		err := json.Unmarshal(event.Payload, payload)
		if err != nil {
			log.Fatal(err.Error())
			return err
		}
		duration, err := strconv.ParseInt(payload.Settings.Duration, 10, 32)
		if err != nil {
			log.Fatal(err.Error())
			return err
		}

		time.Sleep(time.Duration(duration) * time.Second)

		return nil
	})
}

type SleepPropertyInspector struct {
	Settings SleepPropertyInspectorSettings `json:"settings"`
}
type SleepPropertyInspectorSettings struct {
	Duration string `json:"sleepDuration"`
}

type ShutterPropertyInspectorSettings struct {
	Settings struct {
		Address   string `json:"address"`
		ShadeId   string `json:"shadeId"`
		MotorType string `json:"motorType"`
	} `json:"settings"`
}

func shutterAction(event streamdeck.Event, command string) error {

	log.Default().Printf("Payload: %s", event.Payload)
	payload := &ShutterPropertyInspectorSettings{}
	err := json.Unmarshal(event.Payload, payload)
	if err != nil {
		log.Fatal(err.Error())
		return err
	}
	neoController := payload.Settings.Address
	remoteCommand := payload.Settings.ShadeId + command + payload.Settings.MotorType
	log.Default().Printf("Address: %s", neoController)
	log.Default().Printf("Command: %s", remoteCommand)

	con, err := net.DialTimeout("tcp", neoController, neoShadeTimeout)

	if err != nil {
		log.Default().Printf("Error connecting to %s: %v", neoController, err)
		return err
	}

	defer con.Close()

	_, err = con.Write([]byte(remoteCommand))

	if err != nil {
		log.Default().Printf("Error connecting to %s: %v", neoController, err)
		return err
	}

	reply := make([]byte, 1024)

	_, err = con.Read(reply)

	if err != nil {
		log.Default().Printf("Error connecting to %s: %v", neoController, err)
		return err
	}

	fmt.Println(string(reply))

	return nil
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
