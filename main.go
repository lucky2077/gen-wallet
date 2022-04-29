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
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

type Config struct {
	Prefix     string
	Suffix     string
	Concurrent int
	LogCount   int64
}

type contextKey string

func _init() *Config {
	config := &Config{}

	flag.StringVar(&config.Prefix, "prefix", "0x0000", "ERC20 wallet address prefix")
	flag.StringVar(&config.Suffix, "suffix", "", "ERC20 wallet address suffix")
	flag.IntVar(&config.Concurrent, "concurrent", MaxParallelism()-1, "number of goroutine")
	flag.Int64Var(&config.LogCount, "log-count", 100000, "print log per count")

	flag.Parse()
	return config
}

func main() {
	config := _init()

	cancelCtx, cancel := context.WithCancel(context.Background())

	finishCh := make(chan int)
	countCh := make(chan int)
	defer close(finishCh)

	for i := 0; i < config.Concurrent; i++ {
		subCtx := context.WithValue(cancelCtx, contextKey("config"), config)
		go genWorker(subCtx, finishCh, countCh, i)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	var count int64
	startTime := time.Now()

	for {
		select {
		case <-c:
			cancel()
			_exit()
		case <-finishCh:
			cancel()
			_exit()
		case <-countCh:
			count++
			if count%config.LogCount == 0 {
				fmt.Printf(
					"%d wallets generated, speed: %.f/s, time passed %.fs\n",
					count,
					float64(count)/(time.Since(startTime).Seconds()),
					time.Since(startTime).Seconds(),
				)
			}
		}
	}
}

func _exit() {
	fmt.Println("exiting")
	time.Sleep(time.Second * 3)
	fmt.Println("main exited")
	os.Exit(0)
}

func genWorker(ctx context.Context, finish chan int, countCh chan int, index int) {
	fmt.Printf("[worker %d] start\n", index)

	config := ctx.Value(contextKey("config")).(*Config)
	for {
		select {
		case <-ctx.Done():
			fmt.Printf("[worker %d] exited\n", index)
			return
		default:
			if genWallet(config) {
				finish <- 1
			}
			countCh <- 1
		}
	}
}

func genWallet(config *Config) bool {
	privateKey, _ := crypto.GenerateKey()
	publicKey := privateKey.Public()
	publicKeyECDSA, _ := publicKey.(*ecdsa.PublicKey)
	address := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()

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
