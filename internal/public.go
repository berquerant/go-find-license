package internal

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/time/rate"
)

type (
	HTTPClient interface {
		Do(*http.Request) (*http.Response, error)
	}

	Fetcher struct {
		client HTTPClient
	}

	License interface {
		URI() string
		Module() *Module
		Source() string
		Content() string
		Type() string
		Err() error
	}

	license struct {
		uri     string
		module  *Module
		source  string
		content string
		typ     string
	}

	errLicense struct {
		uri    string
		module *Module
		err    error
	}
)

func NewFetcherWithClient(client HTTPClient) *Fetcher {
	return &Fetcher{
		client: client,
	}
}

func NewFetcher() *Fetcher {
	return NewFetcherWithClient(
		&http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					ServerName: "pkg.go.dev",
					MinVersion: tls.VersionTLS13,
				},
			},
			Timeout: time.Second * 5,
		},
	)
}

const (
	fetchLicensesChannelSize = 4
	fetchLicensesPerSec      = 1
	fetchLicensesConcurrency = 2
)

var (
	ErrStatusNotOK   = errors.New("status not ok")
	ErrLoadingModule = errors.New("loading module")
)

// FetchLicenses searches the licenses of the modules from pkg.go.dev.
func (f *Fetcher) FetchLicenses(ctx context.Context, modules Modules) <-chan License {
	Infof("Start fetch %d licenses", len(modules))

	var (
		resultC    = make(chan License, fetchLicensesChannelSize)
		limiter    = rate.NewLimiter(fetchLicensesPerSec, 1)
		semaphoreC = make(chan struct{}, fetchLicensesConcurrency)
		waiter     sync.WaitGroup
	)
	go func() {
		defer func() {
			waiter.Wait()
			close(resultC)
		}()

		for _, module := range modules {
			if err := limiter.Wait(ctx); err != nil {
				return
			}
			semaphoreC <- struct{}{}
			waiter.Add(1)
			module := module
			go func() {
				resultC <- f.fetchLicense(ctx, module)
				waiter.Done()
				<-semaphoreC
			}()
		}
	}()
	return resultC
}

func (f *Fetcher) fetchLicense(ctx context.Context, module *Module) License {
	var (
		targetURL = f.licensesURL(module)
		newErr    = func(err error) License {
			return newErrLicense(module, targetURL, err)
		}
	)

	Infof("Fetch %s", targetURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return newErr(fmt.Errorf("new http req %w", err))
	}
	resp, err := f.client.Do(req)
	if err != nil {
		return newErr(fmt.Errorf("do http req %w", err))
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return newErr(fmt.Errorf("%w %d", ErrStatusNotOK, resp.StatusCode))
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return newErr(fmt.Errorf("new doc reader %w", err))
	}

	typ := doc.Find(`#\#lic-0`).Text()
	source := strings.TrimLeft(doc.Find(".License-source").Text(), "Source: ")
	content := doc.Find(".License-contents").Text()
	return newLicense(module, source, content, typ, targetURL)
}

func (*Fetcher) licensesURL(module *Module) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("https://pkg.go.dev/%s", module.Path))
	if module.Version != "" {
		b.WriteString(fmt.Sprintf("@%s", module.Version))
	}
	b.WriteString("?tab=licenses")
	return b.String()
}

func newLicense(module *Module, source, content, typ, uri string) License {
	return &license{
		uri:     uri,
		module:  module,
		source:  source,
		content: content,
		typ:     typ,
	}
}

func newErrLicense(module *Module, uri string, err error) License {
	return &errLicense{
		uri:    uri,
		module: module,
		err:    err,
	}
}

func (l *license) URI() string     { return l.uri }
func (l *license) Module() *Module { return l.module }
func (l *license) Source() string  { return l.source }
func (l *license) Content() string { return l.content }
func (l *license) Type() string    { return l.typ }
func (*license) Err() error        { return nil }

func (l *errLicense) URI() string     { return l.uri }
func (l *errLicense) Module() *Module { return l.module }
func (*errLicense) Source() string    { return "" }
func (*errLicense) Content() string   { return "" }
func (*errLicense) Type() string      { return "" }
func (l *errLicense) Err() error      { return l.err }
