// Copyright 2022 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package address

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"regexp"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"unicode"
)

const (
	// MustCompile against following expression.
	MustCompile = "^[a-z][a-z0-9-./]*[a-z0-9]$"
	dot         = '.'
)

var (
	ErrAddr = errors.New("address error")

	alnum = &unicode.RangeTable{ //nolint:gochecknoglobals
		R16: []unicode.Range16{
			{'0', '9', 1},
			{'A', 'Z', 1},
			{'a', 'z', 1},
		},
	}
)

type Address struct {
	url      *url.URL
	Host     string
	Instance string
}

// String reassembles the Address into a valid URL string.
// The general form of the result is one of:
//
//	happy://userinfo@host/path?query#fragment
//
// Any non-ASCII characters in host are escaped.
// To obtain the path, String uses net.URL.EscapedPath().
//
// In the second form, the following rules apply:
//   - if u.User is nil, userinfo@ is omitted.
//   - if u.Host is non-empty and u.Path begins with a /,
//     the form host/path does not add its own /.
//   - if u.RawQuery is empty, ?query is omitted.
//   - if u.Fragment is empty, #fragment is omitted.
func (a *Address) String() string {
	return a.url.String()
}

func (a *Address) Parse(ref string) (*Address, error) {
	refurl, err := a.url.Parse(ref)
	if err != nil {
		return nil, err
	}
	return &Address{
		url:      refurl,
		Instance: a.Instance,
		Host:     refurl.Host,
	}, nil
}

func (a *Address) ResolveService(svc string) (*Address, error) {
	if !strings.HasPrefix(svc, "happy://") {
		svc = path.Join(a.Instance, "service", svc)
	}
	svcaddr, err := a.Parse(svc)
	if err != nil {
		return nil, err
	}

	if !strings.HasPrefix(svcaddr.url.Path, "/"+svcaddr.Instance+"/service") {
		return nil, fmt.Errorf("%w: not a service %s", ErrAddr, svcaddr.String())
	}
	return svcaddr, nil
}

func FromModule(host, modulepath string) (*Address, error) {
	// fully qualified ?
	sl := strings.Split(modulepath, "/")
	if len(sl) == 1 {
		return Parse("happy://" + host + "/" + ensure(modulepath))
	}

	var rev []string
	var rmdomain bool
	if strings.Contains(sl[0], ".") {
		rmdomain = true
		domainparts := sort.StringSlice(strings.Split(sl[0], "."))
		sort.Sort(domainparts)
		rev = append(rev, ensure(strings.Join(domainparts, ".")))
	}
	p := len(sl)
	for i := 0; i < p; i++ {
		if rmdomain && i == 0 {
			continue
		}
		rev = append(rev, ensure(sl[i]))
	}
	return Parse("happy://" + host + "/" + strings.Join(rev, "."))
}

// Current returns MustCompile format of current application
// all non alpha numeric characters removed.
func Current() (*Address, error) {
	var name string
	if info, available := debug.ReadBuildInfo(); available {
		if info.Path == "command-line-arguments" {
			name = info.Deps[0].Path
		} else {
			name = info.Path
		}

	} else {
		pc, _, _, _ := runtime.Caller(0)
		ps := strings.Split(runtime.FuncForPC(pc).Name(), ".")
		pl := len(ps)
		if ps[pl-2][0] == '(' {
			name = strings.Join(ps[0:pl-2], ".")
		} else {
			name = strings.Join(ps[0:pl-1], ".")
		}
	}

	host, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	return FromModule(host, name)
}

// Valid returns true if s is string which is valid app name.
func Valid(s string) bool {
	re := regexp.MustCompile(MustCompile)
	return re.MatchString(s)
}

func Parse(rawAddr string) (*Address, error) {
	if rawAddr == "" {
		return nil, fmt.Errorf("%w: empty address", ErrAddr)
	}
	if !strings.HasPrefix(rawAddr, "happy://") {
		host, err := Current()
		if err != nil {
			return nil, err
		}
		full, err := url.JoinPath(host.String(), rawAddr)
		return Parse(full)
	}
	url, err := url.Parse(rawAddr)
	if err != nil {
		return nil, err
	}
	urlparts := strings.Split(url.Path, "/")
	var instance string
	if len(urlparts) > 1 {
		instance = urlparts[1]
	}
	return &Address{
		url:      url,
		Host:     url.Host,
		Instance: instance,
	}, nil
}

func ensure(in string) string {
	var b bytes.Buffer
	for _, c := range in {
		isAlnum := unicode.Is(alnum, c)
		isSpace := unicode.IsSpace(c)
		isLower := unicode.IsLower(c)
		if isSpace || (!isAlnum && c != dot) {
			continue
		}
		if !isLower {
			c = unicode.ToLower(c)
		}
		b.WriteRune(c)
	}
	return b.String()
}
