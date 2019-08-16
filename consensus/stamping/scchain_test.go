package stamping

import (
	"math/rand"
	"testing"
)

type block struct {
	header *Header
	fc     *FinalCertificate
}

const (
	newBlockEvent = 1
)

type event struct {
	Height uint64
	Type   uint
}

func blockGenerator(t *testing.T, chain *Chain, maxHeight uint64, eventCh chan<- event) {
	parent := genesisHeader
	for height := uint64(1); height <= maxHeight; height++ {
		header := NewHeader(height, parent)
		fc := NewFinalCertificate(height, parent)
		parent = header

		err := chain.AddBlock(header, fc)
		if err != nil {
			t.Errorf("AddBlock failed, height=%d, err=%v", header.Height, err)
		}

		eventCh <- event{Height: header.Height, Type: newBlockEvent}
	}
	close(eventCh)
}

func makeStampingGenerator(config *Config, chain *Chain, eventCh <-chan event) <-chan *StampingCertificate {
	ch := make(chan *StampingCertificate)
	go func() {
		for e := range eventCh {
			if e.Height <= config.B {
				continue
			}

			proofHeader := chain.Header(e.Height - config.B)
			if rand.Intn(100) < config.Probability {
				s := NewStampingCertificate(e.Height, proofHeader)
				ch <- s
			}
		}
		close(ch)
	}()
	return ch
}

func buildChain(t *testing.T, maxHeight uint64) *Chain {
	chain := NewChain()

	eventCh := make(chan event, 100)
	go blockGenerator(t, chain, maxHeight, eventCh)
	stampingCh := makeStampingGenerator(defaultConfig, chain, eventCh)

	for s := range stampingCh {
		err := chain.AddStampingCertificate(s)
		if err != nil {
			t.Errorf("AddStampingCertificate failed, height=%d, err=%v", s.Height, err)
			return nil
		}
	}

	return chain
}

func TestNewChain(t *testing.T) {
	const maxHeight = 100000
	chain := buildChain(t, maxHeight)
	chain.Print()
}

func TestSyncChain(t *testing.T) {
	const maxHeight = 10000
	other := buildChain(t, maxHeight)
	other.Print()

	t.Log("---------------------------------after sync-----------------------------------------------------")
	chain := NewChain()
	if err := chain.Sync(other); err != nil {
		t.Errorf("sync error, err:%s", err)
	}
	chain.Print()
}
