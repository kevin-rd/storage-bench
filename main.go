package main

import (
	"context"
	"github.com/kevin-rd/storage-bench/internal/statistics"
	"github.com/kevin-rd/storage-bench/internal/work"
	"github.com/zkMeLabs/mechain-go-sdk/client"
	"github.com/zkMeLabs/mechain-go-sdk/types"
	"log"
	"sync"
	"time"
)

const (
	// Testnet Info
	chainId    = "mechain_5151-1"
	rpcAddr    = "https://testnet-lcd.mechain.tech:443"
	evmRpcAddr = "https://testnet-rpc.mechain.tech"

	concurrency  = 20 // 并发数
	testDuration = 600 * time.Second

	privateKey = "c4eea90f78503630cee2c737d010a198fec1ab5ccf6fbb74d4f7f129cf42dacd"
	bucketName = "b-b837-kevin"
	objectName = "o-b837-1m-b"
	objectSize = 1 * 1024 * 1024
)

func main() {
	log.Printf("Starting...")
	log.Printf("concurrency: %d  testDuration: %.1fs objectName: %s", concurrency, testDuration.Seconds(), objectName)

	// import account
	account, err := types.NewAccountFromPrivateKey("file_test", privateKey)
	if err != nil {
		log.Fatalf("New account from private key error, %v", err)
	}

	// create client
	cli, err := client.New(chainId, rpcAddr, evmRpcAddr, privateKey, client.Option{DefaultAccount: account})
	if err != nil {
		log.Fatalf("unable to new zkMe Chain client, %v", err)
	}

	works := make([]*work.Worker, concurrency)
	for i := 0; i < concurrency; i++ {
		works[i] = work.NewWorker(i, cli, objectSize)
	}

	// begin
	var wg sync.WaitGroup
	var wgReceiver sync.WaitGroup
	ch := make(chan *statistics.TestResult, concurrency)

	// statistics
	wgReceiver.Add(1)
	go func() {
		defer wgReceiver.Done()
		statistics.HandleStatics(concurrency, ch)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), testDuration)
	defer cancel()
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(index int, ch chan<- *statistics.TestResult) {
			defer wg.Done()

			for {
				if ctx.Err() != nil {
					return
				}
				// download file
				// res, _ := works[index].GetObject(ctx, bucketName, objectName)

				// upload file
				res, err := works[index].PutObject(ctx, bucketName, objectName)
				if err != nil {
					log.Printf("worker %d put object error, %v", index, err)
				}

				ch <- res
			}
		}(i, ch)
		// Slow start
		time.Sleep(time.Duration(i%10) * (testDuration / 10 / concurrency))
	}
	wg.Wait()
	close(ch)
	wgReceiver.Wait()
}
