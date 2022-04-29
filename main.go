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
}

type contextKey string

func _init() *Config {
	config := &Config{}

	flag.StringVar(&config.Prefix, "prefix", "0x0000", "ERC20 wallet address prefix")
	flag.StringVar(&config.Suffix, "suffix", "", "ERC20 wallet address suffix")
	flag.IntVar(&config.Concurrent, "concurrent", MaxParallelism()-1, "number of goroutine")

	flag.Parse()
	return config
}

var count int64

func main() {
	config := _init()

	cancelCtx, cancel := context.WithCancel(context.Background())

	finishCh := make(chan int, config.Concurrent)
	defer close(finishCh)

	wg := &sync.WaitGroup{}

	for i := 0; i < config.Concurrent; i++ {
		wg.Add(1)
		subCtx := context.WithValue(cancelCtx, contextKey("config"), config)
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
				fmt.Printf(
					"%d wallets generated, speed: %.f/s, time elapsed %.fs\n",
					count,
					float64(count)/(time.Since(startTime).Seconds()),
					time.Since(startTime).Seconds(),
				)
			}
		}
	}()

	wg.Wait()
	fmt.Println("main exited")
}

func genWorker(ctx context.Context, finish chan int, index int, wg *sync.WaitGroup) {
	fmt.Printf("[worker %d] start\n", index)

	config := ctx.Value(contextKey("config")).(*Config)
	for {
		select {
		case <-ctx.Done():
			wg.Done()
			fmt.Printf("[worker %d] exited\n", index)
			return
		default:
			if genWallet(config) {
				fmt.Printf("[worker %d] success\n", index)
				finish <- 1
			}
		}
	}
}

func genWallet(config *Config) bool {
	privateKey, _ := crypto.GenerateKey()
	publicKey := privateKey.Public()
	publicKeyECDSA, _ := publicKey.(*ecdsa.PublicKey)
	address := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()

	atomic.AddInt64(&count, 1)

	if strings.HasPrefix(address, config.Prefix) && strings.HasSuffix(address, config.Suffix) {
		fmt.Println("Address:", address)
		privateKeyBytes := crypto.FromECDSA(privateKey)
		fmt.Println("SAVE BUT DO NOT SHARE THIS (Private Key):", hexutil.Encode(privateKeyBytes))
		return true
	}

	return false
}

func MaxParallelism() int {
	maxProcs := runtime.GOMAXPROCS(0)
	numCPU := runtime.NumCPU()
	if maxProcs < numCPU {
		return maxProcs
	}
	return numCPU
}
