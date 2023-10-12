package main

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/faiface/beep"
	"github.com/pkg/errors"
)

type fakeSoundPlayer struct {
	playedSounds []fakePlayedSound
	mu           sync.Mutex
}

type fakePlayedSound struct {
	samples [][2]float64
	num     int
}

func (s *fakeSoundPlayer) play(stream beep.Streamer, format beep.Format) {
	s.mu.Lock()
	defer s.mu.Unlock()
	samples := make([][2]float64, 100)
	num, _ := stream.Stream(samples)
	s.playedSounds = append(s.playedSounds, fakePlayedSound{samples: samples[:num], num: num})
	fmt.Printf("Played %d samples\n", num)
}

func (s *fakeSoundPlayer) resampleIfNeeded(stream beep.Streamer, oldFormat beep.Format) playableSound[beep.Streamer] {
	return playableSound[beep.Streamer]{stream, oldFormat}
}

func (s *fakeSoundPlayer) clear() {
	s.playedSounds = make([]fakePlayedSound, 0)
}

type fakeAudioStreamer struct {
	f    beep.Format
	data []byte
	pos  int
}

func (bs *fakeAudioStreamer) Stream(samples [][2]float64) (n int, ok bool) {
	if bs.pos >= len(bs.data) {
		return 0, false
	}
	for i := range samples {
		if bs.pos >= len(bs.data) {
			break
		}

		samples[i] = [2]float64{float64(bs.data[bs.pos]), float64(bs.data[bs.pos+1])}
		bs.pos += 2
		n++
	}
	return n, true
}

func (bs *fakeAudioStreamer) Err() error {
	return nil
}

func (bs *fakeAudioStreamer) Len() int {
	return len(bs.data) / bs.f.Width()
}

func (bs *fakeAudioStreamer) Position() int {
	return bs.pos / bs.f.Width()
}

func (bs *fakeAudioStreamer) Seek(p int) error {
	if p < 0 || bs.Len() < p {
		return fmt.Errorf("buffer: seek position %v out of range [%v, %v]", p, 0, bs.Len())
	}
	bs.pos = p * bs.f.Width()
	return nil
}

func (bs *fakeAudioStreamer) Close() error {
	return nil
}

type fakeAudioFileOpen struct {
	expectedFilePath string
	defaultData      []byte
	err              error
}

func (afo *fakeAudioFileOpen) openAudioFile(filePath string) (beep.StreamSeekCloser, beep.Format, error) {
	filePath = strings.Replace(filePath, "\\", "/", -1)
	if filePath != afo.expectedFilePath {
		afo.err = errors.New(fmt.Sprintf("Expected file path %s, got %s", afo.expectedFilePath, filePath))
	}

	fmt := beep.Format{SampleRate: beep.SampleRate(44100), NumChannels: 2, Precision: 1}
	streamer := fakeAudioStreamer{f: fmt, data: afo.defaultData}
	return &streamer, fmt, nil
}

func execCmd(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
}

func sendCmd(tm *teatest.TestModel, cmd tea.Cmd) {
	msg := execCmd(cmd)
	if msg != nil {
		tm.Send(msg)
	}
}

// func

func samplesEqual(expected [][2]float64, actual [][2]float64) error {
	if len(expected) != len(actual) {
		return errors.New(fmt.Sprintf("Expected %d samples, got %d", len(expected), len(actual)))
	}
	for i := range expected {
		if expected[i] != actual[i] {
			return errors.New(fmt.Sprintf("Expected sample %d to be %v, got %v", i, expected[i], actual[i]))
		}
	}
	return nil
}

func expectSamplesEqual(t *testing.T, expected [][2]float64, actual [][2]float64) {
	err := samplesEqual(expected, actual)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSongList(t *testing.T) {
	speaker := fakeSoundPlayer{}
	audioOpener := fakeAudioFileOpen{defaultData: []byte{1, 2, 3, 4},
		expectedFilePath: "test/preview.ogg",
	}
	m := initialSelectSongListModel(&speaker, &audioOpener)

	f1 := &songFolder{
		name:   "test",
		path:   "test",
		isLeaf: true,
	}
	f2 := &songFolder{
		name:   "test2",
		path:   "test2",
		isLeaf: true,
	}
	m, cmd := m.setSongs([]*songFolder{
		f1, f2,
	}, nil)

	tm := teatest.NewTestModel(t, m)
	sendCmd(tm, cmd)
	t.Log("Waiting for sasmples")

	waitForErr := doWaitFor(func() (bool, error) {
		println("Checked\n")
		speaker.mu.Lock()
		defer speaker.mu.Unlock()
		if audioOpener.err != nil {
			return false, audioOpener.err
		}
		return len(speaker.playedSounds) == 1, nil
	})
	if waitForErr != nil {
		t.Fatal(errors.Wrap(waitForErr, fmt.Sprintf("Expected one sound got %d", len(speaker.playedSounds))))
	}
	expectSamplesEqual(t, [][2]float64{{1, 2}, {3, 4}}, speaker.playedSounds[0].samples)

	selected, ok := m.selectedItem()
	if !ok {
		t.Fatal("No selected item")
	}
	if selected != f1 {
		t.Fatal("Expected first item to be selected")
	}

	audioOpener.defaultData = []byte{5, 6, 7, 8}
	audioOpener.expectedFilePath = "test2/preview.ogg"
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("down")})

	waitForErr = doWaitFor(func() (bool, error) {
		println("Checked\n")
		speaker.mu.Lock()
		defer speaker.mu.Unlock()
		if audioOpener.err != nil {
			return false, audioOpener.err
		}
		if len(speaker.playedSounds) != 1 {
			return false, nil
		}
		err := samplesEqual([][2]float64{{5, 6}, {7, 8}}, speaker.playedSounds[0].samples)

		return err == nil, nil
	})
	if waitForErr != nil {
		t.Fatal(errors.Wrap(waitForErr, fmt.Sprintf("Expected one sound matching 5,6,7,8 got %d", len(speaker.playedSounds))))
	}

	tm.Quit()
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

}

func doWaitFor(condition func() (bool, error)) error {
	wf := teatest.WaitingForContext{
		Duration:      time.Second,
		CheckInterval: 25 * time.Millisecond, //nolint: gomnd
	}
	start := time.Now()
	for time.Since(start) <= wf.Duration {
		result, err := condition()
		if err != nil {
			return err
		}
		if result {
			return nil
		}
		time.Sleep(wf.CheckInterval)
	}
	return fmt.Errorf("WaitFor: condition not met after %s", wf.Duration)
}
