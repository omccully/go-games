package main

import (
	"bytes"
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
	soundStream beep.Streamer
	format      beep.Format
	filePath    string
}

type soundEffects struct {
	wrongNote   playableSound[beep.StreamSeeker]
	initialized bool
}

type songSounds struct {
	guitar playableSound[*effects.Volume]
	song   playableSound[beep.StreamSeeker]
	bass   playableSound[beep.StreamSeeker]
}

func loadSoundEffects(spkr soundPlayer) (soundEffects, error) {
	wrongNoteSound, format, err := openAudioFileNonBuffered("wrong-note.wav")
	if err != nil {
		return soundEffects{}, err
	}

	buf := resampleIntoBuffer(spkr, wrongNoteSound, format)

	return soundEffects{
		wrongNote:   buf,
		initialized: true,
	}, nil
}

func getAudioDecoderForFile(filePath string) decoderFunc {
	if strings.HasSuffix(filePath, ".ogg") {
		return vorbis.Decode
	} else if strings.HasSuffix(filePath, ".wav") {
		return wavDecoder
	} else {
		return nil
	}
}

func openAudioFileNonBuffered(filePath string) (beep.StreamSeeker, beep.Format, error) {
	decoder := getAudioDecoderForFile(filePath)
	return openAudioFileG(filePath, decoder)
}

func wavDecoder(rc io.ReadCloser) (beep.StreamSeekCloser, beep.Format, error) {
	// for some reason vorbis.Decode accepts a ReadCloser instead of just a Reader
	// so we have to adapt the wav.Decode to be accept ReadCloser
	ssc, f, err := wav.Decode(rc)
	return ssc, f, err
}

type decoderFunc func(rc io.ReadCloser) (beep.StreamSeekCloser, beep.Format, error)

func openAudioFileReader(filePath string) (io.ReadCloser, error) {
	isAbsolutePath := filepath.IsAbs(filePath)
	var reader io.ReadCloser
	if isAbsolutePath {
		// if it's an absolute path, use the file system
		// the file gets closed when the beep.StreamSeekCloser is closed
		file, err := os.Open(filePath)
		if err != nil {
			return nil, err
		}
		reader = file
	} else {
		// if it's a relative path, check for an embedded resource
		resourcePath := filepath.Join("sounds", filePath)
		ef, err := readEmbeddedResourceFile(resourcePath)
		if err != nil {
			return nil, err
		}

		// vorbis.Decode accepts a ReadCloser so we need to adapt this
		// to have a Close method
		reader = &ClosableBuffer{bytes.NewBuffer(ef)}
	}
	return reader, nil
}

func openAudioFileG(filePath string, decoder decoderFunc) (beep.StreamSeekCloser, beep.Format, error) {
	fileReader, err := openAudioFileReader(filePath)
	if err != nil {
		return nil, beep.Format{}, err
	}

	return decoder(fileReader)
}

func bufferStreamer(streamer beep.Streamer, format beep.Format) beep.StreamSeeker {
	buffer := beep.NewBuffer(format)
	buffer.Append(streamer)
	return buffer.Streamer(0, buffer.Len())
}

func closeStreamSeeker(streamer beep.Streamer) {
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
