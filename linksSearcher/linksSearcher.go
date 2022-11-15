package linksSearcher

// package main

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
)

const (
	regexpStr = `((([A-Za-z]{3,9}:(?:\/\/)?)(?:[-;:&=\+\$,\w]+@)?[A-Za-z0-9.-]+|(?:www.|[-;:&=\+\$,\w]+@)[A-Za-z0-9.-]+)((?:\/[\+~%\/.\w-_]*)?\??(?:[-\+=&;%@.\w_]*)#?(?:[.\!\/\\w]*))?)`
)

type WaitGroupCount struct {
	sync.WaitGroup
	count int
}

func FindLinks(procNumber int, inputFilename, outputFilename string) error {
	// implement here
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

	for !errors.Is(err, io.EOF) {
		if wg.count == procNumber {
			continue
		}
		wg.Add(1)
		wg.count++

		line := ""

		if iFile == os.Stdin {
			fmt.Println("Enter string:")
		}

		line, err = reader.ReadString('\n')

		if err != nil && !errors.Is(err, io.EOF) {
			return err
		}

		go func() {
			urls := urlSearch(line)
			if len(urls) > 0 {
				writeToSlice(&urls, &res, &mu)
			}
			wg.count--
			wg.Done()
		}()

	}
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

func urlSearch(s string) []string {
	urls := cutString(s)

	res := []string{}

	r := regexp.MustCompile(regexpStr)

	for i, v := range urls {
		urls[i] = r.FindString(v)
	}

	for _, url := range urls {
		resp, err := http.Get(url)
		if err != nil {
			continue
		}
		if resp.StatusCode == 200 {
			res = append(res, url)
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
