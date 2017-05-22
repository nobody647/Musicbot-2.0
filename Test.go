package main

import (
	"github.com/bwmarrin/discordgo"
	"github.com/bwmarrin/dgvoice"
	"os"
	"strings"
	"os/signal"
	"syscall"
)

//var plm map[string][]string

func main() {
	discord, _ := discordgo.New("Bot MTg5MTQ2MDg0NzE3NjI1MzQ0.DANL1A.4cLruFPliFxkd0r41pYB307_D1M")
	discord.Open()
	discord.ChannelMessageSend("104979971667197952", "hey there")

	discord.AddHandler(messageCreate)

	sc := make(chan os.Signal, 1)
	//noinspection ALL
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	discord.Close()
	//plm = make(map[string][]string)

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
		dgv, _ := s.ChannelVoiceJoin("104979971667197952", "156887965392437250", false, false)
		dgvoice.PlayAudioFile(dgv, "test.mp3")
		//c, _ := s.State.Channel(m.ChannelID)

		//plm[c.GuildID] = append(plm[c.GuildID], "wZZ7oFKsKzY") //Adds item to playlist

	}

}

//func play(c discordgo.Channel, )