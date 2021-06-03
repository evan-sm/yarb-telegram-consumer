package main

import (
	"context"
	"encoding/json"
	"fmt"
	//	"io"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/k0kubun/pp"
	log "github.com/sirupsen/logrus"
	yarb "github.com/wmw9/yarb-struct"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func PullMsgsSync(projectID, subID string) error {
	ctx := context.Background()
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return fmt.Errorf("pubsub.NewClient: %v", err)
	}
	defer client.Close()

	sub := client.Subscription(subID)

	// Turn on synchronous mode. This makes the subscriber use the Pull RPC rather
	// than the StreamingPull RPC, which is useful for guaranteeing MaxOutstandingMessages,
	// the max number of messages the client will hold in memory at a time.
	sub.ReceiveSettings.Synchronous = true
	sub.ReceiveSettings.MaxOutstandingMessages = 1

	//	// Receive messages for 5 seconds.
	//	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	//	defer cancel()

	// Create a channel to handle messages to as they come in.
	cm := make(chan *pubsub.Message)
	defer close(cm)
	log.Infof("yarb-telegram-consumer started [*]")
	// Handle individual messages in a goroutine.
	go func() {
		for msg := range cm {
			fmt.Printf("Got message :%q\n", string(msg.Data))

			ok := make(chan bool)
			go func() {
				var p yarb.Payload
				if err := json.Unmarshal(msg.Data, &p); err != nil {
					log.Debugf("unmarshal error %v", err)
					ok <- false
				} else {
					pp.Println("go func():", p)

					if err := sendToTelegram(p); err != nil {
						log.Errorf("sendToTelegram error: %v", err)
						ok <- false
					} else {
						log.Infof("[OK] Telegram\n")
						// Update DB timestamp
						if err := UpdateIGStoriesTs(p); err != nil {
							log.Println(err)
						}
						ok <- true
					}
				}

			}()
			time.Sleep(20 * time.Second)
			if <-ok {
				msg.Ack()
			} else {
				msg.Nack()
			}
		}
	}()

	// Receive blocks until the passed in context is done.
	err = sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		cm <- msg
	})
	if err != nil && status.Code(err) != codes.Canceled {
		return fmt.Errorf("Receive: %v", err)
	}

	return nil
}
