package lspserver

import (
	"io"
	"sync"
)

// pipe is a thread-safe in-memory buffer that implements io.ReadWriteCloser.
type pipe struct {
	mu     sync.Mutex
	cond   *sync.Cond
	buf    []byte
	closed bool
}

func newPipeEnd() *pipe {
	p := &pipe{}
	p.cond = sync.NewCond(&p.mu)
	return p
}

func (p *pipe) Read(data []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for len(p.buf) == 0 && !p.closed {
		p.cond.Wait()
	}
	if len(p.buf) == 0 && p.closed {
		return 0, io.EOF
	}
	n := copy(data, p.buf)
	p.buf = p.buf[n:]
	return n, nil
}

func (p *pipe) Write(data []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return 0, io.ErrClosedPipe
	}
	p.buf = append(p.buf, data...)
	p.cond.Signal()
	return len(data), nil
}

func (p *pipe) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.closed = true
	p.cond.Broadcast()
	return nil
}

// rwc wraps separate reader and writer into a ReadWriteCloser.
type rwc struct {
	reader io.Reader
	writer io.Writer
	closer func() error
}

func (r rwc) Read(p []byte) (int, error)  { return r.reader.Read(p) }
func (r rwc) Write(p []byte) (int, error) { return r.writer.Write(p) }
func (r rwc) Close() error {
	if r.closer != nil {
		return r.closer()
	}
	return nil
}
