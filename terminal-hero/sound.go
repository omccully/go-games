package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	wrongNoteSound, _, err := openAudioFile("wrong-note.wav")
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
	// for some reason vorbis.Decode accepts a ReadCloser instead of just a Reader
	// so we have to adapt the wav.Decode to be accept ReadCloser
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
	isAbsolutePath := filepath.IsAbs(filePath)
	var reader io.ReadCloser
	if isAbsolutePath {
		// if it's an absolute path, use the file system
		file, err := os.Open(filePath)
		if err != nil {
			return nil, beep.Format{}, err
		}
		reader = file
	} else {
		// if it's a relative path, check for an embedded resource
		resourcePath := filepath.Join("sounds", filePath)
		ef, err := readEmbeddedResourceFile(resourcePath)
		if err != nil {
			return nil, beep.Format{}, err
		}

		// vorbis.Decode accepts a ReadCloser so we need to adapt this
		// to have a Close method
		reader = &ClosableBuffer{bytes.NewBuffer(ef)}
	}

	streamer, format, err := decoder(reader)
	if err != nil {
		return nil, beep.Format{}, err
	}

	defer streamer.Close()

	bufferedStreamer := bufferStreamer(streamer, format)
	return bufferedStreamer, format, nil
}

func bufferStreamer(streamer beep.StreamSeeker, format beep.Format) beep.StreamSeeker {
	buffer := beep.NewBuffer(format)
	buffer.Append(streamer)
	return buffer.Streamer(0, buffer.Len())
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

type ClosableBuffer struct {
	*bytes.Buffer
}

func (cb *ClosableBuffer) Close() error {
	return nil
}
