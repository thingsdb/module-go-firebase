// ThingsDB module for Firebase.
//
// For example:
//

package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/messaging"
	timod "github.com/thingsdb/go-timod"
	"github.com/vmihailenco/msgpack"
	"google.golang.org/api/option"
)

var mux sync.Mutex
var app *firebase.App
var client *messaging.Client

type firebaseConf struct {
	Credentials []byte `msgpack:"credentials"`
}

type firebaseReq struct {
	Handler *string `msgpack:"handler"`
}

type message struct {
	Body  string            `msgpack:"body"`
	Data  map[string]string `msgpack:"data"`
	Title string            `msgpack:"title"`
	Token string            `msgpack:"token"`
}

type multicastMessage struct {
	Body   string            `msgpack:"body"`
	Data   map[string]string `msgpack:"data"`
	Title  string            `msgpack:"title"`
	Tokens []string          `msgpack:"tokens"`
}

func handleConf(conf *firebaseConf) error {
	if conf.Credentials == nil {
		return fmt.Errorf("Firebase credentials must not be empty")
	}

	opt := option.WithCredentialsJSON(conf.Credentials)

	var err error
	ctx := context.Background()

	//Firebase admin SDK initialization
	app, err = firebase.NewApp(ctx, nil, opt)
	if err != nil {
		return err
	}

	//Messaging client
	client, err = app.Messaging(ctx)
	if err != nil {
		return err
	}

	return nil
}

func handleSendMessage(pkg *timod.Pkg) error {
	var req message
	err := msgpack.Unmarshal(pkg.Data, &req)
	if err != nil {
		return fmt.Errorf("Failed to unpack Firebase request (%s)", err)
	}

	message := &messaging.Message{
		Notification: &messaging.Notification{
			Title: req.Title,
			Body:  req.Body,
		},
		Data:  req.Data,
		Token: req.Token,
	}

	// Send a message to the device corresponding to the provided
	// registration token.
	_, err = client.Send(context.Background(), message)
	if err != nil {
		return err
	}

	return nil
}

func handleSendMulticastMessage(pkg *timod.Pkg) error {
	var req multicastMessage
	err := msgpack.Unmarshal(pkg.Data, &req)
	if err != nil {
		return fmt.Errorf("Failed to unpack Firebase request (%s)", err)
	}

	message := &messaging.MulticastMessage{
		Notification: &messaging.Notification{
			Title: req.Title,
			Body:  req.Body,
		},
		Data:   req.Data,
		Tokens: req.Tokens,
	}

	_, err = client.SendMulticast(context.Background(), message)
	if err != nil {
		return err
	}

	return nil
}

func onModuleReq(pkg *timod.Pkg) {
	var req firebaseReq
	err := msgpack.Unmarshal(pkg.Data, &req)
	if err != nil {
		timod.WriteEx(
			pkg.Pid,
			timod.ExBadData,
			"Failed to unpack Firebase request")
		return
	}

	if *req.Handler == "send-message" {
		handleSendMessage(pkg)
	} else if req.Handler == nil {
		timod.WriteEx(
			pkg.Pid,
			timod.ExBadData,
			"Missing handler")
	} else if *req.Handler == "send-multicast-message" {
		handleSendMulticastMessage(pkg)
	} else if req.Handler == nil {
		timod.WriteEx(
			pkg.Pid,
			timod.ExBadData,
			"Missing handler")
	} else {
		timod.WriteEx(
			pkg.Pid,
			timod.ExBadData,
			fmt.Sprintf("Unknown handler: %s", *req.Handler))
	}
}

func handler(buf *timod.Buffer, quit chan bool) {
	for {
		select {
		case pkg := <-buf.PkgCh:
			switch timod.Proto(pkg.Tp) {
			case timod.ProtoModuleConf:
				var conf firebaseConf

				err := msgpack.Unmarshal(pkg.Data, &conf)
				if err != nil {
					log.Println("Missing or invalid SMTP configuration")
					timod.WriteConfErr()
					break
				}

				err = handleConf(&conf)
				if err != nil {
					log.Println(err.Error())
					timod.WriteConfErr()
					break
				}

				timod.WriteConfOk()

			case timod.ProtoModuleReq:
				onModuleReq(pkg)

			default:
				log.Printf("Unexpected package type: %d", pkg.Tp)
			}
		case err := <-buf.ErrCh:
			// In case of an error you probably want to quit the module.
			// ThingsDB will try to restart the module a few times if this
			// happens.
			log.Printf("Error: %s", err)
			quit <- true
		}
	}
}

func main() {
	// Starts the module
	timod.StartModule("firebase", handler)
}
