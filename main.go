package main

import (
	"context"
	"fmt"
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
	rpcAddr    = "https://testnet-lcd.mechain.tech:443"
	evmRpcAddr = "https://testnet-rpc.mechain.tech"

	chainId = "mechain_5151-1"
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
	bucketName, objectName := "b-"+"2344"+"-kevin", "o-"+"2344"+"-10m"

	var wg sync.WaitGroup
	ch := make(chan time.Duration, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := getObject(cli, bucketName, objectName, ch); err != nil {
				log.Printf("get object error, %v", err)
				return
			}
		}()
	}
	wg.Wait()
	close(ch)

	var totalDuration time.Duration
	var count int
	for d := range ch {
		if d > 0 {
			totalDuration += d
			count++
		}
	}
	if count > 0 {
		fmt.Printf("average time: %s\n", totalDuration/time.Duration(count))
	}
}

func getObject(cli client.IClient, bucketName, objectName string, ch chan<- time.Duration) error {
	start := time.Now()
	defer func() {
		ch <- time.Since(start)
		log.Printf("get object %s cost %s", objectName, time.Since(start))
	}()

	ctx := context.TODO()
	o, stat, err := cli.GetObject(ctx, bucketName, objectName, types.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("unable to get object, %v", err)
	}
	log.Printf("get object %s successfully, stat: %+v", objectName, stat)
	defer o.Close()
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
