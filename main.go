// from https://github.com/tulir/whatsmeow/issues/659
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal/v3"
	"github.com/urfave/cli/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
)

func eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		fmt.Println("Received a message from !", v.Info.Sender)

		fmt.Println("Received a message!", v.Message.GetConversation())
	}
}

func sendPoll(JID types.JID, headline string, optionNames []string) {
	dbLog := waLog.Stdout("Database", "DEBUG", true)
	// Make sure you add appropriate DB connector imports, e.g. github.com/mattn/go-sqlite3 for SQLite
	container, err := sqlstore.New("sqlite3", "file:sqlite3.db?_foreign_keys=on", dbLog)
	if err != nil {
		panic(err)
	}

	// If you want multiple sessions, remember their JIDs and use .GetDevice(jid) or .GetAllDevices() instead.
	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		panic(err)
	}
	clientLog := waLog.Stdout("Client", "DEBUG", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)
	client.AddEventHandler(eventHandler)

	if client.Store.ID == nil {
		fmt.Println("Client store ID is nil, scanning QR")
		// No ID stored, new login
		qrChan, _ := client.GetQRChannel(context.Background())
		err = client.Connect()
		if err != nil {
			panic(err)
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				// Render the QR code here
				// e.g. qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				// or just manually `echo 2@... | qrencode -t ansiutf8` in a terminal
				fmt.Println("QR code:", evt.Code)
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else {
		// Already logged in, just connect
		fmt.Println("Connecting")

		err = client.Connect()
		if err != nil {
			panic(err)
		}
	}

	pollMessage := client.BuildPollCreation(headline, optionNames, 1)

	fmt.Println("Created Poll Message succuessfully  : ", pollMessage)

	_, err = client.SendMessage(context.Background(), JID, pollMessage)
	if err != nil {
		fmt.Println("Sent Poll Succuessfully ", JID)
	} else {
		fmt.Println("Failed to Send Poll", JID)

	}

	time.Sleep(5 * time.Second)
	// Listen to Ctrl+C (you can also do something else that prevents the program from exiting)
	// c := make(chan os.Signal, 1)
	// signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	// <-c

	client.Disconnect()
	container.Close()
}

func main() {
	cmd := &cli.Command{
		Action: func(ctx context.Context, cmd *cli.Command) error {
			// parse JID
			strJID := "XXXXXXXX@g.us"
			var JID types.JID
			JID, err := types.ParseJID(strJID)
			if err != nil {
				fmt.Println("Failed to parse JID: ", strJID)
			} else {
				fmt.Println("Parsed JID correctly", JID)

			}
			var optionNames []string
			optionNames = append(optionNames, "O1")
			optionNames = append(optionNames, "O2")
			headline := "Test"
			sendPoll(JID, headline, optionNames)
			return nil
		},
	}
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
