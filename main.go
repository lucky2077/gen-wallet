package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"net/http"
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
	flag.IntVar(&config.CountPerGeneration, "count-per-generation", 10, "number of wallet per genWallet()")
	flag.StringVar(&config.DiscordWebhook, "discord-webhook", "", "discord webhook url")
	flag.StringVar(&config.RSAPublicKey, "rsa-public-key", "", "RSA Public key")

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
	for i := 0; i < config.CountPerGeneration; i++ {
		privateKey, _ := crypto.GenerateKey()
		publicKey := privateKey.Public()
		publicKeyECDSA, _ := publicKey.(*ecdsa.PublicKey)
		address := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()

		atomic.AddInt64(&count, 1)

		if strings.HasPrefix(address, config.Prefix) && strings.HasSuffix(address, config.Suffix) {
			privateKeyString := hexutil.Encode(crypto.FromECDSA(privateKey))
			content, err := encryptWithRSAPublicKey(privateKeyString)
			if err != nil {
				printLog(DEBUG, "encryptWithRSAPublicKey failed: %s\n", err)
				continue
			}

			message := fmt.Sprintf("Wallet Address: %s\nPrivate Key: %s", address, content)

			if config.DiscordWebhook != "" {
				sendToDiscord(message)
			} else {
				printLog(INFO, "%s\n", message)
			}
			return true
		}
	}

	return false
}

func encryptWithRSAPublicKey(content string) (string, error) {
	if config.RSAPublicKey == "" {
		return content, nil
	}
	rsaPublicKey, err := base64.StdEncoding.DecodeString(config.RSAPublicKey)
	if err != nil {
		return "", err
	}

	block, _ := pem.Decode([]byte(rsaPublicKey))
	if block == nil {
		return "", errors.New("failed to parse PEM block containing the public key")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return "", err
	}

	encryptedBytes, err := rsa.EncryptOAEP(
		sha256.New(),
		rand.Reader,
		pub.(*rsa.PublicKey),
		[]byte(content),
		nil)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encryptedBytes), nil
}

func sendToDiscord(content string) {
	message := Message{content}
	tmp, _ := json.Marshal(message)
	body := bytes.NewBuffer(tmp)

	// Create client
	client := &http.Client{}

	// Create request
	req, _ := http.NewRequest("POST", config.DiscordWebhook, body)

	// Headers
	req.Header.Add("Content-Type", "application/json; charset=utf-8")

	// Fetch Request
	_, err := client.Do(req)

	if err != nil {
		fmt.Println("Failure : ", err)
	}
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
