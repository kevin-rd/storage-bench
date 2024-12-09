package internal

import (
	"fmt"
	"time"
)

type TestResult struct {
	ID          int           // id
	ChanId      int           // chain id
	ReqTime     time.Time     // request time
	Cost        time.Duration // total cost
	ConnectCost time.Duration // connect cost
	ReadCost    time.Duration // read file cost
	Success     bool          // success
}

func (tr *TestResult) String() string {
	return fmt.Sprintf("Id:%d ReqTime:%s Success:%v Cost:%.3fs+%.3fs=%.3fs", tr.ID, tr.ReqTime.Format("04:05.000"), tr.Success,
		tr.ConnectCost.Seconds(), tr.ReadCost.Seconds(), tr.Cost.Seconds())
}
