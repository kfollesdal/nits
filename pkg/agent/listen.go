package agent

import (
	"context"
	"encoding/json"
	"time"

	"github.com/juju/errors"
	"github.com/numtide/nits/pkg/subject"

	"github.com/charmbracelet/log"
	"github.com/numtide/nits/pkg/types"

	"github.com/nats-io/nats.go"
)

func listenForDeployment(ctx context.Context) (err error) {
	var js nats.JetStreamContext
	var sub *nats.Subscription
	var msg *nats.Msg

	if js, err = Conn.JetStream(); err != nil {
		return
	} else if sub, err = js.SubscribeSync(
		subject.AgentDeploymentWithNKey(NKey),
		nats.DeliverLastPerSubject(),
	); err != nil {
		return
	}

	for {
		select {
		case <-ctx.Done():
			return sub.Unsubscribe()
		default:
			if msg, err = sub.NextMsg(1 * time.Second); err != nil {
				if !errors.Is(err, nats.ErrTimeout) {
					Log.Error("failed to retrieve next deployment msg", "error", err)
				}
				continue
			} else if msg == nil {
				continue
			}

			// we go ahead and ack the message because we don't want re-delivery in case of failure
			// instead a user must evaluate why it failed and publish a new deployment
			if err = msg.Ack(); err != nil {
				Log.Error("failed to ack deployment", "error", err)
				continue
			}

			var config types.Deployment
			if err = json.Unmarshal(msg.Data, &config); err != nil {
				Log.Error("failed to unmarshal deployment", "error", err)
				continue
			}

			startedAt := time.Now()

			Log.Info("new deployment", "action", config.Action, "closure", config.Closure)

			err = deployHandler.Apply(&config, log.WithContext(ctx, Log))
			elapsed := time.Now().Sub(startedAt)

			if err != nil {
				Log.Error("deployment failed", "error", err, "elapsed", elapsed)
			} else {
				Log.Info("deployment complete", "elapsed", elapsed)
			}
		}
	}
}
