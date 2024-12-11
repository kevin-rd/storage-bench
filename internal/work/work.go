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
	"math/rand"
	"time"
)

type Worker struct {
	id         int
	cli        client.IClient
	sequence   int
	objectSize int64
	data       []byte
}

func NewWorker(id int, cli client.IClient, objectSize int64) *Worker {
	data := make([]byte, objectSize)
	for i := range data {
		data[i] = byte(rand.Intn(256))
	}

	return &Worker{
		id:         id,
		cli:        cli,
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
		res.ReadCost = res.Cost - res.ConnectCost
	}()

	// get file object meta
	o, stat, err := w.cli.GetObject(ctx, bucketName, objectName, types.GetObjectOptions{})
	res.ConnectCost = time.Since(res.ReqTime)
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

	bucketName := fmt.Sprintf("%s-%d", bucketPrefix, w.id%5)
	objectName := fmt.Sprintf("%s-%03d%03d", objectPrefix, w.id, w.sequence)

	res = &statistics.TestResult{
		ChanId:  w.id,
		ID:      w.sequence,
		ReqTime: time.Now(),
	}
	w.sequence++
	defer func() {
		res.Cost = time.Since(res.ReqTime)
		res.ReadCost = res.Cost - res.ConnectCost
	}()

	txHash, err := w.cli.CreateObject(ctx, bucketName, objectName, reader, types.CreateObjectOptions{})
	if err != nil {
		return res, fmt.Errorf("unable to create object, %v", err)
	}

	ctxTxTimeout, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	_, err = w.cli.WaitForTx(ctxTxTimeout, txHash)
	res.ConnectCost = time.Since(res.ReqTime)
	if err != nil {
		return res, fmt.Errorf("unable to wait tx success, %v", err)
	}

	err = w.cli.PutObject(ctx, bucketName, objectName, reader2.Size(), reader2, types.PutObjectOptions{
		ContentType:      "application/octet-stream",
		DisableResumable: true,
	})
	if err != nil {
		return res, fmt.Errorf("unable to put object, %v", err)
	}

	timeout := time.After(1 * time.Hour)
	ticker := time.NewTicker(3 * time.Second)
	count := 0
	for {
		select {
		case <-timeout:
			return res, errors.New("object not sealed after one hour")
		case <-ticker.C:
			count++
			headObjOutput, queryErr := w.cli.HeadObject(ctx, bucketName, objectName)
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
