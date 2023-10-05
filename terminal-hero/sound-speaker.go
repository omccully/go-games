package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
)

type thSpeaker struct {
	initialized bool
	format      beep.Format
	mu          sync.Mutex
}

type playableSound[T beep.Streamer] struct {
	soundStream T
	format      beep.Format
}

type soundPlayer interface {
	play(snd sound)
}

func (spkr *thSpeaker) init(format beep.Format) {
	bufSize := format.SampleRate.N(time.Second / 10)
	log.Info(fmt.Sprintf("Initializing speaker %d,%d", format.SampleRate, bufSize))
	speaker.Init(format.SampleRate, bufSize)
	spkr.initialized = true
	spkr.format = format
}

func (spkr *thSpeaker) play(stream beep.Streamer, format beep.Format) {
	spkr.mu.Lock()
	defer spkr.mu.Unlock()

	if !spkr.initialized {
		spkr.init(format)
	} else {
		if format.SampleRate != spkr.format.SampleRate {
			sound := spkr.resampleIfNeeded(stream, format)
			stream = sound.soundStream
		} else {
			log.Info(fmt.Sprintf("No resampling needed from %d to %d", format.SampleRate, spkr.format.SampleRate))
		}
	}

	speaker.Play(stream)
}

func (spkr *thSpeaker) resampleIfNeeded(stream beep.Streamer, oldFormat beep.Format) playableSound[beep.Streamer] {
	spkr.mu.Lock()
	defer spkr.mu.Unlock()

	result := stream
	if !spkr.initialized {
		spkr.format = oldFormat
		spkr.initialized = true
	} else if oldFormat.SampleRate != spkr.format.SampleRate {
		log.Info(fmt.Sprintf("Resampling from %d to %d", oldFormat.SampleRate, spkr.format.SampleRate))
		result = beep.Resample(4, oldFormat.SampleRate, spkr.format.SampleRate, stream)
	} else {
		log.Info(fmt.Sprintf("No resampling needed from %d to %d", oldFormat.SampleRate, spkr.format.SampleRate))
	}

	return playableSound[beep.Streamer]{
		soundStream: result,
		format:      spkr.format,
	}
}

func (spkr *thSpeaker) resampleIntoBuffer(stream beep.Streamer, oldFormat beep.Format) playableSound[beep.StreamSeeker] {
	result := spkr.resampleIfNeeded(stream, oldFormat)
	buffered := bufferStreamer(result.soundStream, spkr.format)

	closeStreamSeeker(stream)

	return playableSound[beep.StreamSeeker]{
		soundStream: buffered,
		format:      spkr.format,
	}
}

func (spkr *thSpeaker) clear() {
	speaker.Clear()
}
