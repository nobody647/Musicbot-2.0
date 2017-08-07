/*
    __  ___   __  __   _____    ____   ______    ____    ____   ______          ___       ____
   /  |/  /  / / / /  / ___/   /  _/  / ____/   / __ )  / __ \ /_  __/         |__ \     / __ \
  / /|_/ /  / / / /   \__ \    / /   / /       / __  | / / / /  / /            __/ /    / / / /
 / /  / /  / /_/ /   ___/ /  _/ /   / /___    / /_/ / / /_/ /  / /            / __/  _ / /_/ /
/_/  /_/   \____/   /____/  /___/   \____/   /_____/  \____/  /_/            /____/ (_)\____/

	A project by Ian Flanagan
*/

package main

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	cowsay "github.com/Code-Hex/Neo-cowsay"
	"github.com/bwmarrin/discordgo"
	"google.golang.org/api/googleapi/transport"
	"google.golang.org/api/youtube/v3"
	"layeh.com/gopus"
)

var plm map[string]*server
var yt *youtube.Service
var discord discordgo.Session

const christianCowsay string = ` ______________________________________
< no wearing this is a christian sever >
 --------------------------------------
        \   ^__^
         \  (oo)\_______
            (__)\       )\/\
                ||----w |
                ||     ||`

func main() {
	discord2, _ := discordgo.New("Bot MTg5MTQ2MDg0NzE3NjI1MzQ0.DANL1A.4cLruFPliFxkd0r41pYB307_D1M")
	discord = *discord2
	discord.Open()
	discord.AddHandler(messageCreate)
	plm = make(map[string]*server)
	client := &http.Client{
		Transport: &transport.APIKey{Key: "AIzaSyBTYNvJ80kHSE8AypP7Yst5Fshc8ZibHRA"},
	}

	yti, err := youtube.New(client)
	yt = yti

	if err != nil {
		log.Fatalf("Error creating new YouTube client: %se", err)
	}
	sc := make(chan os.Signal, 1)
	//noinspection ALL
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	discord.Close()

}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	if strings.HasPrefix(m.Content, "!botsay") {
		s.ChannelMessageSend(m.ChannelID, strings.TrimPrefix(m.Content, "!botsay"))
		s.ChannelMessageDelete(m.ChannelID, m.ID)
	}

	if strings.HasPrefix(m.Content, "!sr") {
		defer func() {
			if r := recover(); r != nil {
				s.ChannelMessageSend(m.ChannelID, "Yikes! Something went wrong!")
			}
		}()

		request, err := getSearch(strings.TrimSpace(strings.TrimPrefix(m.Content, "!sr"))) //Requested song/link

		if err != nil {
			sendAndDelete(m.ChannelID, m.ID, err.Error())
			return
		}

		c, _ := s.State.Channel(m.ChannelID)
		se := getServer(s, c)

		if !songExists(request.Id) { //Download
			go download(request.Id)
		}

		se.pl = append(se.pl, request) //Adds item to playlist

		plm[c.GuildID] = se

		s.ChannelMessageDelete(m.ChannelID, m.ID) //Deletes message

	}

	if strings.HasPrefix(m.Content, "!pll") || strings.HasPrefix(m.Content, "!playlist") || strings.HasPrefix(m.Content, "!pl") {
		defer func() {
			recover()
		}()
		c, _ := s.State.Channel(m.ChannelID)
		se := getServer(s, c)

		st := "There are " + strconv.Itoa(len(plm[c.GuildID].pl)) + " songs in the playlist\n"
		for i := range se.pl {
			st += "\n[" + strconv.Itoa(i) + "] " + se.pl[i].Snippet.Title
		}

		sendAndDelete(m.ChannelID, m.ID, st)
	}

	if strings.HasPrefix(m.Content, "!skip") {
		defer func() {
			if r := recover(); r != nil {
				s.ChannelMessageSend(m.ChannelID, "Yikes! Something went wrong!")
			}
		}()
		c, _ := s.State.Channel(m.ChannelID)
		se := getServer(s, c)

		if m.Content == "!skip" {
			se.skip = true
			se.pause = false
		} else {
			a := strings.TrimSpace(strings.TrimPrefix(m.Content, "!skip"))
			i, err := strconv.Atoi(a)
			if err != nil {
				return
			}
			if i > 0 {
				se.pl = append(se.pl[:i], se.pl[i+1:]...)
			} else if i == 0 {
				m.Content = "!skip"
				messageCreate(s, m)
			}
		}
		s.ChannelMessageDelete(m.ChannelID, m.ID)
	}

	if strings.HasPrefix(m.Content, "!pause") {
		c, _ := s.State.Channel(m.ChannelID)
		se := getServer(s, c)

		se.pause = !se.pause
		sendAndDelete(m.ChannelID, m.ID)
	}

	if strings.HasPrefix(m.Content, "!play") {
		c, _ := s.State.Channel(m.ChannelID)
		se := getServer(s, c)

		se.pause = false
		sendAndDelete(m.ChannelID, m.ID)
	}

	if strings.HasPrefix(m.Content, "!cowsay") {
		say, _ := cowsay.Say(&cowsay.Cow{
			Phrase:      strings.TrimPrefix(m.Content, "!cowsay"),
			Type:        "default",
			BallonWidth: 40,
		})
		s.ChannelMessageSend(m.ChannelID, "```"+say+"```")
		discord.ChannelMessageDelete(m.ChannelID, m.ID)
	}

	// Noswear detection
	file, _ := os.Open("swears.txt")
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.Contains(strings.ToLower(m.Content), strings.Trim(scanner.Text(), "\" :1,")) {
			fmt.Println("swar")
			fmt.Print(christianCowsay)
			//discord.ChannelMessageSend(m.ChannelID, christianCowsay)
			c, _ := s.UserChannelCreate(m.Author.ID)
			s.ChannelMessageSend(c.ID, christianCowsay)
			break
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}

	if (strings.Contains(strings.ToLower(m.Content), "musicbot") || strings.Contains(strings.ToLower(m.Content), "music bot")) && (strings.Contains(strings.ToLower(m.Content), "bug") || strings.Contains(strings.ToLower(m.Content), "broken") || strings.Contains(strings.ToLower(m.Content), "buggy")) || strings.Contains(strings.ToLower(m.Content), "yikes! something went wrong!") {
		//noinspection ALL
		rand.Seed(int64(time.Now().Unix())) //hehe 69 haha
		bugQuotes := []string{
			"Musicbot is 100% properly working and fixed and there are no bugs ever",
			"What the fuck did you just fucking say about me, you little bitch?",
			"I'm not BROKEN. I'm just *special*",
			strings.Replace(strings.Replace(strings.ToLower(m.Content), "musicbot", m.Author.Mention(), -1), "music bot", m.Author.Mention(), -1),
			"Yikes! something went wrong!",
		}
		rn := rand.Intn(len(bugQuotes))
		s.ChannelMessageSend(m.ChannelID, bugQuotes[rn])
	}
}

func getServer(s *discordgo.Session, c *discordgo.Channel) *server {
	se := plm[c.GuildID] //Saves server locally

	if se == nil { //Initializes server
		se = new(server)
		se.pl = make([]youtube.Video, 0)
		se.connect(s, c)
	}
	return se
}

type server struct {
	discordgo.VoiceConnection
	speakers    map[uint32]*gopus.Decoder
	opusEncoder *gopus.Encoder
	run         *exec.Cmd
	sendpcm     bool
	recvpcm     bool
	recv        chan *discordgo.Packet
	send        chan []int16
	mu          sync.Mutex
	skip        bool
	pause       bool
	pl          []youtube.Video
}

func (se *server) connect(s *discordgo.Session, c *discordgo.Channel) {
	g, _ := s.State.Guild(c.GuildID)
	dgv, _ := s.ChannelVoiceJoin(g.ID, g.VoiceStates[0].ChannelID, false, false)
	se.VoiceConnection = *dgv
	go se.playLoop(s)
	return

}

func (se *server) playLoop(s *discordgo.Session) {
	for {
		for len(se.pl) == 0 {
			time.Sleep(time.Second * 1)
		}

		for !songExists(se.pl[0].Id) {
			time.Sleep(time.Second * 1)
		}

		se.pause = false
		se.skip = false
		se.PlayAudioFile("dl/" + se.pl[0].Id + ".mp3")
		npl := make([]youtube.Video, len(se.pl)-1)
		for i := range se.pl {
			if i == 0 {
				continue
			}
			npl[i-1] = se.pl[i]
		}
		se.pl = npl

	}
}

func (se *server) SendPCM(pcm <-chan []int16) {

	// make sure this only runs one instance at a time.
	//noinspection ALL
	se.mu.Lock()
	if se.sendpcm || pcm == nil {
		//noinspection ALL
		se.mu.Unlock()
		return
	}
	se.sendpcm = true
	//noinspection ALL
	se.mu.Unlock()

	defer func() { se.sendpcm = false }()

	var err error

	se.opusEncoder, err = gopus.NewEncoder(frameRate, channels, gopus.Audio)

	if err != nil {
		fmt.Println("NewEncoder Error:", err)
		return
	}

	for {

		// read pcm from chan, exit if channel is closed.
		recv, ok := <-pcm
		if !ok {
			fmt.Println("PCM Channel closed.")
			return
		}

		// try encoding pcm frame with Opus
		opus, err := se.opusEncoder.Encode(recv, frameSize, maxBytes)
		if err != nil {
			fmt.Println("Encoding Error:", err)
			return
		}

		if se.Ready == false || se.OpusSend == nil {
			fmt.Printf("Discordgo not ready for opus packets. %+se : %+se", se.Ready, se.OpusSend)
			return
		}
		// send encoded opus data to the sendOpus channel
		se.OpusSend <- opus
	}
}

func (se *server) PlayAudioFile(filename string) {
	fmt.Printf("Starting to play %s\n", filename)
	// Create a shell command "object" to run.
	se.run = exec.Command("ffmpeg", "-i", filename, "-f", "s16le", "-ar", strconv.Itoa(frameRate), "-ac", strconv.Itoa(channels), "pipe:1")
	//noinspection ALL
	ffmpegout, err := se.run.StdoutPipe()
	if err != nil {
		fmt.Println("StdoutPipe Error:", err)
		return
	}

	ffmpegbuf := bufio.NewReaderSize(ffmpegout, 16384)

	// Starts the ffmpeg command
	//noinspection ALL
	err = se.run.Start()
	if err != nil {
		fmt.Println("RunStart Error:", err)
		return
	}

	// Send "speaking" packet over the voice websocket
	se.Speaking(true)

	// Send not "speaking" packet over the websocket when we finish
	defer se.Speaking(false)

	// will actually only spawn one instance, a bit hacky.
	if se.send == nil {
		se.send = make(chan []int16, 2)
	}
	go se.SendPCM(se.send)

	for {
		if se.skip {
			fmt.Printf("We (%s) just got skipped!\n!", filename)
			return
		}
		for se.pause {
			time.Sleep(time.Millisecond * 200)
		}
		// read data from ffmpeg stdout
		audiobuf := make([]int16, frameSize*channels)
		//noinspection ALL
		err = binary.Read(ffmpegbuf, binary.LittleEndian, &audiobuf)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return
		}
		if err != nil {
			fmt.Println("error reading from ffmpeg stdout :", err)
			return
		}

		// Send received PCM to the sendPCM channel
		se.send <- audiobuf
	}
	fmt.Printf("%s is outta here!\n", filename)
}

func download(s string) { //TODO: Stream using -g flag in yt-dl
	cmd := exec.Command("youtube-dl", "--extract-audio", "--audio-format", "mp3", "--output", "dl/"+s+".mp3", s)

	fmt.Printf("Beginning download with command :%s\n", cmd)
	//noinspection ALL
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error on download: %s\n", err)
	}
	fmt.Printf("Output : %s", output)
	fmt.Println("Finished download")
}

func songExists(s string) bool {
	if _, err := os.Stat("dl/" + s + ".mp3"); os.IsNotExist(err) { //Download
		return false
	} else {
		return true
	}
}

func parseLink(s string) (string, error) {

	s = strings.TrimPrefix(s, "https://")
	s = strings.TrimPrefix(s, "http://")
	s = strings.TrimPrefix(s, "www.")

	if len(s) == 11 {
		return s, nil
	} else if strings.Contains(s, "youtube.com") {
		s = strings.TrimPrefix(s, "youtube.com/watch?v=")
		s = strings.Split(s, "&")[0]
	} else if strings.Contains(s, "youtu.be") {
		s = strings.TrimPrefix(s, "youtu.be/")
		s = strings.Split(s, "?")[0]
	} else {
		return s, errors.New("No video found")
	}
	return s, nil

}

func getSearch(s string) (youtube.Video, error) {
	defer func() {
		recover()
	}()

	l, _ := parseLink(s)
	if l != s {
		res := yt.Videos.List("snippet, id, contentDetails")
		res.Id(l)
		ress, err := res.Do()
		if err != nil {
			return *new(youtube.Video), err
		}
		return *ress.Items[0], nil
	}
	call := yt.Search.List("snippet")
	call = call.MaxResults(1)
	call = call.Q(s)
	call = call.Type("video")

	response, err := call.Do()

	var ne error
	if err != nil {
		return *new(youtube.Video), ne
	}
	if len(response.Items) == 0 {
		return *new(youtube.Video), errors.New("Sorry, we couldn't find any results for *" + s + "*")
	}
	if response.Items[0].Snippet.LiveBroadcastContent == "live" {
		return *new(youtube.Video), errors.New("Sorry, live broadcasts are not supported at the moment")

	}
	res := yt.Videos.List("snippet, id, contentDetails")
	res.Id(response.Items[0].Id.VideoId)
	ress, err := res.Do()
	if err != nil {
		return *new(youtube.Video), err
	}
	return *ress.Items[0], nil

}

func sendAndDelete(c string, m string, s ...string) {
	var iid = make([]string, len(s)+1)
	iid[0] = m
	for i := range s {
		id, _ := discord.ChannelMessageSend(c, s[i])
		iid[i+1] = id.ID
	}

	time.Sleep(time.Second * 5)

	for i := range iid {
		discord.ChannelMessageDelete(c, iid[i])
	}

}

const (
	channels  int = 2                   // 1 for mono, 2 for stereo
	frameRate int = 48000               // audio sampling rate
	frameSize int = 960                 // uint16 size of each audio frame
	maxBytes  int = (frameSize * 2) * 2 // max size of opus data
)
