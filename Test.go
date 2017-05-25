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
)

var plm map[string]*server

func main() {
	discord, _ := discordgo.New("Bot MTg5MTQ2MDg0NzE3NjI1MzQ0.DANL1A.4cLruFPliFxkd0r41pYB307_D1M")
	discord.Open()
	//discord.ChannelMessageSend("104979971667197952", "*hello there*")

	discord.AddHandler(messageCreate)

	plm = make(map[string]*server)

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
	if strings.HasPrefix(m.Content, "!echo") {
		s.ChannelMessageSend(m.ChannelID, m.Content)

	}
	if m.Author.Bot {
		s.ChannelMessageSend(m.ChannelID, m.Author.Mention()+" ur geay")
	}
	if strings.HasPrefix(m.Content, "!botsay") {
		s.ChannelMessageSend(m.ChannelID, strings.TrimPrefix(m.Content, "!botsay"))
		s.ChannelMessageDelete(m.ChannelID, m.ID)
	}

	if strings.HasPrefix(m.Content, "!sr") {
		defer func() {
			if r := recover(); r != nil{
				s.ChannelMessageSend(m.ID, "Hmm, we couldn't find a youtube video with that link")
			}
		}()
		request := parseLink(strings.TrimSpace(strings.TrimPrefix(m.Content,"!sr"))) //Requested song/link

		c, _ := s.State.Channel(m.ChannelID)
		se := plm[c.GuildID] //Saves server locally

		if se == nil { //Initializes server
			se = new(server)
			se.pl = make([]string, 0)
			se.connect(s, c)
		}

		if !songExists(request){ //Download
			go download(request)
		}

		se.pl = append(se.pl, request) //Adds item to playlist

		plm[c.GuildID] = se

	}

	if strings.HasPrefix(m.Content, "!pll"){
		c, _ := s.State.Channel(m.ChannelID)

		s.ChannelMessageSend(m.ChannelID, strconv.Itoa(len(plm[c.GuildID].pl)))
	}
	if strings.HasPrefix(m.Content, "!skip"){

		if m.Content == "!skip"{
			dgvoice.KillPlayer()
		}else{
			a := strings.TrimSpace(strings.TrimPrefix(m.Content, "!skip"))
			i, err := strconv.Atoi(a)
			if err != nil {
				return
			}
			if i < 0{
				c, _ := s.State.Channel(m.ChannelID)
				se := plm[c.GuildID] //Saves server locally

				se.pl = append(se.pl[:i], se.pl[i+1:]...)
			}else if i == 0{
				m.Content = "!skip"
				messageCreate(s, m)
			}
		}
	}
}

func (se *server) playLoop(s *discordgo.Session) {
	for{
		for len(se.pl)==0{
			time.Sleep(time.Second*1)
		}

		for !songExists(se.pl[0]){
			time.Sleep(time.Second*1)
		}


		se.playFile()
		npl := make([]string, len(se.pl)-1)
		for i := range se.pl{
			if i==0{
				continue
			}
			npl[i-1] = se.pl[i]
		}
		se.pl = npl

	}
}

func (se *server)playFile() {
	se.playing = true
	fmt.Println("Playing")
	dgvoice.PlayAudioFile(se.dgv, se.pl[0]+".mp3")
	se.playing = false
	fmt.Println("Stopped playing")
}

func (se *server) connect(s *discordgo.Session, c *discordgo.Channel) {
	g, _ := s.State.Guild(c.GuildID)
	dgv, _ := s.ChannelVoiceJoin(g.ID, g.VoiceStates[0].ChannelID, false,false)
	se.dgv = dgv
	go se.playLoop(s)
	return

}

type server struct{
	dgv *discordgo.VoiceConnection
	pl  []string
	playing bool

}

func download(s string){
	cmd := exec.Command("youtube-dl", "--extract-audio", "--audio-format", "mp3", "--output", ""+s+".mp3" ,s)

	// Combine stdout and stderr
	printCommand(cmd)
	output, err := cmd.CombinedOutput()
	printError(err)
	printOutput(output) // => go version go1.3 darwin/amd64


}

func songExists(s string) bool{
	if _, err := os.Stat(s+".mp3"); os.IsNotExist(err) { //Download
		return false
	}else{
		return true
	}
}
func printCommand(cmd *exec.Cmd) {
	fmt.Printf("==> Executing: %s\n", strings.Join(cmd.Args, " "))
}
func printError(err error) {
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("==> Error: %s\n", err.Error()))
	}
}
func printOutput(outs []byte) {
	if len(outs) > 0 {
		fmt.Printf("==> Output: %s\n", string(outs))
	}
}

func parseLink(s string) string{

	s = strings.TrimPrefix(s, "https://")
	s = strings.TrimPrefix(s, "http://")
	s = strings.TrimPrefix(s, "www.")


	if len(s) == 11{
		return s
	}else if strings.Contains(s, "youtube.com"){
		s = strings.TrimPrefix(s, "youtube.com/watch?v=")
		s = strings.Split(s,"&")[0]
	}else if strings.Contains(s, "youtu.be"){
		s = strings.TrimPrefix(s, "youtu.be/")
		s = strings.Split(s, "?")[0]
	}else{
		panic("No video found")
	}
	return s

}