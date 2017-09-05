/*
    __  ___   __  __   _____    ____   ______    ____    ____   ______          ___       ____
   /  |/  /  / / / /  / ___/   /  _/  / ____/   / __ )  / __ \ /_  __/         |__ \     / __ \
  / /|_/ /  / / / /   \__ \    / /   / /       / __  | / / / /  / /            __/ /    / / / /
 / /  / /  / /_/ /   ___/ /  _/ /   / /___    / /_/ / / /_/ /  / /            / __/  _ / /_/ /
/_/  /_/   \____/   /____/  /___/   \____/   /_____/  \____/  /_/            /____/ (_)\____/

	A project by Ian Flanflan
*/

package main

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
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

	"layeh.com/gopus"

	cowsay "github.com/Code-Hex/Neo-cowsay"
	"github.com/bwmarrin/discordgo"
	"google.golang.org/api/googleapi/transport"
	"google.golang.org/api/youtube/v3"
)

var (
	plm     = make(map[string]*server)
	pmlm    = make(map[string]string)
	cmdlm   = make(map[string]*discordgo.Message)
	yt      *youtube.Service
	discord *discordgo.Session
	cmd     = make(map[string]func(m *discordgo.Message, se *server, s []string))
)

const christianCowsay = ` ______________________________________
< no wearing this is a christian sever >
 --------------------------------------
        \   ^__^
         \  (oo)\_______
            (__)\       )\/\
                ||----w |
                ||     ||`

func init() {
	discord, _ = discordgo.New("Bot MTg5MTQ2MDg0NzE3NjI1MzQ0.DANL1A.4cLruFPliFxkd0r41pYB307_D1M")

	client := &http.Client{
		Transport: &transport.APIKey{Key: "AIzaSyBTYNvJ80kHSE8AypP7Yst5Fshc8ZibHRA"},
	}
	yt, _ = youtube.New(client)
}

func initCommands() {
	cmd["!sr"] = func(m *discordgo.Message, se *server, s []string) {
		request, err := getSearch(strings.TrimSpace(strings.TrimPrefix(m.Content, "!sr"))) //Requested song/link

		if err != nil {
			sendAndDelete(m.ChannelID, m.ID, err.Error())
			return
		}

		if !songExists(request.Id) { //Download
			go download(request.Id)
		}

		se.pl = append(se.pl, request) //Adds item to playlist

		discord.ChannelMessageDelete(m.ChannelID, m.ID) //Deletes message
	}
	plCMD := func(m *discordgo.Message, se *server, s []string) {
		st := "There are " + strconv.Itoa(len(se.pl)) + " songs in the playlist\n"
		for i := range se.pl {
			st += "\n[" + strconv.Itoa(i) + "] " + se.pl[i].Snippet.Title
		}

		sendAndDelete(m.ChannelID, m.ID, st)
	}
	cmd["!pl"] = plCMD
	cmd["!playlist"] = plCMD
	cmd["!skip"] = func(m *discordgo.Message, se *server, s []string) {
		if m.Content == "!skip" {
			se.skip = true
			se.pause = false
		} else if strings.Contains(m.Content.ToLower(), "all") {
			se.pl = se.pl[:0]
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
				checkCommands(m)
			}
		}
		discord.ChannelMessageDelete(m.ChannelID, m.ID)
	}
	cmd["!pause"] = func(m *discordgo.Message, se *server, s []string) {
		se.pause = !se.pause
		sendAndDelete(m.ChannelID, m.ID)
	}
	cmd["!play"] = func(m *discordgo.Message, se *server, s []string) {
		se.pause = false
		sendAndDelete(m.ChannelID, m.ID)
	}
	cmd["!cowsay"] = func(m *discordgo.Message, se *server, s []string) {
		say, _ := cowsay.Say(&cowsay.Cow{
			Phrase:      strings.TrimPrefix(m.Content, "!cowsay "),
			Type:        "default",
			BallonWidth: 40,
		})
		discord.ChannelMessageSend(m.ChannelID, "```"+say+"```")
		discord.ChannelMessageDelete(m.ChannelID, m.ID)
	}
	cmd["!botsay"] = func(m *discordgo.Message, se *server, s []string) {
		discord.ChannelMessageSend(m.ChannelID, strings.TrimPrefix(m.Content, "!botsay"))
		discord.ChannelMessageDelete(m.ChannelID, m.ID)
	}
}

func main() {
	initCommands()
	discord.Open()
	discord.AddHandler(messageHandler)
	sc := make(chan os.Signal, 1)

	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	discord.Close()
}

func messageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == discord.State.User.ID {
		return
	}
	checkCommands(m.Message)

	// Noswear detection
	file, _ := os.Open("swears.txt")
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.Contains(" "+strings.ToLower(m.Content)+" ", " "+strings.Trim(scanner.Text(), "\" :1,")+" ") {
			fmt.Println("swar")
			fmt.Println(scanner.Text())
			fmt.Print(christianCowsay)
			//discord.ChannelMessageSend(m.ChannelID, christianCowsay)
			c, _ := discord.UserChannelCreate(m.Author.ID)
			discord.ChannelMessageSend(c.ID, christianCowsay)
			break
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}

	// Insult generator
	if (strings.Contains(strings.ToLower(m.Content), "musicbot") || strings.Contains(strings.ToLower(m.Content), "music bot")) && (strings.Contains(strings.ToLower(m.Content), "bug") || strings.Contains(strings.ToLower(m.Content), "broken") || strings.Contains(strings.ToLower(m.Content), "buggy")) || strings.Contains(strings.ToLower(m.Content), "yikes! something went wrong!") {
		rand.Seed(int64(time.Now().Unix()))
		bugQuotes := []string{
			"Musicbot is 100% properly working and fixed and there are no bugs ever",
			"What the fuck did you just fucking say about me, you little bitch?",
			"I'm not BROKEN. I'm just *special*",
			strings.Replace(strings.Replace(strings.ToLower(m.Content), "musicbot", m.Author.Mention(), -1), "music bot", m.Author.Mention(), -1),
			"Yikes! something went wrong!",
		}
		rn := rand.Intn(len(bugQuotes))
		discord.ChannelMessageSend(m.ChannelID, bugQuotes[rn])
	}

}

func checkCommands(m *discordgo.Message) {
	if m.Author.ID == discord.State.User.ID {
		return
	}
	c, _ := discord.State.Channel(m.ChannelID)
	se, err := getServer(c)
	if err != nil {
		return
	}

	parts := strings.Split(strings.ToLower(strings.TrimSpace(m.Content)), " ")
	command := cmd[parts[0]]
	if command != nil {
		command(m, se, parts[1:])
	}
}

func getServer(c *discordgo.Channel) (*server, error) {
	if c.GuildID == "" { // If channel is a PM
		// If user is in a voice channel
		for _, se := range discord.State.Guilds {
			for _, vs := range se.VoiceStates {
				if vs.UserID == c.Recipient.ID {
					ch, _ := discord.State.Channel(vs.ChannelID)
					gu, _ := getServer(ch)
					pmlm[c.ID] = gu.GuildID
					return plm[pmlm[c.ID]], nil
				}
			}
		}

		if pmlm[c.ID] != "" {
			return plm[pmlm[c.ID]], nil
		}

		// If user is NOT in a voice channel
		var sList []string // List of servers in common with requester
		for _, se := range discord.State.Guilds {
			for _, me := range se.Members {
				if me.User.ID == c.Recipient.ID {
					sList = append(sList, se.ID)
				}
			}
		}
		m, _ := discord.ChannelMessage(c.ID, c.LastMessageID)
		selected, _ := strconv.Atoi(m.Content)

		if selected != 0 {
			selected = selected - 1
			g, error := discord.State.Guild(sList[selected])
			if error != nil {
				fmt.Println(error)
			}
			gu, _ := getServer(g.Channels[0])
			pmlm[c.ID] = gu.GuildID
			if cmdlm[m.ChannelID] != nil {
				checkCommands(cmdlm[m.ChannelID])
				discord.ChannelMessageSend(m.ChannelID, "Your command `"+cmdlm[m.ChannelID].Content+"` has been run for server `"+g.Name+"`")
				cmdlm[m.ChannelID] = nil
			}
			return plm[pmlm[c.ID]], nil
		} else if len(sList) == 1 { // If only one server in common
			g, _ := discord.State.Guild(sList[0])
			gu, _ := getServer(g.Channels[0])
			pmlm[c.ID] = gu.GuildID
			return plm[pmlm[c.ID]], nil
		} else if len(sList) == 0 {
			discord.ChannelMessageSend(c.ID, "Hmm, I don't seem to have any servers in common with you")
		} else { // If user did not make a selection
			msg := "Please select a server by typing its number"
			for i, gid := range sList {
				msg += "\n"
				msg += "[" + strconv.Itoa(i+1) + "] "
				guild, _ := discord.State.Guild(gid)
				msg += guild.Name
			}
			//msg += "\n Please note that if you just made a request, you will have to make it again after you select a server"
			cmdlm[m.ChannelID] = m
			discord.ChannelMessageSend(c.ID, msg)
			return nil, errors.New("Waiting for selection")
		}

	}

	if plm[c.GuildID] == nil { // Creates new server if one does not exist
		se := server{}
		se.pl = make([]youtube.Video, 0)
		se.connect(c)
		plm[c.GuildID] = &se
	}
	return plm[c.GuildID], nil
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

func (se *server) connect(c *discordgo.Channel) {
	g, _ := discord.State.Guild(c.GuildID)
	var vc string
	if len(g.VoiceStates) == 0 {
		fmt.Println("no vc")
		for _, ch := range g.Channels {
			if ch.Type == "voice" {
				if ch.Position == 0 || strings.Contains(strings.ToLower(ch.Name), "music") {
					vc = ch.ID
					break
				}
			}
		}
	} else {
		vc = g.VoiceStates[0].ChannelID
	}
	dgv, _ := discord.ChannelVoiceJoin(g.ID, vc, false, false)
	se.VoiceConnection = *dgv
	go se.playLoop()
	return

}

func (se *server) playLoop() {
	for {
		for len(se.pl) == 0 {
			time.Sleep(time.Second * 1)
		}
		song := &se.pl[0]

		se.pause = false
		se.skip = false

		if songExists(song.Id) {
			fmt.Println("File exists for " + song.Snippet.Title + ", playing now")
			se.PlayAudioFile("dl/" + se.pl[0].Id + ".mp3")
		} else {
			fmt.Println("Getting stream URL for " + song.Snippet.Title)
			output, err := exec.Command("youtube-dl", "-g", song.Id).Output()
			if err != nil {
				fmt.Print("Error getting URL: ")
				fmt.Println(err)
				continue
			}
			se.PlayAudioFile(strings.Split(string(output), "\n")[1])
		}

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
		if !se.VoiceConnection.Ready {
			se.VoiceConnection.Disconnect()
			ch, _ := discord.Channel(se.ChannelID)
			se.connect(ch)
		}
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
			fmt.Printf("Discordgo not ready for opus packetdiscord. %+se : %+se", se.Ready, se.OpusSend)
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

	fmt.Printf("Beginning download with command :%s\n", cmd.Args)
	//noinspection ALL
	_, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error on download: %s\n", err)
	}
	//fmt.Printf("Output : %s", output)
	fmt.Println("Finished download of " + s)
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
