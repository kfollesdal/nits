package state

import "github.com/nats-io/nats.go"

func Init(conn *nats.Conn) error {
	js, err := conn.JetStream()
	if err != nil {
		return err
	}

	if err = InitKeyValueStores(js); err != nil {
		return err
	}

	if err = InitObjectStores(js); err != nil {
		return err
	}

	return InitStreams(js)
}