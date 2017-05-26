package main

import (
	"github.com/bwmarrin/discordgo"
	"github.com/bwmarrin/dgvoice"
	"os"
	"strings"
	"os/signal"
	"syscall"
	"time"
	"strconv"
	"os/exec"
	"fmt"
	"google.golang.org/api/youtube/v3"
	"net/http"
	"log"
	"google.golang.org/api/googleapi/transport"
)

var plm map[string]*server
var yt *youtube.Service

func main() {
	discord, _ := discordgo.New("Bot MTg5MTQ2MDg0NzE3NjI1MzQ0.DANL1A.4cLruFPliFxkd0r41pYB307_D1M")
	discord.Open()
	discord.AddHandler(messageCreate)

	plm = make(map[string]*server)


	client := &http.Client{
		Transport: &transport.APIKey{Key: "AIzaSyBTYNvJ80kHSE8AypP7Yst5Fshc8ZibHRA"},
	}

	yti, err := youtube.New(client)
	yt = yti

	if err != nil {
		log.Fatalf("Error creating new YouTube client: %v", err)
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
				s.ChannelMessageSend(m.ID, "Hmm, we couldn't find a youtube video with that link")
			}
		}()
		request := getSearch(strings.TrimSpace(strings.TrimPrefix(m.Content, "!sr"))) //Requested song/link
		if request == "" {
			panic("Can't find video")
		}else if request == "live"{
			s.ChannelMessageSend(m.ChannelID, "That's a livestream you moron")
			return
		}
		c, _ := s.State.Channel(m.ChannelID)
		se := plm[c.GuildID] //Saves server locally

		if se == nil { //Initializes server
			se = new(server)
			se.pl = make([]string, 0)
			se.connect(s, c)
		}

		if !songExists(request) { //Download
			go download(request)
		}

		se.pl = append(se.pl, request) //Adds item to playlist

		plm[c.GuildID] = se

		s.ChannelMessageDelete(m.ChannelID, m.ID) //Deletes message

	}

	if strings.HasPrefix(m.Content, "!pll") || strings.HasPrefix(m.Content, "!playlist") || strings.HasPrefix(m.Content, "!pl") {
		defer func(){
			recover()
		}()
		c, _ := s.State.Channel(m.ChannelID)
		se := plm[c.GuildID] //Saves server locally

		if se == nil { //Initializes server
			se = new(server)
			se.pl = make([]string, 0)
			se.connect(s, c)
		}
		st := "There are "+strconv.Itoa(len(plm[c.GuildID].pl))+" songs in the playlist\n"
		for i := range se.pl{
			st += "\n["+strconv.Itoa(i)+"] "+se.pl[i]
		}
		sent, _ :=s.ChannelMessageSend(m.ChannelID, st)

		delete := func(){
			time.Sleep(time.Second* 5)
			s.ChannelMessageDelete(m.ChannelID, m.ID)
			s.ChannelMessageDelete(m.ChannelID, sent.ID)
		}
		go delete()
	}

	if strings.HasPrefix(m.Content, "!skip") {
		c, _ := s.State.Channel(m.ChannelID)
		se := plm[c.GuildID] //Saves server locally

		if se == nil { //Initializes server
			se = new(server)
			se.pl = make([]string, 0)
			se.connect(s, c)
		}

		if m.Content == "!skip" {
			dgvoice.KillPlayer()
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
}

type server struct {
	dgv     *discordgo.VoiceConnection
	pl      []string
	playing bool
}

func (se *server) connect(s *discordgo.Session, c *discordgo.Channel) {
	g, _ := s.State.Guild(c.GuildID)
	dgv, _ := s.ChannelVoiceJoin(g.ID, g.VoiceStates[0].ChannelID, false, false)
	se.dgv = dgv
	go se.playLoop(s)
	return

}

func (se *server) playLoop(s *discordgo.Session) {
	for {
		for len(se.pl) == 0 {
			time.Sleep(time.Second * 1)
		}

		for !songExists(se.pl[0]) {
			time.Sleep(time.Second * 1)
		}

		se.playFile()
		npl := make([]string, len(se.pl)-1)
		for i := range se.pl {
			if i == 0 {
				continue
			}
			npl[i-1] = se.pl[i]
		}
		se.pl = npl

	}
}

func (se *server) playFile() {
	se.playing = true
	fmt.Println("Playing")
	dgvoice.PlayAudioFile(se.dgv, se.pl[0]+".mp3")
	se.playing = false
	fmt.Println("Stopped playing")
}

func download(s string) {
	cmd := exec.Command("youtube-dl", "--extract-audio", "--audio-format", "mp3", "--output", s+".mp3", s)

	// Combine stdout and stderr
	fmt.Println(cmd)
	output, err := cmd.CombinedOutput()
	fmt.Println(err)
	fmt.Println(output) // => go version go1.3 darwin/amd64

}

func songExists(s string) bool {
	if _, err := os.Stat(s + ".mp3"); os.IsNotExist(err) { //Download
		return false
	} else {
		return true
	}
}

func parseLink(s string) string {

	s = strings.TrimPrefix(s, "https://")
	s = strings.TrimPrefix(s, "http://")
	s = strings.TrimPrefix(s, "www.")

	if len(s) == 11 {
		return s
	} else if strings.Contains(s, "youtube.com") {
		s = strings.TrimPrefix(s, "youtube.com/watch?v=")
		s = strings.Split(s, "&")[0]
	} else if strings.Contains(s, "youtu.be") {
		s = strings.TrimPrefix(s, "youtu.be/")
		s = strings.Split(s, "?")[0]
	} else {
		panic("No video found")
	}
	return s

}

func getSearch(s string) string{
	defer func(){
		recover()
	}()

	call := yt.Search.List("snippet")
	call = call.MaxResults(1)
	call = call.Q(s)

	response, err := call.Do()

	if err != nil {
		log.Fatal(err)
	}
	if len(response.Items) == 0{
		panic("No results")
	}
	if response.Items[0].Snippet.LiveBroadcastContent == "live"{
		return "live"
	}
	return response.Items[0].Id.VideoId

}