/*
 * Copyright (c) 2013 Matt Jibson <matt.jibson@gmail.com>
 *
 * Permission to use, copy, modify, and distribute this software for any
 * purpose with or without fee is hereby granted, provided that the above
 * copyright notice and this permission notice appear in all copies.
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 */

package appstats

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	KEY_PREFIX = "__appstats__:"
	KEY_PART   = KEY_PREFIX + "%06d:part"
	KEY_FULL   = KEY_PREFIX + "%v:full"
	DISTANCE   = 100
	MODULUS    = 1000
)

type RequestStats struct {
	User        string
	Admin       bool
	Method      string
	Path, Query string
	Status      int
	Start       time.Time
	Duration    time.Duration
	RPCStats    []RPCStat

	lock sync.Mutex
	wg   sync.WaitGroup
}

type stats_part RequestStats

type stats_full struct {
	Header http.Header
	Stats  *RequestStats
}

func (r RequestStats) PartKey() string {
	t := (r.Start.Nanosecond() / 1000 / DISTANCE) % MODULUS * DISTANCE
	return fmt.Sprintf(KEY_PART, t)
}

func (r RequestStats) FullKey() string {
	return fmt.Sprintf(KEY_FULL, r.Start.Nanosecond())
}

type RPCStat struct {
	Service, Method string
	Start           time.Time
	Offset          time.Duration
	Duration        time.Duration
	StackData       string
	In, Out         string
}

func (r RPCStat) Name() string {
	return r.Service + "." + r.Method
}

func (r RPCStat) Request() string {
	return r.In
}

func (r RPCStat) Response() string {
	return r.Out
}

func (r RPCStat) Stack() Stack {
	s := Stack{}

	lines := strings.Split(r.StackData, "\n")
	for i := 0; i < len(lines); i++ {
		idx := strings.LastIndex(lines[i], " ")
		if idx == -1 {
			break
		}

		cidx := strings.LastIndex(lines[i], ":")
		lineno, _ := strconv.Atoi(lines[i][cidx+1 : idx])
		f := &Frame{
			Location: lines[i][:cidx],
			Lineno:   lineno,
		}

		if i+1 < len(lines) && strings.HasPrefix(lines[i+1], "\t") {
			f.Call = strings.TrimSpace(lines[i+1])
			i++
		}

		s = append(s, f)
	}

	return s[2:]
}

type Stack []*Frame

type Frame struct {
	Location string
	Call     string
	Lineno   int
}

type AllRequestStats []*RequestStats

func (s AllRequestStats) Len() int           { return len(s) }
func (s AllRequestStats) Less(i, j int) bool { return s[i].Start.Sub(s[j].Start) > 0 }
func (s AllRequestStats) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

type StatsByName []*StatByName

func (s StatsByName) Len() int           { return len(s) }
func (s StatsByName) Less(i, j int) bool { return s[i].Count < s[j].Count }
func (s StatsByName) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

type StatByName struct {
	Name          string
	Count         int
	Cost, CostPct int
	SubStats      []*StatByName
	Requests      int
	RecentReqs    []int
	RequestStats  *RequestStats
	Duration      time.Duration
}

type Reverse struct{ sort.Interface }

func (r Reverse) Less(i, j int) bool { return r.Interface.Less(j, i) }

type SKey struct {
	a, b string
}
