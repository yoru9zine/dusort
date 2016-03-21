package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/nsf/termbox-go"
)

func main() {
	if !piped(os.Stdin) {
		log.Fatal("stdin is not pipe. please use `du -sh /path/to/dir/* | dusort`")
	}
	var m sync.Mutex
	if err := termbox.Init(); err != nil {
		log.Fatalf("failed to initialize terminal: %s", err)
	}
	defer termbox.Close()

	entryc := make(chan entry)
	var first *entry
	go readEntry(entryc)
	go func() {
		draw(nil, false)
		for e := range entryc {
			m.Lock()
			ee := e
			if first == nil {
				first = &ee
			} else {
				first = first.Insert(&ee)
			}
			draw(first.List(), false)
			m.Unlock()
		}
		draw(first.List(), true)
	}()

	for {
		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventKey:
			switch ev.Key {
			case termbox.KeyEsc:
				return
			}
			switch ev.Ch {
			case 'q':
				return
			}
		}
	}
}

func draw(entries []*entry, finished bool) {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	xmax, ymax := termbox.Size()
	x, y := 0, 0
	state := "Waiting input..."
	if finished {
		state = "Finished"
	}
	for _, r := range state {
		termbox.SetCell(x, y, r, termbox.ColorDefault, termbox.ColorDefault)
		x++
	}
	x = 0
	y++
	for _, e := range entries {
		x = 0
		for _, r := range e.String() {
			termbox.SetCell(x, y, r, termbox.ColorDefault, termbox.ColorDefault)
			x++
			if x > xmax {
				break
			}
		}
		y++
		if y > ymax {
			break
		}
	}
	termbox.Flush()
}

type entry struct {
	Name       string
	SizeString string
	Size       float64

	Next *entry
	Prev *entry
}

func (e *entry) String() string {
	return fmt.Sprintf("%s\t%s", e.SizeString, e.Name)
}

func (e *entry) List() (list []*entry) {
	ee := e
	for {
		if ee == nil {
			break
		}
		list = append(list, ee)
		ee = ee.Next
	}
	return
}

func (e *entry) Insert(n *entry) *entry {
	ee := e
	for {
		if ee.Size <= n.Size {
			if ee.Prev != nil {
				ee.Prev.Next = n
			}
			n.Prev = ee.Prev
			n.Next = ee
			ee.Prev = n
			return ee.First()
		}
		if ee.Next == nil {
			ee.Next = n
			n.Prev = ee
			return ee.First()
		}
		ee = ee.Next
	}
}

func (e *entry) First() *entry {
	ee := e
	for {
		if ee.Prev == nil {
			return ee
		}
		ee = ee.Prev
	}
}

var sizeSuffix = map[byte]float64{}

func init() {
	n := 1.0
	for _, r := range []byte{'K', 'M', 'G', 'T', 'P', 'E', 'Z', 'Y'} {
		n = n * 1024
		sizeSuffix[r] = n
	}
}

func parseSize(s string) (float64, error) {
	s = strings.TrimPrefix(s, " ")
	suffix := s[len(s)-1]
	m, ok := sizeSuffix[suffix]
	var num float64
	var err error
	if ok {
		num, err = strconv.ParseFloat(s[:len(s)-1], 64)
		num *= m
	} else {
		num, err = strconv.ParseFloat(s, 64)
	}
	return num, err
}

func readEntry(c chan entry) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		tokens := strings.Split(scanner.Text(), "\t")
		sizeStr := tokens[0]
		name := tokens[1]
		size, err := parseSize(sizeStr)
		if err != nil {
			log.Printf("parse error %s: %s\n", sizeStr, err)
		}
		c <- entry{
			Name:       name,
			SizeString: sizeStr,
			Size:       size,
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("error at readEntry: %s\n", err)
	}
	close(c)
}

func piped(f *os.File) bool {
	stat, _ := f.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		return true
	}
	return false
}
