package work

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	types2 "github.com/evmos/evmos/v12/x/storage/types"
	"github.com/kevin-rd/storage-bench/internal/statistics"
	"github.com/zkMeLabs/mechain-go-sdk/client"
	"github.com/zkMeLabs/mechain-go-sdk/types"
	"io"
	"log"
	"math/rand"
	"strings"
	"time"
)

const (
	// Testnet Info
	chainId    = "mechain_5151-1"
	rpcAddr    = "https://testnet-lcd.mechain.tech:443"
	evmRpcAddr = "https://testnet-rpc.mechain.tech"
)

type Worker struct {
	id       int
	cli      client.IClient
	sequence int

	account    *types.Account
	objectSize int64
	data       []byte
	bucketName string
}

func NewWorker(id int, objectSize int64, privateKey string) *Worker {
	data := make([]byte, objectSize)
	for i := range data {
		data[i] = byte(rand.Intn(256))
	}

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

	return &Worker{
		id:         id,
		cli:        cli,
		account:    account,
		objectSize: objectSize,
		data:       data,
	}
}

func (w *Worker) GetObject(ctx context.Context, bucketName, objectName string) (res *statistics.TestResult, err error) {
	res = &statistics.TestResult{
		ChanId:  w.id,
		ID:      w.sequence,
		ReqTime: time.Now(),
	}
	w.sequence++
	defer func() {
		res.Cost = time.Since(res.ReqTime)
	}()

	// get file object meta
	o, stat, err := w.cli.GetObject(ctx, bucketName, objectName, types.GetObjectOptions{})
	res.Step1 = time.Since(res.ReqTime)
	if err != nil {
		return res, fmt.Errorf("unable to get object, %v", err)
	}
	defer func() {
		_ = o.Close()
	}()

	// read file object content
	buf := make([]byte, stat.Size)
	for {
		_, err = o.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return res, fmt.Errorf("unable to read object, %v", err)
		}
	}
	res.Success = true
	return
}

func (w *Worker) PutObject(ctx context.Context, bucketPrefix, objectPrefix string) (res *statistics.TestResult, err error) {
	reader := bytes.NewReader(w.data)
	reader2 := bytes.NewReader(w.data)

	objectName := genObjectName(w.account.GetAddress().String(), w.id, w.sequence)

	res = &statistics.TestResult{
		ChanId:  w.id,
		ID:      w.sequence,
		ReqTime: time.Now(),
	}
	w.sequence++
	defer func() {
		res.Cost = time.Since(res.ReqTime)
	}()

	txHash, err := w.cli.CreateObject(ctx, w.bucketName, objectName, reader, types.CreateObjectOptions{})
	if err != nil {
		return res, fmt.Errorf("unable to create object, %v", err)
	}

	ctxTxTimeout, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	_, err = w.cli.WaitForTx(ctxTxTimeout, txHash)
	res.Step1 = time.Since(res.ReqTime)
	if err != nil {
		return res, fmt.Errorf("unable to wait tx success, %v", err)
	}

	err = w.cli.PutObject(ctx, w.bucketName, objectName, reader2.Size(), reader2, types.PutObjectOptions{
		ContentType:      "application/octet-stream",
		DisableResumable: true,
	})
	res.Step2 = time.Since(res.ReqTime) - res.Step1
	if err != nil {
		return res, fmt.Errorf("unable to put object, %v", err)
	}

	ticker := time.NewTicker(1 * time.Second)
	count := 0
	for {
		select {
		case <-ctx.Done():
			return res, errors.New("object was not sealed within the specified time. ")
		case <-ticker.C:
			count++
			headObjOutput, queryErr := w.cli.HeadObject(ctx, w.bucketName, objectName)
			if queryErr != nil {
				return res, fmt.Errorf("unable to query object status, %v", queryErr)
			}
			if headObjOutput.ObjectInfo.GetObjectStatus() == types2.OBJECT_STATUS_SEALED {
				ticker.Stop()
				res.Success = true
				return
			}
		}
	}
}

func (w *Worker) InitPut(spAddr string) error {
	bucketName := genBucketName(w.account.GetAddress().String())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	bucketInfo, err := w.cli.HeadBucket(ctx, bucketName)
	if err == nil && bucketInfo != nil {
		w.bucketName = bucketName
		return nil
	}

	log.Printf("bucket %s not found, create new bucket", bucketName)

	opts := types.CreateBucketOptions{
		Visibility:   types2.VISIBILITY_TYPE_PUBLIC_READ,
		ChargedQuota: 1000000000000,
	}
	_, err = w.cli.CreateBucket(ctx, bucketName, spAddr, opts)
	if err != nil {
		return err
	}
	w.bucketName = bucketName
	return nil
}

func genBucketName(addr string) string {
	return fmt.Sprintf("b-%s-upload", strings.ToLower(strings.TrimPrefix(addr, "0x")[0:4]))
}

func genObjectName(addr string, a, b int) string {
	return fmt.Sprintf("o-%s-%s-%03d%03d", strings.ToLower(strings.TrimPrefix(addr, "0x")[0:4]), "ac", a, b)
}
