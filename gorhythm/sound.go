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
		return openOggAudioFile(filePath)
	} else if strings.HasSuffix(filePath, ".wav") {
		return openWavAudioFile(filePath)
	} else {
		return nil, beep.Format{}, fmt.Errorf("unknown file type: %s", filePath)
	}
}

func wavDecoder(rc io.ReadCloser) (beep.StreamSeekCloser, beep.Format, error) {
	ssc, f, err := wav.Decode(rc)
	return ssc, f, err
}

func openWavAudioFile(filePath string) (beep.StreamSeeker, beep.Format, error) {
	return openAudioFileG(filePath, wavDecoder)
}

func openOggAudioFile(filePath string) (beep.StreamSeeker, beep.Format, error) {
	return openAudioFileG(filePath, vorbis.Decode)
}

func openAudioFileG(filePath string, decoder func(rc io.ReadCloser) (beep.StreamSeekCloser, beep.Format, error)) (beep.StreamSeeker, beep.Format, error) {
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

func closeSoundStreams(songSounds songSounds) {
	guitarCLoser, ok := songSounds.guitar.(beep.StreamSeekCloser)
	if ok && guitarCLoser != nil {
		guitarCLoser.Close()
	}

	songCLoser, ok := songSounds.song.(beep.StreamSeekCloser)
	if ok && songCLoser != nil {
		songCLoser.Close()
	}

	bassCloser, ok := songSounds.bass.(beep.StreamSeekCloser)
	if ok && bassCloser != nil {
		bassCloser.Close()
	}
}
