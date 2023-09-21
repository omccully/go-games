package com.omccully.midtochart;

import java.util.Scanner;
import java.util.ArrayList;
import javax.sound.midi.Track;
import javax.sound.midi.Sequencer;
import java.io.PrintWriter;
import javax.sound.midi.MidiSystem;
import java.io.File;
import javax.sound.midi.Sequence;

// 
// Decompiled by Procyon v0.5.36
// From Chart2Mid2Chart
// 

public class Mid2Chart
{
    private static String source;
    private static String XSingle;
    private static String HSingle;
    private static String MSingle;
    private static String ESingle;
    private static String Sync;
    private static String Events;
    private static String XLead;
    private static String HLead;
    private static String MLead;
    private static String ELead;
    private static String XBass;
    private static String HBass;
    private static String MBass;
    private static String EBass;
    private static String XDrums;
    private static String HDrums;
    private static String MDrums;
    private static String EDrums;
    private static String Header;
    private static double scaler;
    private static String chartName;
    private static String coop;
    private static boolean valid;
    private static boolean hasEvents;
    private static Sequence midi;
    
    public Mid2Chart(final String src) {
        Mid2Chart.source = src;
    }
    
    public String convert() throws Exception {
        initialize();
        String notices = "";
        notices = notices + Mid2Chart.source.substring(Mid2Chart.source.lastIndexOf("\\") + 1, Mid2Chart.source.length()) + ":\n";
        final File inFile = new File(Mid2Chart.source);
        Sequencer sequencer = null;
        final Sequence sequence = null;
        try {
            sequencer = MidiSystem.getSequencer();
            Mid2Chart.midi = MidiSystem.getSequence(inFile);
            sequencer.setSequence(sequence);
        }
        catch (Exception e) {
            e.printStackTrace();
            return notices + "Unknown Error: Could not read MIDI sequence.\n\n----------\n";
        }
        final Track[] trackArr = Mid2Chart.midi.getTracks();
        boolean named = false;
        int j = 0;
        String name = "";
        for (int i = 0; i < trackArr.length; ++i) {
            for (Track track = trackArr[i]; !named && j < track.size(); ++j) {
                final byte[] event = track.get(j).getMessage().getMessage();
                if (event[0] == -1 && event[1] == 3) {
                    for (int k = 3; k < event.length; ++k) {
                        name += (char)event[k];
                    }
                    named = true;
                }
            }
            if (name.equals("PART GUITAR")) {
                Mid2Chart.valid = true;
            }
            if (name.equals("EVENTS")) {
                Mid2Chart.hasEvents = true;
            }
        }
        if (!Mid2Chart.valid) {
            return notices + "PART GUITAR not found. No chart created.\n\n--------------\n";
        }
        Mid2Chart.scaler = 192.0 / Mid2Chart.midi.getResolution();
        System.out.println("Resolution = " + Mid2Chart.midi.getResolution() + "\n");
        System.out.println("Scaler = " + Mid2Chart.scaler + "\n");
        notices = notices + "NumTracks = " + trackArr.length + "\n";
        writeSync(trackArr[0]);
        Mid2Chart.Header = Mid2Chart.Header + "\tName = " + Mid2Chart.chartName + "\n";
        Mid2Chart.Header += "\tOffset = 0\n";
        Mid2Chart.Header += "\tResolution = 192\n";
        for (int i = 1; i < trackArr.length; ++i) {
            final Track track = trackArr[i];
            named = false;
            j = 0;
            name = "";
            while (!named && j < track.size()) {
                final byte[] event = track.get(j).getMessage().getMessage();
                if (event[0] == -1 && event[1] == 3) {
                    for (int k = 3; k < event.length; ++k) {
                        name += (char)event[k];
                    }
                    named = true;
                }
                ++j;
            }
            if (named) {
                if (name.equals("PART GUITAR")) {
                    writeNoteSection(track, 0);
                    Mid2Chart.valid = true;
                }
                else if (name.equals("PART GUITAR COOP")) {
                    writeNoteSection(track, 1);
                }
                else if (name.equals("PART RHYTHM")) {
                    Mid2Chart.coop = "rhythm";
                    writeNoteSection(track, 3);
                }
                else if (name.equals("PART BASS")) {
                    writeNoteSection(track, 3);
                }
                else if (name.equals("EVENTS")) {
                    writeNoteSection(track, 4);
                }
                else if (name.equals("PART DRUMS")) {
                    writeNoteSection(track, 5);
                }
                else {
                    notices = notices + "Track " + i + " (" + name + ") ignored.\n";
                }
            }
            else {
                notices = notices + "Track " + i + " ignored.\n";
            }
        }
        Mid2Chart.Header = Mid2Chart.Header + "\tPlayer2 = " + Mid2Chart.coop + "\n";
        final String line = "}\n";
        Mid2Chart.Header += line;
        Mid2Chart.XSingle += line;
        Mid2Chart.HSingle += line;
        Mid2Chart.MSingle += line;
        Mid2Chart.ESingle += line;
        Mid2Chart.XLead += line;
        Mid2Chart.HLead += line;
        Mid2Chart.MLead += line;
        Mid2Chart.ELead += line;
        Mid2Chart.XBass += line;
        Mid2Chart.HBass += line;
        Mid2Chart.MBass += line;
        Mid2Chart.EBass += line;
        Mid2Chart.XDrums += line;
        Mid2Chart.HDrums += line;
        Mid2Chart.MDrums += line;
        Mid2Chart.EDrums += line;
        Mid2Chart.Events += line;
        String chartPath = Mid2Chart.source.substring(0, Mid2Chart.source.length() - 4);
        chartPath += ".chart";
        final PrintWriter out = new PrintWriter(new File(chartPath));
        writeChart(out);
        out.close();
        notices += "Conversion Complete!\n";
        notices += "\n---------------------\n";
        return notices;
    }
    
    private static void writeSync(final Track track) {
        Mid2Chart.Sync += "[SyncTrack]\n{\n";
        long tick = 0L;
        byte[] event = null;
        for (int i = 0; i < track.size(); ++i) {
            tick = track.get(i).getTick();
            //System.out.println("Tick = " + tick + "\n");
            //System.out.println("Scaler = " + Mid2Chart.scaler + "\n");
            tick = (long)(tick * Mid2Chart.scaler);
            //System.out.println("TickAfter = " + tick + "\n");
            //System.console().readLine("enter to continue");
            event = track.get(i).getMessage().getMessage();
            final int type = event[1];
            if (type == 3) {
                String text = "";
                for (int j = 3; j < event.length; ++j) {
                    text += (char)event[j];
                }
                Mid2Chart.chartName = "\"" + text + "\"";
            }
            else if (type == 81) {
                final byte[] data = { event[3], event[4], event[5] };
                final int mpq = byteArrayToInt(data, 0);
                final int bpm = (int)Math.floor(6.0E7 / mpq * 1000.0);
                Mid2Chart.Sync = Mid2Chart.Sync + "\t" + tick + " = B " + bpm + "\n";
            }
            else if (type == 88) {
                final int num = event[3];
                Mid2Chart.Sync = Mid2Chart.Sync + "\t" + tick + " = TS " + num + "\n";
            }
            else if (type == 6 && !Mid2Chart.hasEvents) {
                String text = "section ";
                for (int j = 3; j < event.length; ++j) {
                    text += (char)event[j];
                }
                writeEventLine(4, tick, text);
            }
        }
        Mid2Chart.Sync += "}\n";
    }
    
    private static void writeNoteSection(final Track track, final int sec) {
        final boolean[] skip = new boolean[track.size()];
        for (int i = 0; i < skip.length; ++i) {
            skip[i] = false;
        }
        for (int i = 0; i < track.size(); ++i) {
            if (!skip[i]) {
                final byte[] event = track.get(i).getMessage().getMessage();
                long tick = track.get(i).getTick();
                //System.out.println("Tick = " + tick + "\n");
                tick = (long)(tick * Mid2Chart.scaler);
                final String line = "";
                int type = event[0] & 0xFF;
                if (type >= 144 && type <= 159) {
                    final int note = event[1];
                    long off = -1L;
                    for (int j = i + 1; off < 0L && j != track.size(); ++j) {
                        final byte[] e = track.get(j).getMessage().getMessage();
                        type = (e[0] & 0xFF);
                        if (e[1] == note) {
                            if (type >= 128 && type <= 143) {
                                off = track.get(j).getTick();
                                //System.out.println("offt1 = " + off);
                                off = (long)(off * Mid2Chart.scaler);
                            }
                            else if (type >= 144 && type <= 159) {
                                off = track.get(j).getTick();
                                //System.out.println("offt2 = " + off);
                                off = (long)(off * Mid2Chart.scaler);
                                skip[j] = true;
                            }
                        }
                    }
                    long sus = off - tick;
                    if (sus < 96L) {
                        sus = 0L;
                    }
                    //System.out.println(sus + " = " + off + " - " + tick + "\n");
                    //System.console().readLine();
                    writeNoteLine(sec, tick, note, sus);
                }
                else if (event[0] == 255 && event[1] == 1) {
                    final ArrayList validEvents = loadEvents(sec - 4);
                    String text = "";
                    for (int k = 3; k < event.length; ++k) {
                        text += (char)event[k];
                    }
                    if (validEvents.contains(text) || text.contains("[section ")) {
                        text = text.substring(1, text.length() - 1);
                        writeEventLine(sec, tick, text);
                    }
                }
            }
        }
    }
    
    private static void writeNoteLine(final int sec, final long tick, final int note, final long sus) {
        final int n = note % 12;
        String line = "";
        if (n >= 0 && n <= 4) {
            line = "\t" + tick + " = N " + n + " " + sus + "\n";
        }
        else if (n == 7) {
            line = "\t" + tick + " = S 2 " + sus + "\n";
        }
        else if (n == 9) {
            line = "\t" + tick + " = S 0 " + sus + "\n";
        }
        else {
            if (n != 10) {
                return;
            }
            line = "\t" + tick + " = S 1 " + sus + "\n";
        }
        String diff = "";
        if (note >= 60) {
            diff = "Easy";
        }
        if (note >= 72) {
            diff = "Medium";
        }
        if (note >= 84) {
            diff = "Hard";
        }
        if (note >= 96) {
            diff = "Expert";
        }
        if (diff.equals("Expert")) {
            if (sec == 0) {
                Mid2Chart.XSingle += line;
            }
            else if (sec == 1) {
                Mid2Chart.XLead += line;
            }
            else if (sec == 3) {
                Mid2Chart.XBass += line;
            }
            else if (sec == 5) {
                Mid2Chart.XDrums += line;
            }
        }
        else if (diff.equals("Hard")) {
            if (sec == 0) {
                Mid2Chart.HSingle += line;
            }
            else if (sec == 1) {
                Mid2Chart.HLead += line;
            }
            else if (sec == 3) {
                Mid2Chart.HBass += line;
            }
            else if (sec == 5) {
                Mid2Chart.HDrums += line;
            }
        }
        else if (diff.equals("Medium")) {
            if (sec == 0) {
                Mid2Chart.MSingle += line;
            }
            else if (sec == 1) {
                Mid2Chart.MLead += line;
            }
            else if (sec == 3) {
                Mid2Chart.MBass += line;
            }
            else if (sec == 5) {
                Mid2Chart.MDrums += line;
            }
        }
        else if (diff.equals("Easy")) {
            if (sec == 0) {
                Mid2Chart.ESingle += line;
            }
            else if (sec == 1) {
                Mid2Chart.ELead += line;
            }
            else if (sec == 3) {
                Mid2Chart.EBass += line;
            }
            else if (sec == 5) {
                Mid2Chart.EDrums += line;
            }
        }
    }
    
    private static void writeEventLine(final int sec, final long tick, String event) {
        if (sec == 4) {
            event = "\"" + event + "\"";
        }
        final String line = "\t" + tick + " = E " + event + "\n";
        if (sec == 4) {
            Mid2Chart.Events += line;
        }
        else if (sec == 0) {
            Mid2Chart.XSingle += line;
            Mid2Chart.HSingle += line;
            Mid2Chart.MSingle += line;
            Mid2Chart.ESingle += line;
        }
        else if (sec == 1) {
            Mid2Chart.XLead += line;
            Mid2Chart.HLead += line;
            Mid2Chart.MLead += line;
            Mid2Chart.ELead += line;
        }
        else if (sec == 3) {
            Mid2Chart.XBass += line;
            Mid2Chart.HBass += line;
            Mid2Chart.MBass += line;
            Mid2Chart.EBass += line;
        }
    }
    
    private static void writeChart(final PrintWriter out) {
        Scanner read = new Scanner(Mid2Chart.Header);
        read.useDelimiter("\n");
        while (read.hasNext()) {
            out.println(read.next());
        }
        out.flush();
        read = new Scanner(Mid2Chart.Sync);
        read.useDelimiter("\n");
        while (read.hasNext()) {
            out.println(read.next());
        }
        out.flush();
        read = new Scanner(Mid2Chart.Events);
        read.useDelimiter("\n");
        while (read.hasNext()) {
            out.println(read.next());
        }
        out.flush();
        read = new Scanner(Mid2Chart.XSingle);
        read.useDelimiter("\n");
        while (read.hasNext()) {
            out.println(read.next());
        }
        out.flush();
        read = new Scanner(Mid2Chart.HSingle);
        read.useDelimiter("\n");
        while (read.hasNext()) {
            out.println(read.next());
        }
        out.flush();
        read = new Scanner(Mid2Chart.MSingle);
        read.useDelimiter("\n");
        while (read.hasNext()) {
            out.println(read.next());
        }
        out.flush();
        read = new Scanner(Mid2Chart.ESingle);
        read.useDelimiter("\n");
        while (read.hasNext()) {
            out.println(read.next());
        }
        out.flush();
        read = new Scanner(Mid2Chart.XLead);
        read.useDelimiter("\n");
        while (read.hasNext()) {
            out.println(read.next());
        }
        out.flush();
        read = new Scanner(Mid2Chart.HLead);
        read.useDelimiter("\n");
        while (read.hasNext()) {
            out.println(read.next());
        }
        out.flush();
        read = new Scanner(Mid2Chart.MLead);
        read.useDelimiter("\n");
        while (read.hasNext()) {
            out.println(read.next());
        }
        out.flush();
        read = new Scanner(Mid2Chart.ELead);
        read.useDelimiter("\n");
        while (read.hasNext()) {
            out.println(read.next());
        }
        out.flush();
        read = new Scanner(Mid2Chart.XBass);
        read.useDelimiter("\n");
        while (read.hasNext()) {
            out.println(read.next());
        }
        out.flush();
        read = new Scanner(Mid2Chart.HBass);
        read.useDelimiter("\n");
        while (read.hasNext()) {
            out.println(read.next());
        }
        out.flush();
        read = new Scanner(Mid2Chart.MBass);
        read.useDelimiter("\n");
        while (read.hasNext()) {
            out.println(read.next());
        }
        out.flush();
        read = new Scanner(Mid2Chart.EBass);
        read.useDelimiter("\n");
        while (read.hasNext()) {
            out.println(read.next());
        }
        out.flush();
        read = new Scanner(Mid2Chart.XDrums);
        read.useDelimiter("\n");
        while (read.hasNext()) {
            out.println(read.next());
        }
        out.flush();
        read = new Scanner(Mid2Chart.HDrums);
        read.useDelimiter("\n");
        while (read.hasNext()) {
            out.println(read.next());
        }
        out.flush();
        read = new Scanner(Mid2Chart.MDrums);
        read.useDelimiter("\n");
        while (read.hasNext()) {
            out.println(read.next());
        }
        out.flush();
        read = new Scanner(Mid2Chart.EDrums);
        read.useDelimiter("\n");
        while (read.hasNext()) {
            out.println(read.next());
        }
        out.flush();
    }
    
    private static void initialize() {
        Mid2Chart.XSingle = "[ExpertSingle]\n{\n";
        Mid2Chart.HSingle = "[HardSingle]\n{\n";
        Mid2Chart.MSingle = "[MediumSingle]\n{\n";
        Mid2Chart.ESingle = "[EasySingle]\n{\n";
        Mid2Chart.Sync = "";
        Mid2Chart.Events = "[Events]\n{\n";
        Mid2Chart.XLead = "[ExpertDoubleGuitar]\n{\n";
        Mid2Chart.HLead = "[HardDoubleGuitar]\n{\n";
        Mid2Chart.MLead = "[MediumDoubleGuitar]\n{\n";
        Mid2Chart.ELead = "[EasyDoubleGuitar]\n{\n";
        Mid2Chart.XBass = "[ExpertDoubleBass]\n{\n";
        Mid2Chart.HBass = "[HardDoubleBass]\n{\n";
        Mid2Chart.MBass = "[MediumDoubleBass]\n{\n";
        Mid2Chart.EBass = "[EasyDoubleBass]\n{\n";
        Mid2Chart.XDrums = "[ExpertDrums]\n{\n";
        Mid2Chart.HDrums = "[HardDrums]\n{\n";
        Mid2Chart.MDrums = "[MediumDrums]\n{\n";
        Mid2Chart.EDrums = "[EasyDrums]\n{\n";
        Mid2Chart.Header = "[Song]\n{\n";
        Mid2Chart.chartName = "";
        Mid2Chart.coop = "bass";
        Mid2Chart.valid = true;
    }
    
    private static int byteArrayToInt(final byte[] b, final int offset) {
        int value = 0;
        value += (b[0] & 0xFF) << 16;
        value += (b[1] & 0xFF) << 8;
        value += (b[2] & 0xFF);
        return value;
    }
    
    private static ArrayList loadEvents(final int track) {
        final ArrayList arr = new ArrayList();
        if (track == 0) {
            arr.add("[lighting (chase)]");
            arr.add("[lighting (strobe)]");
            arr.add("[lighting (color1)]");
            arr.add("[lighting (color2)]");
            arr.add("[lighting (sweep)]");
            arr.add("[crowd_lighters_fast]");
            arr.add("[crowd_lighters_off]");
            arr.add("[crowd_lighters_slow]");
            arr.add("[crowd_half_tempo]");
            arr.add("[crowd_normal_tempo]");
            arr.add("[crowd_double_tempo]");
            arr.add("[band_jump]");
            arr.add("[sync_head_bang]");
            arr.add("[sync_wag]");
            arr.add("[lighting ()]");
            arr.add("[lighting (flare)]");
            arr.add("[lighting (blackout)]");
            arr.add("[music_start]");
            arr.add("[verse]");
            arr.add("[chorus]");
            arr.add("[solo]");
            arr.add("[end]");
            return arr;
        }
        arr.add("[idle]");
        arr.add("[play]");
        arr.add("[solo_on]");
        arr.add("[solo_off]");
        arr.add("[wail_on]");
        arr.add("[wail_off]");
        arr.add("[ow_face_on]");
        arr.add("[ow_face_off]");
        arr.add("[half_tempo]");
        arr.add("[normal_tempo]");
        return arr;
    }
    
    static {
        Mid2Chart.XSingle = "[ExpertSingle]\n{\n";
        Mid2Chart.HSingle = "[HardSingle]\n{\n";
        Mid2Chart.MSingle = "[MediumSingle]\n{\n";
        Mid2Chart.ESingle = "[EasySingle]\n{\n";
        Mid2Chart.Sync = "";
        Mid2Chart.Events = "[Events]\n{\n";
        Mid2Chart.XLead = "[ExpertDoubleGuitar]\n{\n";
        Mid2Chart.HLead = "[HardDoubleGuitar]\n{\n";
        Mid2Chart.MLead = "[MediumDoubleGuitar]\n{\n";
        Mid2Chart.ELead = "[EasyDoubleGuitar]\n{\n";
        Mid2Chart.XBass = "[ExpertDoubleBass]\n{\n";
        Mid2Chart.HBass = "[HardDoubleBass]\n{\n";
        Mid2Chart.MBass = "[MediumDoubleBass]\n{\n";
        Mid2Chart.EBass = "[EasyDoubleBass]\n{\n";
        Mid2Chart.XDrums = "[ExpertDrums]\n{\n";
        Mid2Chart.HDrums = "[HardDrums]\n{\n";
        Mid2Chart.MDrums = "[MediumDrums]\n{\n";
        Mid2Chart.EDrums = "[EasyDrums]\n{\n";
        Mid2Chart.Header = "[Song]\n{\n";
        Mid2Chart.chartName = "";
        Mid2Chart.coop = "bass";
        Mid2Chart.valid = true;
        Mid2Chart.hasEvents = false;
    }
}