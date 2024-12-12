package main

import (
	"context"
	"cosmossdk.io/math"
	evmtypes "github.com/evmos/evmos/v12/sdk/types"
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
	chainId       = "mechain_5151-1"
	rpcAddr       = "https://testnet-lcd.mechain.tech:443"
	evmAddr       = "https://testnet-rpc.mechain.tech"
	primarySpAddr = "0xCC8aC1b69D013E33E886Cc997CD0aEBc70798e5c"

	concurrency  = 2 // 并发数
	testDuration = 600 * time.Second

	privateKey = "27cb97c6b79b255a6558bf89d9e673e00febbb739b4741861a5654c140b37621"
	bucketName = "b-b837-kevin"
	objectName = "o-b837-1m-b"
	objectSize = 10 * 1024 * 1024
)

var accounts = []string{
	"dbf2999f925145213f7262580a7a3a0562426509746d1e10cd1e610198e679a0",
	"23c7159b2b8b02b1f45edc6069c1771784a2630358c9d0cdb82c41033b79f635",
	"b93f760c5524e6883d0019f03cb82797603b5b90870669e501e5296f79e156a6",
	"f4411f3e1323f7b6238f109510781d62d08a00b3041ff16c7cae0a9c4d111cae",
	"5be9f77d4c91b4acb422ece974547eea0721a72108f1e4027ac01eef02ba9439",
	"036a73c77f115dca418331145fd9bf0dce0ae6688419262b90af9cb780104323",
	"50e74ff5e31974911bbe3c3fe38d50ad0b0a9628fb7feef3ed02e57994f3381b",
	"1ddcb6f8a014aa74bc0a79a49214ca31d7a3dd6fea4067229b1b1437cc6f8a31",
	"bbd6b49607fa05345cb9befa2026332836c991bc8630e2c91bd34211b06443dc",
	"f68f2154aa0c832b335155bceb20860271865374f9c272941e43590993b97ee4",
	"9af210b337bb6eaa314ef6f5b06ce42c225cb74b87b33a75aae330fd83abde72",
	"29c16bfe979aa8f487dd3cab6279b5f68ee7c40a48a6c48cee74f1b90bc9978c",
	"c6fb0b4bf94b43299c350a6ea617d5ea7dc3e6b5c82789817463cbdd54e1a9c1",
	"a2d323c5ae2a42911ec69bca9f3497a6a0af627c3b335cbdf01471ab6eb1318f",
	"6d9967588e152a73eca90325c73ede3986597b9debcab44206acafb57cbbfc60",
	"5ef415eade3cd5835552845f8ab981701dd7141f53a3f8c1a58f5c648043d89d",
	"f0d4ef4f44f221cde7ccc1ced22af076e92d14a192881ade2e6cc2500cb722fe",
	"aad1ea6b21e6d6b775f9fcd815d115331880db14a2df2164191213dc853fd066",
	"c6ab4ad9a06b1fada520e3c9037f961d1e1f58c425c7cec5506dae5a907682e1",
	"b037aa92437290cec697f4614ac48f53db4f830d7a82a36e0f369b300372bd7a",
}

func main() {
	log.Printf("Starting...")
	log.Printf("concurrency: %d  testDuration: %.1fs objectName: %s", concurrency, testDuration.Seconds(), objectName)

	works := make([]*work.Worker, concurrency)
	for i := 0; i < concurrency; i++ {
		works[i] = work.NewWorker(i, objectSize, accounts[i%len(accounts)])
		if err := works[i].InitPut(primarySpAddr); err != nil {
			log.Fatalf("worker %d init error, %v", i, err)
		}
	}

	// begin
	var wg sync.WaitGroup
	var wgReceiver sync.WaitGroup
	ch := make(chan *statistics.TestResult, concurrency)

	// statistics
	wgReceiver.Add(1)
	go func() {
		defer wgReceiver.Done()
		log.Printf("statistics start...")
		statistics.HandleStatics(concurrency, ch)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), testDuration)
	defer cancel()
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		log.Printf("worker %d start...", i)
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
		if i%10 == 0 {
			time.Sleep(testDuration / 10 / concurrency)
		}
	}
	wg.Wait()
	close(ch)
	wgReceiver.Wait()
}

func main2() {
	// import account
	account, err := types.NewAccountFromPrivateKey("file_test", privateKey)
	if err != nil {
		log.Fatalf("New account from private key error, %v", err)
	}

	// create client
	cli, err := client.New(chainId, rpcAddr, evmAddr, privateKey, client.Option{DefaultAccount: account})
	if err != nil {
		log.Fatalf("unable to new zkMe Chain client, %v", err)
	}

	log.Println("account address list:")
	for i := 0; i < 10; i++ {
		account, private, err := types.NewAccount("mechain-account")
		if err != nil {
			log.Fatalf("New account error, %v", err)
		}

		txHash, err := cli.Transfer(context.TODO(), account.GetAddress().String(), math.NewIntWithDecimal(1, 18).MulRaw(10), evmtypes.TxOption{})
		if err != nil {
			log.Fatalf("transfer error, %v", err)
		}

		if _, err = cli.WaitForTx(context.TODO(), txHash); err != nil {
			log.Fatalf("wait tx error, %v", err)
		}

		log.Printf(private)
	}
}
