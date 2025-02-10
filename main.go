// ThingsDB module for Firebase.
//
// For example:
//

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	timod "github.com/thingsdb/go-timod"
	"github.com/vmihailenco/msgpack"
	"google.golang.org/api/option"
)

var mux sync.Mutex
var app *firebase.App
var client *messaging.Client

type firebaseConf struct {
	Credentials map[string]string `msgpack:"credentials"`
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
	mux.Lock()
	defer mux.Unlock()

	if conf.Credentials == nil {
		return fmt.Errorf("firebase credentials must not be empty")
	}

	json, err := json.Marshal(conf.Credentials)
	if err != nil {
		return err
	}

	opt := option.WithCredentialsJSON(json)

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

func handleSendMessage(pkg *timod.Pkg) {
	var req message
	err := msgpack.Unmarshal(pkg.Data, &req)
	if err != nil {
		timod.WriteEx(
			pkg.Pid,
			timod.ExBadData,
			fmt.Sprintf("Failed to unpack Firebase request (%s)", err))
		return
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
	resp, err := client.Send(context.Background(), message)
	if err != nil {
		timod.WriteEx(
			pkg.Pid,
			timod.ExBadData,
			fmt.Sprintf("Failed to send message (%s)", err))
		return
	}

	timod.WriteResponse(pkg.Pid, resp)
}

func handleSendMulticastMessage(pkg *timod.Pkg) {
	var req multicastMessage
	err := msgpack.Unmarshal(pkg.Data, &req)
	if err != nil {
		timod.WriteEx(
			pkg.Pid,
			timod.ExBadData,
			fmt.Sprintf("Failed to unpack Firebase request (%s)", err))
		return
	}

	message := &messaging.MulticastMessage{
		Notification: &messaging.Notification{
			Title: req.Title,
			Body:  req.Body,
		},
		Data:   req.Data,
		Tokens: req.Tokens,
	}

	br, err := client.SendEachForMulticast(context.Background(), message)
	if err != nil {
		timod.WriteEx(
			pkg.Pid,
			timod.ExBadData,
			fmt.Sprintf("Failed to send multicast message (%s)", err))
		return
	}

	timod.WriteResponse(pkg.Pid, br)
}

func onModuleReq(pkg *timod.Pkg) {
	// Not sure if thread safe
	mux.Lock()
	defer mux.Unlock()

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
		return
	}

	if req.Handler == nil {
		timod.WriteEx(
			pkg.Pid,
			timod.ExBadData,
			"Missing handler")
		return
	}

	if *req.Handler == "send-multicast-message" {
		handleSendMulticastMessage(pkg)
		return
	}

	if req.Handler == nil {
		timod.WriteEx(
			pkg.Pid,
			timod.ExBadData,
			"Missing handler")
		return
	}

	timod.WriteEx(
		pkg.Pid,
		timod.ExBadData,
		fmt.Sprintf("Unknown handler: %s", *req.Handler))

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
					log.Println("Missing or invalid Firebase configuration")
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
