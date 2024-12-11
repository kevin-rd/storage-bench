package statistics

import (
	"fmt"
	"time"
)

type TestResult struct {
	ID      int           // id
	ChanId  int           // chain id
	ReqTime time.Time     // request time
	Cost    time.Duration // total cost
	Step1   time.Duration // step1 cost
	Step2   time.Duration // step2 cost
	Success bool          // success
}

func (tr *TestResult) String() string {
	return fmt.Sprintf("Id:%d ReqTime:%s Success:%v Cost:%.3fs & %.3fs => %.3fs", tr.ID, tr.ReqTime.Format("04:05.000"), tr.Success,
		tr.Step1.Seconds(), tr.Step2.Seconds(), tr.Cost.Seconds())
}
