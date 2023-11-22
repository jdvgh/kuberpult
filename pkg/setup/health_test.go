/*This file is part of kuberpult.

Kuberpult is free software: you can redistribute it and/or modify
it under the terms of the Expat(MIT) License as published by
the Free Software Foundation.

Kuberpult is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
MIT License for more details.

You should have received a copy of the MIT License
along with kuberpult. If not, see <https://directory.fsf.org/wiki/License:Expat>.

Copyright 2023 freiheit.com*/

package setup

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/google/go-cmp/cmp"
)

func TestHealthReporter(t *testing.T) {
	tcs := []struct {
		Name               string
		ReportHealth       Health
		ReportMessage      string
		ExpectedHealthBody string
		ExpectedStatus     int
		ExpectedMetricBody string
	}{
		{
			Name: "reports starting",

			ExpectedStatus:     500,
			ExpectedHealthBody: `{"a":{"health":"starting"}}`,
			ExpectedMetricBody: `# HELP background_job_ready 
# TYPE background_job_ready gauge
background_job_ready{name="a"} 0
`,
		},
		{
			Name:          "reports ready",
			ReportHealth:  HealthReady,
			ReportMessage: "running",

			ExpectedStatus:     200,
			ExpectedHealthBody: `{"a":{"health":"ready","message":"running"}}`,
			ExpectedMetricBody: `# HELP background_job_ready 
# TYPE background_job_ready gauge
background_job_ready{name="a"} 1
`,
		},
		{
			Name:          "reports failed",
			ReportHealth:  HealthFailed,
			ReportMessage: "didnt work",

			ExpectedStatus:     500,
			ExpectedHealthBody: `{"a":{"health":"failed","message":"didnt work"}}`,
			ExpectedMetricBody: `# HELP background_job_ready 
# TYPE background_job_ready gauge
background_job_ready{name="a"} 0
`,
		},
	}
	for _, tc := range tcs {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			stateChange := make(chan struct{})
			cfg := ServerConfig{
				HTTP: []HTTPConfig{
					{
						Port: "18883",
					},
				},
				Background: []BackgroundTaskConfig{
					{
						Name: "a",
						Run: func(ctx context.Context, hr *HealthReporter) error {
							hr.ReportHealth(tc.ReportHealth, tc.ReportMessage)
							stateChange <- struct{}{}
							<-ctx.Done()
							return nil
						},
					},
				},
			}
			ctx, cancel := context.WithCancel(context.Background())
			doneCh := make(chan struct{})
			go func() {
				Run(ctx, cfg)
				doneCh <- struct{}{}
			}()
			<-stateChange
			status, body := getHttp(t, "http://localhost:18883/healthz")
			if status != tc.ExpectedStatus {
				t.Errorf("wrong http status, expected %d, got %d", tc.ExpectedStatus, status)
			}
			d := cmp.Diff(body, tc.ExpectedHealthBody)
			if d != "" {
				t.Errorf("wrong body, diff: %s", d)
			}
			_, metricBody := getHttp(t, "http://localhost:18883/metrics")
			if status != tc.ExpectedStatus {
				t.Errorf("wrong http status, expected %d, got %d", tc.ExpectedStatus, status)
			}
			d = cmp.Diff(metricBody, tc.ExpectedMetricBody)
			if d != "" {
				t.Errorf("wrong body, diff: %s", d)
			}
			cancel()
			<-doneCh

		})
	}
}

type mockBackoff struct {
	called   uint
	resetted uint
}

func (b *mockBackoff) NextBackOff() time.Duration {
	b.called = b.called + 1
	return 1 * time.Nanosecond
}

func (b *mockBackoff) Reset() {
	b.resetted = b.resetted + 1
	return
}

func TestHealthReporterRetry(t *testing.T) {
	type step struct {
		ReportHealth  Health
		ReportMessage string
		ReturnError   error

		ExpectReady         bool
		ExpectBackoffCalled uint
		ExpectResetCalled   uint
	}
	tcs := []struct {
		Name string

		Steps []step

		ExpectError string
	}{
		{
			Name: "reports healthy",
			Steps: []step{
				{
					ReportHealth: HealthReady,

					ExpectReady:       true,
					ExpectResetCalled: 1,
				},
			},
		},
		{
			Name: "reports unhealthy if there is an error",
			Steps: []step{
				{
					ReturnError: fmt.Errorf("no"),

					ExpectReady:         false,
					ExpectBackoffCalled: 1,
				},
			},
		},
		{
			Name: "doesnt retry permanent errors",
			Steps: []step{
				{
					ReturnError: Permanent(fmt.Errorf("no")),

					ExpectReady:         false,
					ExpectBackoffCalled: 0,
				},
			},
			ExpectError: "no",
		},
		{
			Name: "retries some times and resets once it's healthy",
			Steps: []step{
				{
					ReturnError: fmt.Errorf("no"),

					ExpectReady:         false,
					ExpectBackoffCalled: 1,
				},
				{
					ReturnError: fmt.Errorf("no"),

					ExpectReady:         false,
					ExpectBackoffCalled: 2,
				},
				{
					ReturnError: fmt.Errorf("no"),

					ExpectReady:         false,
					ExpectBackoffCalled: 3,
				},
				{
					ReportHealth: HealthReady,

					ExpectReady:         true,
					ExpectBackoffCalled: 3,
					ExpectResetCalled:   1,
				},
			},
		},
	}
	for _, tc := range tcs {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			stepCh := make(chan step)
			stateChange := make(chan struct{}, len(tc.Steps))
			bo := &mockBackoff{}
			hs := HealthServer{}
			hs.BackOffFactory = func() backoff.BackOff { return bo }
			ctx, cancel := context.WithCancel(context.Background())
			errCh := make(chan error)
			go func() {
				hr := hs.Reporter("a")
				errCh <- hr.Retry(ctx, func() error {
					for {
						select {
						case <-ctx.Done():
							return nil
						case st := <-stepCh:
							if st.ReturnError != nil {

								stateChange <- struct{}{}
								return st.ReturnError
							}
							hr.ReportHealth(st.ReportHealth, st.ReportMessage)
							stateChange <- struct{}{}
						}
					}
				})
			}()
			for _, st := range tc.Steps {
				stepCh <- st
				<-stateChange
				ready := hs.IsReady("a")
				if st.ExpectReady != ready {
					t.Errorf("expected ready status to %t but got %t", st.ExpectReady, ready)
				}
				if st.ExpectBackoffCalled != bo.called {
					t.Errorf("wrong number of backoffs called, expected %d, but got %d", st.ExpectBackoffCalled, bo.called)
				}
				if st.ExpectResetCalled != bo.resetted {
					t.Errorf("wrong number of backoff resets, expected %d, but got %d", st.ExpectResetCalled, bo.resetted)
				}

			}
			cancel()
			err := <-errCh
			if tc.ExpectError == "" {
				if err != nil {
					t.Errorf("expected no error but got %q", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error %q but got nil", tc.ExpectError)
				} else if err.Error() != tc.ExpectError {
					t.Errorf("expected error %q but got %q", tc.ExpectError, err)
				}
			}
			close(stepCh)

		})
	}
}

func getHttp(t *testing.T, url string) (int, string) {
	for i := 0; i < 10; i = i + 1 {
		resp, err := http.Get(url)
		if err != nil {
			t.Log(err)
			<-time.After(time.Second)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, string(body)
	}
	t.FailNow()
	return 0, ""
}