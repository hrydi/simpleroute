package signal

import (
	"context"
	"log"
	"os"
	"os/signal"
)

const (
	sigChSize = 32
)

func HandleSignals(sigs ...os.Signal) <- chan os.Signal {
	ch := make(chan os.Signal, sigChSize)
	go func() {
		sigCh := make(chan os.Signal, sigChSize)
		signal.Notify(sigCh, sigs...)
		for {
			sig := <-sigCh
			log.Printf("received signal %s", sig.String())
			ch <- sig
		}
	}()

	return ch
}

func CreateContext(ch <-chan os.Signal) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	go func ()  {
		<-ch
		cancel()
	}()

	return ctx, cancel
}