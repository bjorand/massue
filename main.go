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

	"github.com/cactus/go-statsd-client/statsd"
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
	workerCount  = flag.Int("c", 1, "Number of parallel worker")
	numReqs      = flag.Int("n", 1, "Total number of requests to perform")
	url          = flag.String("u", "", "URL")
	userPassword = flag.String("A", "", "Add Basic WWW Authentication, the attributes are a colon separated username and password.")
	statsdServer = flag.String("S", "", "Statsd server for metrics collection")
	benchStart   time.Time
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
	for _, s := range workersStat {
		totalFailed = totalFailed + s.failed
		totalSuccess = totalSuccess + s.success
	}
	fmt.Println("success:", totalSuccess, "failed:", totalFailed)
	duration := end.UnixNano() - start.UnixNano()
	fmt.Printf("%.2freq/s, duration: %s\n", (float64(totalFailed+totalSuccess))/(float64(duration)/1000/1000/1000), durationFormatter(duration))
}

func worker(workerID int, numReqs int, sleep time.Duration, statsdClient statsd.Statter) {
	defer log.Printf("worker %d exited", workerID)
	defer wg.Done()
	log.Printf("worker %d: delaying start for %s", workerID, durationFormatter(int64(sleep)))
	time.Sleep(sleep)
	client := fasthttp.Client{
		Name: fmt.Sprintf("massue worker:%d", workerID),
	}

	workerStats := &workerStats{}
	for i := 0; i < numReqs; i++ {
		// start := time.Now().UnixNano()
		if benchStart.IsZero() {
			benchStart = time.Now()
		}
		var req fasthttp.Request
		req.Header.SetMethod("POST")
		req.SetRequestURI(*url)
		var resp fasthttp.Response
		err := client.DoTimeout(&req, &resp, 10*time.Second)
		if err != nil {
			log.Printf("worker %d: %+v", workerID, err)
			statsdClient.Inc("failed", 1, 1.0)
			workerStats.failed++
			continue
		}
		statsdClient.Inc("success", 1, 1.0)
		// duration := time.Now().UnixNano() - start
		// log.Printf("worker %d: got %d in %s", workerID, resp.StatusCode(), durationFormatter(duration))
		workerStats.success++
	}
	workersStat = append(workersStat, workerStats)
}

func main() {
	statsdClient, err := statsd.NewClient("localhost:8125", "massue")

	if err != nil {
		log.Fatalf("Error connecting to statsd: %+v", err)
	}
	defer statsdClient.Close()
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
		go worker(workerID, numReqsReal, sleep, statsdClient)
		workers++
	}
	go func() {
		for {
			compileWorkersStat(benchStart, time.Now())
			time.Sleep(1 * time.Second)
		}
	}()
	wg.Wait()
	end := time.Now()
	compileWorkersStat(benchStart, end)
}
