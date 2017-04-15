package ssh

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"
)

func TestServerShutdown(t *testing.T) {
	l := newLocalListener()
	testBytes := []byte("Hello world\n")
	s := &Server{
		Handler: func(s Session) {
			s.Write(testBytes)
			time.Sleep(50 * time.Millisecond)
		},
	}
	go func() {
		err := s.Serve(l)
		if err != nil && err != ErrServerClosed {
			t.Fatal(err)
		}
	}()
	sessDone := make(chan struct{})
	sess, cleanup := newClientSession(t, l.Addr().String(), nil)
	go func() {
		defer cleanup()
		defer close(sessDone)
		var stdout bytes.Buffer
		sess.Stdout = &stdout
		if err := sess.Run(""); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(stdout.Bytes(), testBytes) {
			t.Fatalf("expected = %s; got %s", testBytes, stdout.Bytes())
		}
	}()

	srvDone := make(chan struct{})
	go func() {
		defer close(srvDone)
		err := s.Shutdown(context.Background())
		if err != nil {
			t.Fatal(err)
		}
	}()

	timeout := time.After(time.Second * 2)
	select {
	case <-timeout:
		t.Fatal("timeout")
		return
	case <-srvDone:
		// TODO: add timeout for sessDone
		<-sessDone
		return
	}
}

func TestServerClose(t *testing.T) {
	l := newLocalListener()
	s := &Server{
		Handler: func(s Session) {
			time.Sleep(5 * time.Second)
		},
	}
	go func() {
		err := s.Serve(l)
		if err != nil && err != ErrServerClosed {
			t.Fatal(err)
		}
	}()

	doneCh := make(chan struct{})
	sess, cleanup := newClientSession(t, l.Addr().String(), nil)
	go func() {
		defer cleanup()
		defer close(doneCh)
		if err := sess.Run(""); err != nil && err != io.EOF {
			t.Fatal(err)
		}
	}()

	go func() {
		err := s.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	timeout := time.After(time.Millisecond * 100)
	select {
	case <-timeout:
		t.Error("timeout")
		return
	case <-s.getDoneChan():
		<-doneCh
		return
	}
}
