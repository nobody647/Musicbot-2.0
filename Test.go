package main

//import "fmt"
import (
	"github.com/bwmarrin/discordgo"
	"os"
	"os/signal"
	"syscall"
	"strings"
)

func main() {
	discord, _ := discordgo.New("Bot MTg5MTQ2MDg0NzE3NjI1MzQ0.DANL1A.4cLruFPliFxkd0r41pYB307_D1M")
	discord.Open()
	discord.ChannelMessageSend("104979971667197952", "hey there")
	discord.AddHandler(messageCreate)
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}
	if strings.HasPrefix(m.Content, "!echo") {
		s.ChannelMessageSend(m.ChannelID, m.Content)
	}
	if m.Author.Bot {
		s.ChannelMessageSend(m.ChannelID, m.Author.Mention()+" ur gay")
	}
	if strings.HasPrefix(m.Content, "!botsay") {
		s.ChannelMessageSend(m.ChannelID, strings.TrimPrefix(m.Content, "!botsay"))
		s.ChannelMessageDelete(m.ChannelID, m.ID)
	}

	if strings.HasPrefix(m.Content, "!") {

	}
}
