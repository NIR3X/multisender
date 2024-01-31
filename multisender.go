package multisender

import (
	"io"
	"sync"
	"time"

	"github.com/NIR3X/filecache"
)

type MultiSenderWriter struct {
	mtx     sync.Mutex
	wg      sync.WaitGroup
	writers []io.Writer
}

func NewMultiSenderWriter() *MultiSenderWriter {
	return &MultiSenderWriter{
		mtx:     sync.Mutex{},
		wg:      sync.WaitGroup{},
		writers: make([]io.Writer, 0),
	}
}

func (m *MultiSenderWriter) Wait() {
	m.wg.Wait()
}

func (m *MultiSenderWriter) Write(p []uint8) (n int, err error) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	for i := 0; i < len(m.writers); i++ {
		w := m.writers[i]
		_, err = w.Write(p)
		if err != nil {
			lastIndex := len(m.writers) - 1
			m.writers[i] = m.writers[lastIndex]
			m.writers = m.writers[:lastIndex]
			i--
		}
	}
	return len(p), nil
}

type MultiSender struct {
	mtx              sync.Mutex
	accumulationTime time.Duration
	fileCache        *filecache.FileCache
	writers          map[string]*MultiSenderWriter
}

func NewMultiSender(fileCache *filecache.FileCache, accumulationTime ...time.Duration) *MultiSender {
	if len(accumulationTime) == 0 {
		accumulationTime = []time.Duration{1 * time.Second}
	}
	return &MultiSender{
		mtx:              sync.Mutex{},
		accumulationTime: accumulationTime[0],
		fileCache:        fileCache,
		writers:          make(map[string]*MultiSenderWriter),
	}
}

func (m *MultiSender) Add(path string, w io.Writer) *MultiSenderWriter {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	if _, ok := m.writers[path]; !ok {
		writer := NewMultiSenderWriter()
		writer.wg.Add(1)
		m.writers[path] = writer
		accumulationTime := m.accumulationTime
		go func() {
			time.Sleep(accumulationTime)
			m.mtx.Lock()
			delete(m.writers, path)
			r, pw, err := m.fileCache.Get(path)
			m.mtx.Unlock()
			if err == nil {
				if pw != nil {
					defer pw.Close()
				}
				_, _ = io.Copy(writer, r)
			}
			writer.wg.Done()
		}()
	}

	multiSenderWriter := m.writers[path]
	multiSenderWriter.mtx.Lock()
	defer multiSenderWriter.mtx.Unlock()
	multiSenderWriter.writers = append(multiSenderWriter.writers, w)
	return multiSenderWriter
}
