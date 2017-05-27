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
	"errors"
)

var plm map[string]*server
var yt *youtube.Service
var discord discordgo.Session

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
				s.ChannelMessageSend(m.ChannelID, "Yikes! Something went wrong!")
			}
		}()

		request, err := getSearch(strings.TrimSpace(strings.TrimPrefix(m.Content, "!sr"))) //Requested song/link

		if err != nil {
			sent, _ := s.ChannelMessageSend(m.ChannelID, err.Error())

			delete := func(){
				time.Sleep(time.Second* 5)
				s.ChannelMessageDelete(m.ChannelID, m.ID)
				s.ChannelMessageDelete(m.ChannelID, sent.ID)
			}

			go delete()
			return
		}
		c, _ := s.State.Channel(m.ChannelID)
		se := plm[c.GuildID] //Saves server locally

		if se == nil { //Initializes server
			se = new(server)
			se.pl = make([]youtube.Video, 0)
			se.connect(s, c)
		}

		if !songExists(request.Id) { //Download
			go download(request.Id)
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
			se.pl = make([]youtube.Video, 0)
			se.connect(s, c)
		}
		st := "There are "+strconv.Itoa(len(plm[c.GuildID].pl))+" songs in the playlist\n"
		for i := range se.pl{
			st += "\n["+strconv.Itoa(i)+"] "+se.pl[i].Snippet.Title
		}
		sent, _ := s.ChannelMessageSend(m.ChannelID, st)

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
			se.pl = make([]youtube.Video, 0)
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
	pl      []youtube.Video
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

		for !songExists(se.pl[0].Id) {
			time.Sleep(time.Second * 1)
		}

		se.playFile(se.pl[0])
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

func (se *server) playFile(v youtube.Video) {
	se.playing = true
	fmt.Println("Playing")
	discord.UpdateStatus(0, v.Snippet.Title)
	dgvoice.PlayAudioFile(se.dgv, v.Id+".mp3")
	discord.UpdateStatus(0, "")
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

func getSearch(s string) (youtube.Video, error){
	defer func(){
		recover()
	}()

	call := yt.Search.List("snippet")
	call = call.MaxResults(1)
	call = call.Q(s)
	call = call.Type("video")

	response, err := call.Do()

	var ne error
	if err != nil {
		return *new(youtube.Video), ne
	}
	if len(response.Items) == 0{
		return *new(youtube.Video), errors.New("Sorry, we couldn't find any results for *"+s+"*")
	}
	if response.Items[0].Snippet.LiveBroadcastContent == "live"{
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