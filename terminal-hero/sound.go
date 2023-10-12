package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/faiface/beep"
	"github.com/faiface/beep/effects"
	"github.com/faiface/beep/wav"
)

const (
	instrumentGuitar = "Guitar"
	instrumentBass   = "Bass"
	instrumentDrums  = "Drums"
	instrumentMisc   = "Misc"
)

var instrumentSoundFiles = map[string]*regexp.Regexp{
	instrumentDrums:  takeRegex(regexp.Compile(`^drums(_[0-9]+)?\.`)),
	instrumentGuitar: takeRegex(regexp.Compile(`^guitar\.`)),
	instrumentBass:   takeRegex(regexp.Compile(`^(bass|rhythm)\.`)),
	instrumentMisc:   takeRegex(regexp.Compile(`^(song|vocals|keys)\.`)),
}

func isMatchingInstrumentSoundFile(instrument string, fileName string) bool {
	regex, ok := instrumentSoundFiles[instrument]
	if !ok {
		return false
	}
	return regex.MatchString(fileName)
}

func takeRegex(r *regexp.Regexp, err error) *regexp.Regexp {
	if err != nil {
		panic(err)
	}
	return r
}

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
	bass   playableSound[*effects.Volume]
	drums  playableSound[*effects.Volume]
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
		return DecodeVorbis
	} else if strings.HasSuffix(filePath, ".wav") {
		return wavDecoder
	} else {
		return nil
	}
}

type audioFileOpener interface {
	openAudioFile(filePath string) (beep.StreamSeekCloser, beep.Format, error)
}

type audioFileOpen int

func (afo audioFileOpen) openAudioFile(filePath string) (beep.StreamSeekCloser, beep.Format, error) {
	return openAudioFileNonBuffered(filePath)
}

func openAudioFileNonBuffered(filePath string) (beep.StreamSeekCloser, beep.Format, error) {
	decoder := getAudioDecoderForFile(filePath)
	if decoder == nil {
		return nil, beep.Format{}, errors.New("Unsupported file type: " + filePath)
	}
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
