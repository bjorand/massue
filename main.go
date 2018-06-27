package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/valyala/fasthttp"
)

var netTransport = &http.Transport{
	Dial: (&net.Dialer{
		Timeout: 5 * time.Second,
	}).Dial,
	TLSHandshakeTimeout: 5 * time.Second,
}
var netClient = &http.Client{
	Timeout:   time.Second * 3,
	Transport: netTransport,
}
var done chan int
var workersStat []*workerStats
var wg sync.WaitGroup

var (
	workerCount = flag.Int("c", 1, "Number of multiple requests to make at a time")
	numReqs     = flag.Int("n", 1, "Number of requests to perform")
	url         = flag.String("u", "", "URL")
	benchStart  time.Time
)

type workerStats struct {
	failed  int64
	success int64
}

// d in ns
func durationFormatter(d int64) string {
	d = d / 1000 / 1000
	if d >= 1000 {
		return fmt.Sprintf("%.2fs", float64(d)/1000)
	}
	return fmt.Sprintf("%dms", d)
}

func compileWorkersStat(start time.Time, end time.Time) {
	var totalFailed int64
	var totalSuccess int64
	fmt.Println("Computing workers stats")
	for _, s := range workersStat {
		totalFailed = totalFailed + s.failed
		totalSuccess = totalSuccess + s.success
	}
	fmt.Println("success:", totalSuccess, "failed:", totalFailed)
	duration := end.UnixNano() - start.UnixNano()
	fmt.Printf("%.2freq/s, duration: %s", (float64(totalFailed+totalSuccess))/(float64(duration)/1000/1000/1000), durationFormatter(duration))
}

func worker(workerID int, numReqs int, sleep time.Duration) {
	defer log.Printf("worker %d exited", workerID)
	defer wg.Done()
	log.Printf("worker %d: delaying start for %s", workerID, durationFormatter(int64(sleep)))
	time.Sleep(sleep)
	client := fasthttp.Client{
		Name: fmt.Sprintf("massue worker:%d", workerID),
	}
	workerStats := &workerStats{}
	for i := 0; i < numReqs; i++ {
		start := time.Now().UnixNano()
		if benchStart.IsZero() {
			benchStart = time.Now()
		}
		statusCode, _, err := client.GetDeadline(nil, *url, time.Now().Add(10*time.Second))
		if err != nil {
			log.Printf("worker %d: %+v", workerID, err)
			workerStats.failed++
			continue
		}
		duration := time.Now().UnixNano() - start
		log.Printf("worker %d: got %d in %s", workerID, statusCode, durationFormatter(duration))
		workerStats.success++
	}
	workersStat = append(workersStat, workerStats)
}

func main() {
	flag.Parse()
	rand.Seed(time.Now().Unix())
	workers := 0
	for workerID := 0; workerID < *workerCount; workerID++ {
		sleep := time.Duration(rand.Intn(1000)) * time.Millisecond
		wg.Add(1)
		numReqsReal := *numReqs / *workerCount
		if workerID == 0 {
			numReqsReal += *numReqs % *workerCount
		}
		go worker(workerID, numReqsReal, sleep)
		workers++
	}
	wg.Wait()
	end := time.Now()
	compileWorkersStat(benchStart, end)
}
