package runner_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers/runner"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Runner", func() {
	Describe("RunFunc", func() {
		It("adapts a function to the Runner interface", func() {
			var called atomic.Bool
			r := runner.RunFunc(func(ctx context.Context, ready chan<- struct{}) error {
				called.Store(true)
				close(ready)
				<-ctx.Done()
				return nil
			})

			ctx, cancel := context.WithCancel(context.Background())
			ready := make(chan struct{})
			done := make(chan error, 1)
			go func() { done <- r.Run(ctx, ready) }()

			Eventually(ready).Should(BeClosed())
			Expect(called.Load()).To(BeTrue())
			cancel()
			Eventually(done).Should(Receive(BeNil()))
		})
	})

	Describe("StartOrdered", func() {
		It("starts all members and signals ready in order", func() {
			var mu sync.Mutex
			var order []string

			makeRunner := func(name string) runner.Runner {
				return runner.RunFunc(func(ctx context.Context, ready chan<- struct{}) error {
					mu.Lock()
					order = append(order, name+"-started")
					mu.Unlock()
					close(ready)
					<-ctx.Done()
					mu.Lock()
					order = append(order, name+"-stopped")
					mu.Unlock()
					return nil
				})
			}

			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan error, 1)
			go func() {
				done <- runner.StartOrdered(ctx, []runner.Member{
					{Name: "first", Runner: makeRunner("first")},
					{Name: "second", Runner: makeRunner("second")},
					{Name: "third", Runner: makeRunner("third")},
				})
			}()

			Eventually(func() int {
				mu.Lock()
				defer mu.Unlock()
				return len(order)
			}).Should(BeNumerically(">=", 3))
			cancel()
			Eventually(done).Should(Receive(BeNil()))

			mu.Lock()
			defer mu.Unlock()
			Expect(order[:3]).To(Equal([]string{"first-started", "second-started", "third-started"}))
		})

		It("shuts down in LIFO order", func() {
			shutdownOrder := make(chan string, 3)

			makeRunner := func(name string) runner.Runner {
				return runner.RunFunc(func(ctx context.Context, ready chan<- struct{}) error {
					close(ready)
					<-ctx.Done()
					shutdownOrder <- name
					return nil
				})
			}

			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan error, 1)
			go func() {
				done <- runner.StartOrdered(ctx, []runner.Member{
					{Name: "A", Runner: makeRunner("A")},
					{Name: "B", Runner: makeRunner("B")},
					{Name: "C", Runner: makeRunner("C")},
				})
			}()

			time.Sleep(50 * time.Millisecond)
			cancel()
			Eventually(done, 5*time.Second).Should(Receive(BeNil()))

			Eventually(shutdownOrder).Should(HaveLen(3))
			close(shutdownOrder)
			var order []string
			for name := range shutdownOrder {
				order = append(order, name)
			}
			Expect(order).To(Equal([]string{"C", "B", "A"}))
		})

		It("returns error if a runner exits before ready", func() {
			failing := runner.RunFunc(func(_ context.Context, _ chan<- struct{}) error {
				return errors.New("init failed")
			})

			healthy := runner.RunFunc(func(ctx context.Context, ready chan<- struct{}) error {
				close(ready)
				<-ctx.Done()
				return nil
			})

			err := runner.StartOrdered(context.Background(), []runner.Member{
				{Name: "healthy", Runner: healthy},
				{Name: "failing", Runner: failing},
			})

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failing"))
			Expect(err.Error()).To(ContainSubstring("init failed"))
		})

		It("propagates errors from runners", func() {
			r := runner.RunFunc(func(ctx context.Context, ready chan<- struct{}) error {
				close(ready)
				<-ctx.Done()
				return errors.New("shutdown error")
			})

			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan error, 1)
			go func() {
				done <- runner.StartOrdered(ctx, []runner.Member{
					{Name: "erroring", Runner: r},
				})
			}()

			time.Sleep(20 * time.Millisecond)
			cancel()

			var err error
			Eventually(done).Should(Receive(&err))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("shutdown error"))
		})

		It("exits when a member fails at runtime", func() {
			failing := runner.RunFunc(func(ctx context.Context, ready chan<- struct{}) error {
				close(ready)
				return errors.New("runtime crash")
			})

			blocking := runner.RunFunc(func(ctx context.Context, ready chan<- struct{}) error {
				close(ready)
				<-ctx.Done()
				return nil
			})

			done := make(chan error, 1)
			go func() {
				done <- runner.StartOrdered(context.Background(), []runner.Member{
					{Name: "blocking", Runner: blocking},
					{Name: "failing", Runner: failing},
				})
			}()

			var err error
			Eventually(done, 5*time.Second).Should(Receive(&err))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("runtime crash"))
		})

		It("exits cleanly when context is cancelled externally", func() {
			r := runner.RunFunc(func(ctx context.Context, ready chan<- struct{}) error {
				close(ready)
				<-ctx.Done()
				return nil
			})

			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan error, 1)
			go func() {
				done <- runner.StartOrdered(ctx, []runner.Member{
					{Name: "service", Runner: r},
				})
			}()

			time.Sleep(50 * time.Millisecond)
			cancel()

			Eventually(done, 5*time.Second).Should(Receive(BeNil()))
		})
	})

	Describe("HTTPServer", func() {
		It("serves HTTP requests and shuts down gracefully", func() {
			handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = fmt.Fprint(w, "ok")
			})

			srv := runner.HTTPServer("localhost:0", handler, nil)

			ctx, cancel := context.WithCancel(context.Background())
			ready := make(chan struct{})
			done := make(chan error, 1)

			go func() { done <- srv.Run(ctx, ready) }()

			Eventually(ready).Should(BeClosed())
			addr := srv.(interface{ Addr() string }).Addr()

			resp, err := http.Get("http://" + addr + "/")
			Expect(err).NotTo(HaveOccurred())
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			Expect(string(body)).To(Equal("ok"))

			cancel()
			Eventually(done).Should(Receive(BeNil()))
		})

		It("serves HTTPS when TLS config is provided", func() {
			certDir := os.Getenv("AUTOSCALER_TEST_CERT_DIR")
			if certDir == "" {
				certDir = "../../test-certs"
			}

			certFile := certDir + "/api.crt"
			keyFile := certDir + "/api.key"
			caFile := certDir + "/autoscaler-ca.crt"

			if _, err := os.Stat(certFile); os.IsNotExist(err) {
				Skip("test certs not available")
			}

			cert, err := tls.LoadX509KeyPair(certFile, keyFile)
			Expect(err).NotTo(HaveOccurred())

			tlsConfig := &tls.Config{
				Certificates: []tls.Certificate{cert},
				MinVersion:   tls.VersionTLS12,
			}

			handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = fmt.Fprint(w, "secure")
			})

			srv := runner.HTTPServer("localhost:0", handler, tlsConfig)
			ctx, cancel := context.WithCancel(context.Background())
			ready := make(chan struct{})
			done := make(chan error, 1)
			go func() { done <- srv.Run(ctx, ready) }()

			Eventually(ready).Should(BeClosed())
			addr := srv.(interface{ Addr() string }).Addr()

			caCert, err := os.ReadFile(caFile)
			Expect(err).NotTo(HaveOccurred())
			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(caCert)

			client := &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						RootCAs:    caCertPool,
						ServerName: "api",
						MinVersion: tls.VersionTLS12,
					},
				},
			}

			resp, err := client.Get("https://" + addr + "/")
			Expect(err).NotTo(HaveOccurred())
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			Expect(string(body)).To(Equal("secure"))

			cancel()
			Eventually(done).Should(Receive(BeNil()))
		})
	})
})
