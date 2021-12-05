package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/ethclient"
)

type syncCounter struct {
	sync.Mutex
	counter int64
}

var (
	ethUrl         = os.Getenv("GETH_URL")
	tgBotToken     = os.Getenv("BOT_TOKEN")
	reportInterval = mustParseDuration(os.Getenv("REPORT_INTERVAL"))
	checkInterval  = mustParseDuration(os.Getenv("CHECK_INTERVAL"))
	alertGroup     = mustParseInt64(os.Getenv("ALERT_GROUP"))
)

func init() {
	if reportInterval <= checkInterval {
		panic("report interval must be greater than check interval")
	}
}

func main() {
	c, err := createGethClient(ethUrl)
	if err != nil {
		log.Fatalf("error creating geth client: %s", err)
	}
	b, err := createTelegramBot(tgBotToken)
	if err != nil {
		log.Fatalf("error creating telegram bot: %s", err)
	}
	checkSyncing(c, b, alertGroup, checkInterval, reportInterval)
}

func mustParseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		panic(err)
	}
	return d
}

func mustParseInt64(s string) int64 {
	i, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
	return int64(i)
}

func createGethClient(url string) (*ethclient.Client, error) {
	return ethclient.Dial(url)
}

func createTelegramBot(token string) (*gotgbot.Bot, error) {
	b, err := gotgbot.NewBot(os.Getenv("BOT_TOKEN"), &gotgbot.BotOpts{
		Client:      http.Client{},
		GetTimeout:  gotgbot.DefaultGetTimeout,
		PostTimeout: gotgbot.DefaultPostTimeout,
	})
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (s *syncCounter) get() int64 {
	s.Lock()
	defer s.Unlock()
	return s.counter
}

func (s *syncCounter) increase() {
	s.Lock()
	defer s.Unlock()
	s.counter++
}

func (s *syncCounter) reset() {
	s.Lock()
	defer s.Unlock()
	s.counter = 0
}

func checkSyncing(c *ethclient.Client, b *gotgbot.Bot, alertGroup int64, checkInterval, reportInterval time.Duration) {
	checkTicker := time.NewTicker(checkInterval)
	defer checkTicker.Stop()
	reportTicker := time.NewTicker(reportInterval)
	defer reportTicker.Stop()

	counter := syncCounter{}
	var sync *ethereum.SyncProgress

	go func() {
		for range checkTicker.C {
			var err error
			sync, err = c.SyncProgress(context.Background())
			if err != nil {
				log.Printf("error while checking sync status: %s", err)
				continue
			}
			if sync == nil {
				counter.increase()
			}
		}
	}()

	var prevOutOfSynced bool
	for range reportTicker.C {
		if counter.get() > 0 && prevOutOfSynced {
			log.Println("node is back in sync")
			_, err := b.SendMessage(alertGroup, inSyncMsg(), nil)
			if err != nil {
				log.Printf("error sending message: %s", err)
			}
			prevOutOfSynced = false
		} else if counter.get() == 0 && !prevOutOfSynced {
			log.Printf("node is out of sync: current block %d, highest block %d", sync.CurrentBlock, sync.HighestBlock)
			_, err := b.SendMessage(alertGroup, outOfSyncMsg(sync, reportInterval), nil)
			if err != nil {
				log.Printf("error sending message: %s", err)
			}
			prevOutOfSynced = true
		}
		counter.reset()
	}
}

func outOfSyncMsg(sync *ethereum.SyncProgress, r time.Duration) string {
	var s strings.Builder
	s.WriteString(fmt.Sprintf("🔴 your node is out of sync since %s\n", r))
	s.WriteString(fmt.Sprintf("Current block: %d\n", sync.CurrentBlock))
	s.WriteString(fmt.Sprintf("Highest block: %d\n", sync.HighestBlock))
	return s.String()
}

func inSyncMsg() string {
	return "🟢 your node is back in sync"
}
