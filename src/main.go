// from https://github.com/tulir/whatsmeow/issues/659
package main

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/shlex"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal/v3"
	"github.com/urfave/cli/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

func initClient() (*whatsmeow.Client, *sqlstore.Container, waLog.Logger) {
	//TODO use stderr here
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
	client := whatsmeow.NewClient(deviceStore, nil)

	// handle login via QR
	if client.Store.ID == nil {
		slog.Info("Client store ID is nil, scanning QR")
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
				fmt.Println("QR code: ", evt.Code)
			} else {
				slog.Debug("Login event", "event", evt.Event)
			}
		}
	} else {
		// Already logged in, just connect
		slog.Info("Connecting")

		err = client.Connect()
		if err != nil {
			panic(err)
		}
	}
	return client, container, dbLog
}

func sendMessage(message *waE2E.Message, jidStr string, client *whatsmeow.Client) {
	if message != nil {
		var JID types.JID
		JID, err := types.ParseJID(jidStr)
		if err != nil {
			slog.Error("Failed to parse", "JID", jidStr)
		} else {
			slog.Info("Parsed correctly", "JID", JID)
		}
		_, err = client.SendMessage(context.Background(), JID, message)
		// FIXME showing error even when successful
		if err != nil {
			slog.Error("Failed to send message", "error", err)
		} else {
			slog.Info("Sent Message successfully")

		}
	}
}
func main() {
	var jidStr, header, text string
	var message *waE2E.Message
	var client *whatsmeow.Client
	var container *sqlstore.Container
	//var dbLog waLog.Logger
	cmd := &cli.Command{
		Usage: "Run WhatsApp actions from your CLI. User JID has to end with '@s.whatsapp.net', Group ID with '@g.us'." +
			"Defaults to listening on stdin for batch processing.",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			fmt.Println("No command specified. Reading from stdin. Press Ctrl+D to exit or run with --help to get help.")
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				if err := scanner.Err(); err != nil {
					slog.Error("Error reading", "error", err)
					continue
				}
				args, err := shlex.Split(scanner.Text())
				if err != nil || len(args) == 0 {
					slog.Error("Error parsing command", "command", err)
					continue
				}
				subcmd := cmd.Command(args[0])
				if subcmd != nil {
					subcmd.Run(ctx, args)
				} else {
					slog.Error("Unknown command", "command", args[0])
				}
			}
			fmt.Println("Stdin closed.")
			return nil
		},
		Commands: []*cli.Command{
			{
				Name:  "message",
				Usage: "send a message using <JID> <MESSAGE>",
				Arguments: []cli.Argument{
					&cli.StringArg{Name: "jid", Destination: &jidStr},
					&cli.StringArg{Name: "message", Destination: &text},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					message = &waE2E.Message{Conversation: proto.String(text)}
					sendMessage(message, jidStr, client)
					return nil
				},
			},
			{
				Name:  "getgroups",
				Usage: "print all available group info",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					groups, err := client.GetJoinedGroups(ctx)
					if err != nil {
						slog.Error("Failed to fetch group info", "error", err)
					} else {
						// log a primitve csv to stdOut
						fmt.Printf("%q, %q, %q\n", "jid", "name", "parentJid")
						for _, item := range groups {
							fmt.Printf("%q, \"%s\", %q\n", item.JID, item.GroupName.Name, item.GroupLinkedParent.LinkedParentJID)
						}
					}
					return nil
				},
			},
			{
				Name:  "poll",
				Usage: "send a poll to a group using <JID> <HEADER> <OPTIONS>",
				Arguments: []cli.Argument{
					&cli.StringArg{Name: "jid", Destination: &jidStr},
					&cli.StringArg{Name: "header", Destination: &header},
					&cli.StringArgs{Name: "options", Min: 2, Max: -1},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					// parse JID
					message = client.BuildPollCreation(header, cmd.StringArgs("options"), 1)
					sendMessage(message, jidStr, client)
					return nil
				},
			},
			{
				// send an image using https://pkg.go.dev/go.mau.fi/whatsmeow#Client.Upload
				Name:  "image",
				Usage: "send an image using <JID> <PATH> <CAPTION>",
				Arguments: []cli.Argument{
					&cli.StringArg{Name: "jid", Destination: &jidStr},
					&cli.StringArg{Name: "path"},
					&cli.StringArg{Name: "caption"},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					data, err := os.ReadFile(cmd.StringArg("path"))
					resp, err := client.Upload(ctx, data, whatsmeow.MediaImage)
					// figure out the mimetype to specify
					// https://stackoverflow.com/questions/51209439/mime-type-checking-of-files-uploaded-golang
					if err != nil {
						slog.Error("Failed to open file", "error", err)
					}
					imageMsg := &waE2E.ImageMessage{
						Caption: proto.String(cmd.StringArg("caption")),
						// TODO replace this with the actual mime type
						Mimetype: proto.String("image/jpeg"),
						// you can also optionally add other fields like ContextInfo and JpegThumbnail here

						URL:           &resp.URL,
						DirectPath:    &resp.DirectPath,
						MediaKey:      resp.MediaKey,
						FileEncSHA256: resp.FileEncSHA256,
						FileSHA256:    resp.FileSHA256,
						FileLength:    &resp.FileLength,
					}
					sendMessage(&waE2E.Message{ImageMessage: imageMsg}, jidStr, client)
					return nil
				},
			},
		},
	}
	// before
	client, container, _ = initClient()
	// run
	cmd.Run(context.Background(), os.Args)
	// after
	time.Sleep(5 * time.Second)
	client.Disconnect()
	container.Close()
}
