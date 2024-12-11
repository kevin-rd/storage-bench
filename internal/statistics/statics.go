package statistics

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

func HandleStatics(concurrency uint64, ch <-chan *TestResult) {
	var (
		costTimeList    []time.Duration                  // 耗时数组
		processingTime  time.Duration   = 0              // processingTime 处理总耗时
		requestCostTime time.Duration   = 0              // requestCostTime 请求总时间
		maxTime         time.Duration   = 0              // maxTime 至今为止单个请求最大耗时
		minTime         time.Duration   = 24 * time.Hour // minTime 至今为止单个请求最小耗时
		successNum      uint64          = 0
		failureNum      uint64          = 0
		chanIdLen       uint64          = 0 // chanIdLen 并发数
		stopChan                        = make(chan bool)
		mutex                           = sync.RWMutex{}
		chanIds                         = make(map[int]bool)
	)

	startTime := time.Now()
	respCodeMap := sync.Map{}
	ticker := time.NewTicker(time.Second)

	go func() {
		for {
			select {
			case <-ticker.C:
				curTime := time.Now()
				mutex.Lock()
				go calculateData(concurrency, processingTime, curTime.Sub(startTime), maxTime, minTime, successNum, failureNum, chanIdLen, &respCodeMap)
				mutex.Unlock()
			case <-stopChan:
				return
			}
		}
	}()

	printHeader()
	for respRes := range ch {
		mutex.Lock()

		// total process time
		processingTime = processingTime + respRes.Cost
		if respRes.Success {
			successNum = successNum + 1
			if maxTime <= respRes.Cost {
				maxTime = respRes.Cost
			}
			if minTime > respRes.Cost {
				minTime = respRes.Cost
			}

			if _, ok := chanIds[respRes.ChanId]; !ok {
				chanIds[respRes.ChanId] = true
				chanIdLen = uint64(len(chanIds))
			}

			// success cost time, for P90 cal
			costTimeList = append(costTimeList, respRes.Cost)

		} else {
			failureNum = failureNum + 1
		}

		mutex.Unlock()
	}

	// 数据全部接受完成，停止定时输出统计数据
	stopChan <- true
	endTime := time.Now()
	requestCostTime = endTime.Sub(startTime)
	calculateData(concurrency, processingTime, requestCostTime, maxTime, minTime, successNum, failureNum, chanIdLen, &respCodeMap)

	fmt.Printf("\n\n")
	fmt.Println("*************************  结果 stat  ****************************")
	fmt.Println("处理协程数量:", concurrency)
	fmt.Printf("请求总数: %d 总请求时间: %.3f秒 successNum: %d failureNum: %d\n",
		successNum+failureNum, requestCostTime.Seconds(), successNum, failureNum)
	printTop(costTimeList)
	fmt.Println("*************************  结果 end   ****************************")
	fmt.Printf("\n\n")
}

func calculateData(concurrent uint64, processingTime, costTime, maxTime, minTime time.Duration, successNum, failureNum, chanIdLen uint64, respCodeMap *sync.Map) {
	if processingTime == 0 || successNum == 0 {
		return
	}

	// QPS: 协程数 * (成功数/总耗时)
	qps := float64(successNum*concurrent) / processingTime.Seconds()

	// avg cost: 总耗时/总请求数
	averageTime := processingTime.Seconds() / float64(successNum)

	result := fmt.Sprintf("%4.0fs│%7d│%7d│%7d│%8.2f│%10.2fs│%10.2fs│%10.2fs│%v",
		costTime.Seconds(), chanIdLen, successNum, failureNum, qps, averageTime, minTime.Seconds(), maxTime.Seconds(), printMap(respCodeMap))
	fmt.Println(result)
}

func printHeader() {
	fmt.Printf("\n\n")
	fmt.Println("─────┬───────┬───────┬───────┬────────┬───────────┬───────────┬───────────┬────────")
	fmt.Println(" cost│concurr│success│ failed│   qps  │ avg cost/s│ min cost/s│max cost/ms│ xxx    ")
	fmt.Println("─────┼───────┼───────┼───────┼────────┼───────────┼───────────┼───────────┼────────")
	return
}

// 打印响应状态码及数量, 如 200:5
func printMap(respCodeMap *sync.Map) (mapStr string) {
	var mapArr []string

	respCodeMap.Range(func(key, value interface{}) bool {
		mapArr = append(mapArr, fmt.Sprintf("%v:%v", key, value))
		return true
	})
	sort.Strings(mapArr)
	mapStr = strings.Join(mapArr, ";")
	return
}

// printTop top 90 95 99
func printTop(costTimeList []time.Duration) {
	if len(costTimeList) == 0 {
		return
	}

	all := durationArray{}
	all = costTimeList
	sort.Sort(all)
	fmt.Println("P90:", fmt.Sprintf("%.2fs", all[int(float64(len(all))*0.90)].Seconds()))
	fmt.Println("P95:", fmt.Sprintf("%.2fs", all[int(float64(len(all))*0.95)].Seconds()))
	fmt.Println("P99:", fmt.Sprintf("%.2fs", all[int(float64(len(all))*0.99)].Seconds()))
}

type durationArray []time.Duration

func (array durationArray) Len() int           { return len(array) }
func (array durationArray) Swap(i, j int)      { array[i], array[j] = array[j], array[i] }
func (array durationArray) Less(i, j int) bool { return array[i] < array[j] }
