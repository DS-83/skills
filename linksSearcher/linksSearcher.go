package linksSearcher

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	regexpStr = `((([A-Za-z]{3,9}:(?:\/\/)?)(?:[-;:&=\+\$,\w]+@)?[A-Za-z0-9.-]+|(?:www.|[-;:&=\+\$,\w]+@)[A-Za-z0-9.-]+)((?:\/[\+~%\/.\w-_]*)?\??(?:[-\+=&;%@.\w_]*)#?(?:[.\!\/\\w]*))?)`
)

var regUrl = regexp.MustCompile(regexpStr)

type WaitGroupCount struct {
	sync.WaitGroup
	sync.Mutex
	count int
}

func (w *WaitGroupCount) decrement() {
	defer w.Unlock()
	w.Lock()
	w.count--
}
func (w *WaitGroupCount) increment() {
	defer w.Unlock()
	w.Lock()
	w.count++
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

	wg := WaitGroupCount{}
	var mu sync.Mutex

	res := []string{}
	done := make(chan struct{})

	c := sync.NewCond(&sync.Mutex{})

	go func() {

		for !errors.Is(err, io.EOF) {
			if wg.count == procNumber {
				c.L.Lock()
				c.Wait()
				c.L.Unlock()
			}

			line := ""

			if iFile == os.Stdin {
				fmt.Println("Enter string:")
			}

			line, err = reader.ReadString('\n')

			if err != nil && !errors.Is(err, io.EOF) {
				fmt.Println(err)
				done <- struct{}{}
			}

			wg.Add(1)
			wg.increment()
			go func() {
				defer wg.Done()
				defer wg.decrement()

				urls := urlSearch(line)
				if len(urls) > 0 {
					writeToSlice(&urls, &res, &mu)
				}
				if wg.count == procNumber {
					c.Broadcast()
				}
			}()

		}
		done <- struct{}{}

	}()
	<-done
	wg.Wait()

	for _, v := range res {
		fmt.Fprintln(oFile, v)
	}

	return nil
}

func writeToSlice(in, out *[]string, mu *sync.Mutex) {
	mu.Lock()
	defer mu.Unlock()

	*out = append(*out, *in...)
}

func urlSearch(s string) (res []string) {
	urls := cutString(s)

	for i, v := range urls {
		urls[i] = regUrl.FindString(v)
	}

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
		case result := <-ch:
			res = append(res, result)
		case <-timeout:
			return
		}
	}
	return res
}

func cutString(s string) []string {
	res := []string{}

	idx1 := strings.Index(s, "http")

	if idx1 != -1 {
		idx2 := strings.Index(s[idx1+4:], "http")
		if idx2 != -1 {
			idx2 += idx1 + 4
			res = append(res, s[idx1:idx2])
			if next := cutString(s[idx2:]); len(next) > 0 {
				res = append(res, next...)
			}
		} else {
			res = append(res, s[idx1:])
		}
	}

	return res
}
