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
	"errors"
	"fmt"
	"github.com/bwmarrin/dgvoice"
	"github.com/bwmarrin/discordgo"
	"google.golang.org/api/googleapi/transport"
	"google.golang.org/api/youtube/v3"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
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
			sendAndDelete(m.ChannelID, m.ID, err.Error())
			return
		}

		c, _ := s.State.Channel(m.ChannelID)
		se := plm[c.GuildID] //Saves server locally

		if se == nil {
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
		defer func() {
			recover()
		}()
		c, _ := s.State.Channel(m.ChannelID)
		se := plm[c.GuildID] //Saves server locally

		if se == nil { //Initializes server
			se = new(server)
			se.pl = make([]youtube.Video, 0)
			se.connect(s, c)
		}
		st := "There are " + strconv.Itoa(len(plm[c.GuildID].pl)) + " songs in the playlist\n"
		for i := range se.pl {
			st += "\n[" + strconv.Itoa(i) + "] " + se.pl[i].Snippet.Title
		}

		sendAndDelete(m.ChannelID, m.ID, st)
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

	if (strings.Contains(strings.ToLower(m.Content), "musicbot") || strings.Contains(strings.ToLower(m.Content), "music bot")) && (strings.Contains(strings.ToLower(m.Content), "bug") || strings.Contains(strings.ToLower(m.Content), "broken") || strings.Contains(strings.ToLower(m.Content), "buggy")) || strings.Contains(strings.ToLower(m.Content), "yikes! something went wrong!") {
		//noinspection ALL
		rand.Seed(int64(time.Now().Unix())) //hehe 69 haha
		bugQuotes := []string{
			"Musicbot is 100% properly working and fixed and there are no bugs ever™",
			"Fuck you",
			"At least I'm not as stupid as you",
			"What the fuck did you just fucking say about me, you little bitch?",
			"@here " + m.Author.Mention() + " has aids",
			"Please submit all bug reports by SHOVING THEM UP YOUR ASS",
			"I'm not BROKEN. I'm just 🌼*special*🌼",
			"At least I'm not a virgin who spends all their time on the internet laughing at shitty memes",
			strings.Replace(strings.Replace(strings.ToLower(m.Content), "musicbot", m.Author.Mention(), -1), "music bot", m.Author.Mention(), -1),
			"Yikes! something went wrong!",
		}
		rn := rand.Intn(len(bugQuotes))
		s.ChannelMessageSend(m.ChannelID, bugQuotes[rn])
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
	dgvoice.PlayAudioFile(se.dgv, "dl/"+v.Id+".mp3")
	discord.UpdateStatus(0, "")
	se.playing = false
	fmt.Println("Stopped playing")
}

func download(s string) {
	cmd := exec.Command("youtube-dl", "--extract-audio", "--audio-format", "mp3", "--output", "dl/"+s+".mp3", s)

	fmt.Println(cmd)
	//noinspection ALL
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(output)

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
		res.Id(s)
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
