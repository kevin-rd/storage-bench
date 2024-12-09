package main

import (
	"context"
	"fmt"
	"github.com/kevin-rd/storage-bench/internal"
	"github.com/zkMeLabs/mechain-go-sdk/client"
	"github.com/zkMeLabs/mechain-go-sdk/types"
	"io"
	"log"
	"strings"
	"sync"
	"time"
)

const (
	privateKey = "27cb97c6b79b255a6558bf89d9e673e00febbb739b4741861a5654c140b37621"

	// Testnet Info
	chainId    = "mechain_5151-1"
	rpcAddr    = "https://testnet-lcd.mechain.tech:443"
	evmRpcAddr = "https://testnet-rpc.mechain.tech"

	concurrency  = 20 // 并发数
	testDuration = 60 * time.Second
)

func main() {
	// import account
	account, err := types.NewAccountFromPrivateKey("test", privateKey)
	if err != nil {
		log.Fatalf("New account from private key error, %v", err)
	}

	// create client
	cli, err := client.New(chainId, rpcAddr, evmRpcAddr, privateKey, client.Option{DefaultAccount: account})
	if err != nil {
		log.Fatalf("unable to new zkMe Chain client, %v", err)
	}

	// 2. Create a bucket
	_ = strings.TrimPrefix(account.GetAddress().String(), "0x")[0:4]
	bucketName, objectName := "b-"+"2344"+"-kevin", "o-"+"2344"+"-10k"

	var wg sync.WaitGroup
	var wgReceiver sync.WaitGroup
	ch := make(chan *internal.TestResult, concurrency)

	wgReceiver.Add(1)
	go func() {
		defer wgReceiver.Done()
		internal.HandleStatics(concurrency, ch)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), testDuration)
	defer cancel()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(index int, ch chan<- *internal.TestResult) {
			defer wg.Done()

			for j := 0; true; {
				if ctx.Err() != nil {
					return
				}
				r := &internal.TestResult{
					ID:        j*10000 + index,
					ChanId:    index,
					Timestamp: time.Now(),
				}
				err := getObject(ctx, cli, bucketName, objectName)
				r.Cost = time.Since(r.Timestamp)
				if err != nil {
					r.Err = err
				}
				ch <- r
			}
		}(i, ch)
	}
	wg.Wait()
	close(ch)
	wgReceiver.Wait()
}

func getObject(ctx context.Context, cli client.IClient, bucketName, objectName string) error {
	o, stat, err := cli.GetObject(ctx, bucketName, objectName, types.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("unable to get object, %v", err)
	}
	// log.Printf("get object %s successfully, stat: %+v", objectName, stat)
	defer func() {
		_ = o.Close()
	}()
	buf := make([]byte, stat.Size)
	for {
		_, err := o.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("unable to read object, %v", err)
		}
	}
	return nil
}
