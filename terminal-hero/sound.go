package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/faiface/beep"
	"github.com/faiface/beep/effects"
	"github.com/faiface/beep/vorbis"
	"github.com/faiface/beep/wav"
)

type sound struct {
	soundStream beep.StreamSeeker
	format      beep.Format
	filePath    string
}

type soundEffects struct {
	wrongNote   sound
	initialized bool
}

type songSounds struct {
	guitar       beep.StreamSeeker
	song         beep.StreamSeeker
	bass         beep.StreamSeeker
	songFormat   beep.Format
	guitarVolume *effects.Volume
}

func loadSoundEffects() (soundEffects, error) {
	wrongNoteSound, _, err := openAudioFile("assets/sounds/wrong-note.wav")
	if err != nil {
		return soundEffects{}, err
	}

	return soundEffects{
		wrongNote: sound{
			soundStream: wrongNoteSound,
		},
		initialized: true,
	}, nil
}

func openAudioFile(filePath string) (beep.StreamSeeker, beep.Format, error) {
	if strings.HasSuffix(filePath, ".ogg") {
		return openBufferedOggAudioFile(filePath)
	} else if strings.HasSuffix(filePath, ".wav") {
		return openBufferedWavAudioFile(filePath)
	} else {
		return nil, beep.Format{}, fmt.Errorf("unknown file type: %s", filePath)
	}
}

func wavDecoder(rc io.ReadCloser) (beep.StreamSeekCloser, beep.Format, error) {
	ssc, f, err := wav.Decode(rc)
	return ssc, f, err
}

func openBufferedWavAudioFile(filePath string) (beep.StreamSeeker, beep.Format, error) {
	return openBufferedAudioFileG(filePath, wavDecoder)
}

func openBufferedOggAudioFile(filePath string) (beep.StreamSeeker, beep.Format, error) {
	return openBufferedAudioFileG(filePath, vorbis.Decode)
}

type decoderFunc func(rc io.ReadCloser) (beep.StreamSeekCloser, beep.Format, error)

func openBufferedAudioFileG(filePath string, decoder decoderFunc) (beep.StreamSeeker, beep.Format, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, beep.Format{}, err
	}
	streamer, format, err := decoder(file)
	if err != nil {
		return nil, beep.Format{}, err
	}

	buffer := beep.NewBuffer(format)
	buffer.Append(streamer)
	streamer.Close()
	bufferedStreamer := buffer.Streamer(0, buffer.Len())
	return bufferedStreamer, format, nil
}

func openOggAudioFile(filePath string) (beep.StreamSeekCloser, beep.Format, error) {
	return openAudioFileG(filePath, vorbis.Decode)
}

func openAudioFileG(filePath string, decoder decoderFunc) (beep.StreamSeekCloser, beep.Format, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, beep.Format{}, err
	}
	streamer, format, err := decoder(file)
	if err != nil {
		return nil, beep.Format{}, err
	}
	return streamer, format, nil
}

func closeSoundStreams(songSounds songSounds) {
	closeStreamSeeker(songSounds.guitar)
	closeStreamSeeker(songSounds.song)
	closeStreamSeeker(songSounds.bass)
}

func closeStreamSeeker(streamer beep.StreamSeeker) {
	closer, ok := streamer.(beep.StreamSeekCloser)
	if ok && closer != nil {
		closer.Close()
	}
}

func (s sound) close() {
	closeStreamSeeker(s.soundStream)
}
