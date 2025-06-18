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
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

func eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		fmt.Println("Received a message from !", v.Info.Sender)

		fmt.Println("Received a message!", v.Message.GetConversation())
	}
}

func initClient() (*whatsmeow.Client, *sqlstore.Container, waLog.Logger) {
	dbLog := waLog.Stdout("Database", "DEBUG", true)
	// Make sure you add appropriate DB connector imports, e.g. github.com/mattn/go-sqlite3 for SQLite
	container, err := sqlstore.New(context.Background(), "sqlite3", "file:sqlite3.db?_foreign_keys=on", dbLog)
	if err != nil {
		panic(err)
	}

	// If you want multiple sessions, remember their JIDs and use .GetDevice(jid) or .GetAllDevices() instead.
	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		panic(err)
	}
	clientLog := waLog.Stdout("Client", "DEBUG", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)
	client.AddEventHandler(eventHandler)

	// handle login via QR
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
	return client, container, dbLog
}

func main() {
	var jidStr, header, text string
	var message *waE2E.Message
	var client *whatsmeow.Client
	var container *sqlstore.Container
	//var dbLog waLog.Logger
	cmd := &cli.Command{
		Usage: "Run WhatsApp actions from your CLI. User JID has to end with '@s.whatsapp.net', Group ID with '@g.us'",
		Commands: []*cli.Command{
			{
				Name:  "message",
				Usage: "send a message using <JID> <MESSAGE>",
				Arguments: []cli.Argument{
					&cli.StringArg{Name: "jid", Destination: &jidStr},
					&cli.StringArg{Name: "message", Destination: &text},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					// parse JID
					var JID types.JID
					JID, err := types.ParseJID(jidStr)
					if err != nil {
						log.Fatal("Failed to parse JID: ", jidStr)
					} else {
						log.Println("Parsed JID correctly", JID)
					}
					message = &waE2E.Message{Conversation: proto.String(text)}
					return nil
				},
			},
			{
				Name:  "getgroups",
				Usage: "print all available group info",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					fmt.Println(client.GetJoinedGroups())
					return nil
				},
			},
			{
				Name:  "poll",
				Usage: "send a poll using <JID> <HEADER> <OPTIONS>",
				Arguments: []cli.Argument{
					&cli.StringArg{Name: "jid", Destination: &jidStr}, // use id field of group
					&cli.StringArg{Name: "header", Destination: &header},
					&cli.StringArgs{Name: "options", Min: 2, Max: -1},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					// parse JID
					message = client.BuildPollCreation(header, cmd.StringArgs("options"), 1)
					return nil
				},
			},
		},
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			client, container, _ = initClient()
			return nil, nil
		},
		After: func(ctx context.Context, cmd *cli.Command) error {

			if message != nil {
				var JID types.JID
				JID, err := types.ParseJID(jidStr)
				if err != nil {
					log.Fatal("Failed to parse JID: ", jidStr)
				} else {
					log.Println("Parsed JID correctly", JID)
				}
				_, err = client.SendMessage(context.Background(), JID, message)
				// FIXME showing error even when successful
				if err != nil {
					log.Println("Sent Message successfully")
				} else {
					fmt.Println("Failed to Send Message", err)

				}
				time.Sleep(5 * time.Second)
			}
			// Listen to Ctrl+C (you can also do something else that prevents the program from exiting)
			// c := make(chan os.Signal, 1)
			// signal.Notify(c, os.Interrupt, syscall.SIGTERM)
			// <-c

			client.Disconnect()
			container.Close()
			return nil
		},
	}
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
