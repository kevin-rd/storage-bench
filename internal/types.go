package internal

import "time"

type TestResult struct {
	ID        int           // id
	ChanId    int           // chain id
	Cost      time.Duration // 耗时
	Timestamp time.Time     // 请求时间
	Err       error
}
