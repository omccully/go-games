package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
)

const (
	speakerNotInitialized speakerState = iota
	speakerSampleRateDecidedButNotInitialized
	speakerFullyInitialized
)

type speakerState int

type thSpeaker struct {
	state speakerState

	format beep.Format
	mu     sync.Mutex
}

type playableSound[T beep.Streamer] struct {
	soundStream T
	format      beep.Format
}

type soundPlayer interface {
	play(stream beep.Streamer, format beep.Format)
	resampleIfNeeded(stream beep.Streamer, oldFormat beep.Format) playableSound[beep.Streamer]
	clear()
}

func (spkr *thSpeaker) init(format beep.Format) {
	spkr.partialInit(format)
	spkr.finishInit()
}

// sets the format but does not initialize the speaker
func (spkr *thSpeaker) partialInit(format beep.Format) {
	if spkr.state != speakerNotInitialized {
		panic(fmt.Sprintf("Speaker already initialized in state %d", spkr.state))
	}

	spkr.format = format
	spkr.state = speakerSampleRateDecidedButNotInitialized
}

// initializes the speaker with the format set in partialInit
func (spkr *thSpeaker) finishInit() {
	if spkr.state != speakerSampleRateDecidedButNotInitialized {
		panic(fmt.Sprintf("Speaker not in correct state to finish init: %d", spkr.state))
	}

	bufSize := spkr.format.SampleRate.N(time.Second / 10)
	speaker.Init(spkr.format.SampleRate, bufSize)
	log.Info(fmt.Sprintf("Initialized speaker with format %d", spkr.format))
	spkr.state = speakerFullyInitialized
}

// plays sounds. should only be called from the main thread
func (spkr *thSpeaker) play(stream beep.Streamer, format beep.Format) {
	log.Info(fmt.Sprintf("thSpeaker.play: %d", format))
	spkr.mu.Lock()
	defer spkr.mu.Unlock()

	if spkr.state == speakerNotInitialized {
		// definitely doesn't need resampled because it's the first sound
		spkr.init(format)
	} else {
		// may need resampled
		if spkr.state == speakerSampleRateDecidedButNotInitialized {
			spkr.finishInit()
		}

		if format.SampleRate != spkr.format.SampleRate {
			sound := spkr.resampleIfNeeded(stream, format)
			stream = sound.soundStream
			log.Info(fmt.Sprintf("thSpeaker.play: Auto resampling %d to %d", format.SampleRate, spkr.format.SampleRate))
		} else {
			log.Info(fmt.Sprintf("No resampling needed from %d to %d", format.SampleRate, spkr.format.SampleRate))
		}
	}

	log.Info("playing sound")
	speaker.Play(stream)
}

func (spkr *thSpeaker) resampleIfNeeded(stream beep.Streamer, oldFormat beep.Format) playableSound[beep.Streamer] {
	log.Info("resampleIfNeeded %d to %d", oldFormat, spkr.format)
	spkr.mu.Lock()
	defer spkr.mu.Unlock()

	result := stream
	if spkr.state == speakerNotInitialized {
		spkr.partialInit(oldFormat)
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

func resampleIntoBuffer(spkr soundPlayer, stream beep.Streamer, oldFormat beep.Format) playableSound[beep.StreamSeeker] {
	log.Info("resampleIntoBuffer %d", oldFormat)

	result := spkr.resampleIfNeeded(stream, oldFormat)
	spkrFormat := result.format
	buffered := bufferStreamer(result.soundStream, spkrFormat)

	closeStreamSeeker(stream)

	return playableSound[beep.StreamSeeker]{
		soundStream: buffered,
		format:      spkrFormat,
	}
}

func (spkr *thSpeaker) clear() {
	speaker.Clear()
}
