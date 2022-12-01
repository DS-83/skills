package linksSearcher

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sync"
	"time"
)

const (
	regexpStr = `(((http[s]:(?:\/\/)?)(?:[-;:&=\+\$,\w]+@)?[A-Za-z0-9.-]+|(?:www.|[-;:&=\+\$,\w]+@)[A-Za-z0-9.-]+)((?:\/[\+~%\/.\w-_]*)?\??(?:[-\+=&;%@.\w_]*)#?(?:[.\!\/\\w]*))?)`
)

var regUrl = regexp.MustCompile(regexpStr)

type Job string
type Results string

type workerPool struct {
	workersCount int
	jobs         chan Job
	results      chan Results
}

func NewWorkerPool(c int) *workerPool {
	return &workerPool{
		workersCount: c,
		jobs:         make(chan Job, c),
		results:      make(chan Results),
	}
}

func (w *workerPool) Run() {
	wg := sync.WaitGroup{}

	for i := 0; i < w.workersCount; i++ {
		wg.Add(1)
		go worker(w.jobs, w.results, &wg)
	}

	wg.Wait()
	close(w.results)
}

func FindLinks(procNumber int, inputFilename, outputFilename string) error {
	iFile, err := os.Open(inputFilename)
	if err != nil {
		iFile = os.Stdin
	}
	defer iFile.Close()

	oFile, err := os.Create(outputFilename)
	if err != nil {
		oFile = os.Stdout
	}
	defer oFile.Close()

	reader := bufio.NewReader(iFile)

	done := make(chan struct{})

	wp := NewWorkerPool(procNumber)

	go func() {

		for !errors.Is(err, io.EOF) {
			line := ""

			if iFile == os.Stdin {
				fmt.Println("Enter string:")
			}

			line, err = reader.ReadString('\n')

			if err != nil && !errors.Is(err, io.EOF) {
				fmt.Println(err)
				done <- struct{}{}
			}

			wp.jobs <- Job(line)

		}
		done <- struct{}{}
		close(wp.jobs)

	}()

	go wp.Run()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		for v := range wp.results {
			fmt.Fprintln(oFile, v)
		}
		wg.Done()
	}()

	<-done
	wg.Wait()

	return nil
}

func worker(jobs <-chan Job, result chan<- Results, wg *sync.WaitGroup) {
	defer wg.Done()
	for s := range jobs {
		urls := regUrl.FindAllString(string(s), -1)

		ch := make(chan string)
		for _, url := range urls {
			go func(url string) {
				resp, err := http.Get(url)
				if err != nil {
					return
				}
				if resp.StatusCode == http.StatusOK {
					ch <- url
				}
			}(url)
		}

		timeout := time.After(5 * time.Second)
		for i := 0; i < len(urls); i++ {
			select {
			case r := <-ch:
				result <- Results(r)
			case <-timeout:
				return
			}
		}
	}
}
