package main

import (
	"context"
	"crypto/ecdsa"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

type Config struct {
	Prefix     string
	Suffix     string
	Concurrent int
	LOG_LEVEL  int
}

const (
	DEBUG = iota
	INFO
)

type contextKey string

var count int64
var config *Config

func _init() {
	config = &Config{}

	flag.StringVar(&config.Prefix, "prefix", "0x0000", "ERC20 wallet address prefix")
	flag.StringVar(&config.Suffix, "suffix", "", "ERC20 wallet address suffix")
	flag.IntVar(&config.Concurrent, "concurrent", maxParallelism()-1, "number of goroutine")
	flag.IntVar(&config.LOG_LEVEL, "log-level", INFO, "log level")

	flag.Parse()
}

func main() {
	_init()

	cancelCtx, cancel := context.WithCancel(context.Background())

	finishCh := make(chan int, config.Concurrent)
	defer close(finishCh)

	wg := &sync.WaitGroup{}

	for i := 0; i < config.Concurrent; i++ {
		wg.Add(1)
		subCtx := context.WithValue(cancelCtx, contextKey("dummy"), "dummy")
		go genWorker(subCtx, finishCh, i, wg)
	}

	startTime := time.Now()

	c := make(chan os.Signal, 1)
	defer close(c)
	signal.Notify(c, os.Interrupt)

	tick := time.NewTicker(time.Second)
	defer tick.Stop()

	go func() {
		for {
			select {
			case <-c:
				cancel()
				return
			case <-finishCh:
				cancel()
				return
			case <-tick.C:
				printLog(
					INFO,
					"\r%d wallets generated, speed: %.f/s, time elapsed %.fs",
					count,
					float64(count)/(time.Since(startTime).Seconds()),
					time.Since(startTime).Seconds(),
				)
			}
		}
	}()

	wg.Wait()
	printLog(DEBUG, "main exited")
}

func genWorker(ctx context.Context, finishCh chan int, index int, wg *sync.WaitGroup) {
	printLog(DEBUG, "[worker %d] start\n", index)

	for {
		select {
		case <-ctx.Done():
			wg.Done()
			printLog(DEBUG, "[worker %d] exit\n", index)
			return
		default:
			if genWallet() {
				finishCh <- 1
			}
		}
	}
}

func genWallet() bool {
	privateKey, _ := crypto.GenerateKey()
	publicKey := privateKey.Public()
	publicKeyECDSA, _ := publicKey.(*ecdsa.PublicKey)
	address := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()

	atomic.AddInt64(&count, 1)

	if strings.HasPrefix(address, config.Prefix) && strings.HasSuffix(address, config.Suffix) {
		printLog(INFO, "\nAddress: %s\n", address)
		privateKeyBytes := crypto.FromECDSA(privateKey)
		printLog(INFO, "SAVE BUT DO NOT SHARE THIS (Private Key): %s\n", hexutil.Encode(privateKeyBytes))
		return true
	}

	return false
}

func maxParallelism() int {
	maxProcs := runtime.GOMAXPROCS(0)
	numCPU := runtime.NumCPU()
	if maxProcs < numCPU {
		return maxProcs
	}
	return numCPU
}

func printLog(level int, format string, a ...any) {
	if config.LOG_LEVEL <= level {
		fmt.Printf(format, a...)
	}
}
