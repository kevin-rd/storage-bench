package internal

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

func HandleStatics(concurrency uint64, ch <-chan *TestResult) {
	var (
		requestCostTimeList []time.Duration     // 耗时数组
		processingTime      time.Duration   = 0 // processingTime 处理总耗时
		requestCostTime     time.Duration   = 0 // requestCostTime 请求总时间
		maxTime             time.Duration   = 0 // maxTime 至今为止单个请求最大耗时
		minTime             time.Duration   = 0 // minTime 至今为止单个请求最小耗时
		successNum          uint64          = 0
		failureNum          uint64          = 0
		chanIdLen           uint64          = 0 // chanIdLen 并发数
		stopChan                            = make(chan bool)
		mutex                               = sync.RWMutex{}
		chanIds                             = make(map[int]bool)
	)

	startTime := time.Now()
	respCodeMap := sync.Map{}
	ticker := time.NewTicker(time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				endTime := time.Now()
				mutex.Lock()
				go calculateData(concurrency, processingTime, endTime.Sub(startTime), maxTime, minTime, successNum, failureNum, chanIdLen, &respCodeMap)
				mutex.Unlock()
			case <-stopChan:
				return
			}
		}
	}()
	printHeader()
	for respRes := range ch {
		mutex.Lock()
		processingTime = processingTime + respRes.Cost
		if maxTime <= respRes.Cost {
			maxTime = respRes.Cost
		}
		if minTime == 0 {
			minTime = respRes.Cost
		} else if minTime > respRes.Cost {
			minTime = respRes.Cost
		}
		if respRes.Err == nil {
			successNum = successNum + 1
		} else {
			failureNum = failureNum + 1
		}

		// 统计并发数
		if _, ok := chanIds[int(respRes.ChanId)]; !ok {
			chanIds[int(respRes.ChanId)] = true
			chanIdLen = uint64(len(chanIds))
		}
		requestCostTimeList = append(requestCostTimeList, respRes.Cost)
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
	fmt.Println("请求总数（并发数*请求数 -c * -n）:", successNum+failureNum, "总请求时间:",
		fmt.Sprintf("%.3f", float64(requestCostTime)/1e9),
		"秒", "successNum:", successNum, "failureNum:", failureNum)
	printTop(requestCostTimeList)
	fmt.Println("*************************  结果 end   ****************************")
	fmt.Printf("\n\n")
}

func calculateData(concurrent uint64, processingTime, costTime, maxTime, minTime time.Duration, successNum, failureNum, chanIdLen uint64, respCodeMap *sync.Map) {
	if processingTime == 0 || chanIdLen == 0 {
		return
	}

	var qps, averageTime, maxTimeFloat, minTimeFloat, requestCostTimeFloat float64

	// 平均 QPS 成功数*总协程数/总耗时 (每秒)
	qps = float64(successNum*1e9*concurrent) / float64(processingTime)

	// 平均耗时 总耗时/总请求数/并发数 纳秒=>毫秒
	if successNum != 0 && concurrent != 0 {
		averageTime = float64(processingTime) / float64(successNum*1e6)
	}
	maxTimeFloat = float64(maxTime) / 1e6
	minTimeFloat = float64(minTime) / 1e6
	requestCostTimeFloat = float64(costTime) / 1e9

	result := fmt.Sprintf("%4.0fs│%7d│%7d│%7d│%8.2f│%11.2f│%11.2f│%11.2f│%v",
		requestCostTimeFloat, chanIdLen, successNum, failureNum, qps, maxTimeFloat, minTimeFloat, averageTime, printMap(respCodeMap))
	fmt.Println(result)
}

func printHeader() {
	fmt.Printf("\n\n")
	fmt.Println("─────┬───────┬───────┬───────┬────────┬───────────┬───────────┬───────────┬────────")
	fmt.Println(" cost│ 并发数 │ 成功数 │ 失败数 │   qps  │max cost/ms│min cost/ms│avg cost/ms│ xxx    ")
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

// printTop 排序后计算 top 90 95 99
func printTop(requestCostTimeList []time.Duration) {
	if len(requestCostTimeList) == 0 {
		return
	}
	all := durationArray{}
	all = requestCostTimeList
	sort.Sort(all)
	fmt.Println("tp90:", fmt.Sprintf("%.3fms", float64(all[int(float64(len(all))*0.90)]/1e6)))
	fmt.Println("tp95:", fmt.Sprintf("%.3fms", float64(all[int(float64(len(all))*0.95)]/1e6)))
	fmt.Println("tp99:", fmt.Sprintf("%.3fms", float64(all[int(float64(len(all))*0.99)]/1e6)))
}

type durationArray []time.Duration

func (array durationArray) Len() int           { return len(array) }
func (array durationArray) Swap(i, j int)      { array[i], array[j] = array[j], array[i] }
func (array durationArray) Less(i, j int) bool { return array[i] < array[j] }
