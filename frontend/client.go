package main

import (
	"fmt"
	"log"
	"github.com/jroimartin/gocui"

	"math/rand"
	"time"
	//"net/url"


	"github.com/gorilla/websocket"

	"crypto/tls"
	"net/url"
	b "github.com/engineerbeard/barrenschat/shared"
	"strings"

)

type server struct{}


type BChatClient struct {
	Name string
	Room string
	WsConn *websocket.Conn
	Uid string
}
func init() {
    rand.Seed(time.Now().UnixNano())
	BClient = BChatClient{}

}
func (c *BChatClient) ChangeName(s string) {
	c.Name = s
}
func (c *BChatClient) SendMessage(s b.BMessage) {
	c.WsConn.WriteJSON(s)
}
var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
var BClient BChatClient
func RandStringRunes(n int) string {
    a := make([]rune, n)
    for i := range a {
        a[i] = letterRunes[rand.Intn(len(letterRunes))]
    }
    return string(a)
}
func main() {
	// Setup Ws connection
	u := url.URL{Scheme: "wss", Host: "localhost:8081", Path: "/bchatws"}
	d := websocket.Dialer{TLSClientConfig: &tls.Config{InsecureSkipVerify:true}}
	c, _, err := d.Dial(u.String(), nil)
	if err != nil {
		log.Fatalln(err)
	}

	//c.WriteJSON()

	//bhelpers.BMessage{MsgType:B_CONNECT, Uid:RandStringRunes(32)}
	BClient = BChatClient{WsConn:c, Uid:RandStringRunes(32), Name:"Anon", Room:b.MAIN_ROOM}
	BClient.SendMessage(b.BMessage{MsgType:b.B_CONNECT, Uid:RandStringRunes(32), Payload:BClient.Name})

	// Setup CUI
	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		log.Panicln(err)
	}
	defer g.Close()

	g.Highlight = true
	g.Cursor = true

	g.SetManagerFunc(setLayout)

	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("", gocui.KeyEnter, gocui.ModNone, onEnterEvt(c)); err != nil {
		log.Panicln(err)
	}


	go handleConnection(c, g)

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Panicln(err)
	}

}
func handleConnection(c *websocket.Conn, g *gocui.Gui) {
	var bMessage b.BMessage
	c.ReadJSON(&bMessage)
	processMsg(bMessage, g)
	for {
		err := c.ReadJSON(&bMessage)
		if err != nil {
			log.Fatal(err)
		}
		processMsg(bMessage, g)
	}
}

func processMsg( msg b.BMessage, g *gocui.Gui) {
	g.Update(func(g *gocui.Gui) error {
		v, _ := g.View("chatwindow")
		if msg.MsgType == b.B_CONNECT || msg.MsgType == b.B_NAMECHANGE || msg.MsgType == b.B_DISCONNECT{
			o, _ := g.View("online")
			o.Clear()
			fmt.Fprint(o, msg.Data)
		}
		fmt.Fprintln(v,fmt.Sprintf("%s (%s) %s",msg.TimeStamp.Format("2006-01-02 15:04"),msg.Name, msg.Payload))
		return nil
	})
}
func setActiveView(g *gocui.Gui, name string) (*gocui.View, error) {
	if _, err := g.SetCurrentView(name); err != nil {
		return nil, err
	}
	return g.SetViewOnTop(name)
}

func onEnterEvt(c *websocket.Conn) func(g *gocui.Gui, v *gocui.View) error {
	return func(g *gocui.Gui, v *gocui.View) error {
		buf := strings.Replace(v.Buffer(), "\n","", -1)
		if len(buf) < 1 {
			v.SetCursor(0, 0)
			return nil
		}
		msgType := b.B_MESSAGE
		if strings.Contains(buf, "/name") && len(strings.SplitAfter(buf, "/name")) > 1{
			newName := strings.SplitAfter(buf, "/name")[1]
			newName = strings.TrimSpace(newName)
			buf = fmt.Sprintf("%s changed name to %s", BClient.Name, newName)
			BClient.Name = newName
			msgType = b.B_NAMECHANGE

		}
		err := c.WriteJSON(b.BMessage{
			MsgType:msgType,
			TimeStamp:time.Now(),
			Name:BClient.Name,
			Room:BClient.Room,
			Uid:BClient.Uid,
			Payload:buf,
		})
		if err != nil {
			log.Println(err)
		}

		v.Clear()
		v.SetCursor(0, 0)
		return nil
	}

}
func setLayout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	if v, err := g.SetView("online", 0, 0, 20, 14); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Online"
		fmt.Fprintln(v, "")
	}

	if v, err := g.SetView("roomslist", 0, 15, 20, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Rooms"
		fmt.Fprintln(v, "")
	}

	if v, err := g.SetView("input", 21, maxY-3, maxX-1, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Autoscroll = true
		v.Title = "Type To Chat"
		v.Editable = true
		v.Wrap = true
		//fmt.Fprintf(v, "H")
	}

	if v, err := g.SetView("chatwindow", 21, 0, maxX-1, maxY-4); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Autoscroll = true
		v.Title = "BChats"
		v.Editable = false
		v.Wrap = true
	}

	if _, err := setActiveView(g, "input"); err != nil {
		return err
	}
	return nil
}
func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}
