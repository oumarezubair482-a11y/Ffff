package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

var (
	client        *whatsmeow.Client
	container     *sqlstore.Container
	reactionCount int
)

func sendReaction(from types.JID, msgID string) {
	if client == nil || !client.IsConnected() {
		return
	}
	emoji := getReaction(reactionCount)
	reactionCount++
	_, _ = client.SendMessage(context.Background(), from, &waProto.Message{
		ReactionMessage: &waProto.ReactionMessage{
			Key: &waProto.MessageKey{
				RemoteJid: proto.String(from.String()),
				FromMe:    proto.Bool(false),
				Id:        proto.String(msgID),
			},
			Text:              proto.String(emoji),
			SenderTimestampMs: proto.Int64(time.Now().UnixMilli()),
		},
	})
}

func handleMessage(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		if v.Info.IsFromMe {
			return
		}
		from := v.Info.Chat
		go sendReaction(from, v.Info.ID)

		text := v.Message.GetConversation()
		if v.Message.ExtendedTextMessage != nil {
			text = v.Message.ExtendedTextMessage.GetText()
		}
		text = strings.TrimSpace(text)

		reply := func(msg string) {
			if client == nil {
				return
			}
			_, _ = client.SendMessage(context.Background(), from, &waProto.Message{
				Conversation: proto.String(msg),
			})
		}

		switch text {
		case ".menu", ".help":
			reply(getMenu())
		case ".ping":
			start := time.Now()
			reply(fmt.Sprintf("🏓 *Pong!* %dms", time.Since(start).Milliseconds()))
		case ".owner":
			reply("👑 *Owner:* Hina\n📱 Hina MD Bot\n🌸 Version: 1.0")
		case ".alive":
			reply(getAlive())
		case ".id":
			reply(fmt.Sprintf("👤 *User ID:*\n%s\n\n💬 *Chat ID:*\n%s",
				v.Info.Sender.ToNonAD().String(),
				v.Info.Chat.ToNonAD().String(),
			))
		}

	case *events.Connected:
		fmt.Println("✅ Hina MD Connected!")
	case *events.Disconnected:
		fmt.Println("🔄 Reconnecting...")
		time.Sleep(3 * time.Second)
		if client != nil {
			_ = client.Connect()
		}
	}
}

func startHTTP() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		status := "❌ Disconnected"
		if client != nil && client.IsConnected() {
			status = "✅ Connected"
		}
		fmt.Fprintf(w, "🌸 Hina MD Bot\nStatus: %s\n\nEndpoints:\n/pair/NUMBER\n/status\n/logout", status)
	})

	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		connected := client != nil && client.IsConnected()
		fmt.Fprintf(w, `{"bot":"Hina MD","version":"1.0","connected":%v}`, connected)
	})

	http.HandleFunc("/pair/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 3 || parts[2] == "" {
			http.Error(w, "Use: /pair/923001234567", 400)
			return
		}
		number := strings.NewReplacer("+", "", " ", "", "-", "").Replace(parts[2])
		if len(number) < 10 {
			http.Error(w, "Invalid number", 400)
			return
		}
		if container == nil {
			http.Error(w, "DB not ready", 500)
			return
		}
		if client != nil && client.IsConnected() {
			client.Disconnect()
			time.Sleep(2 * time.Second)
		}

		dev := container.NewDevice()
		tmp := whatsmeow.NewClient(dev, waLog.Stdout("Pair", "ERROR", true))
		tmp.AddEventHandler(handleMessage)
		if err := tmp.Connect(); err != nil {
			http.Error(w, "Connect failed: "+err.Error(), 500)
			return
		}
		time.Sleep(2 * time.Second)

		code, err := tmp.PairPhone(context.Background(), number, true, whatsmeow.PairClientChrome, "Chrome (Linux)")
		if err != nil {
			tmp.Disconnect()
			http.Error(w, "Pair failed: "+err.Error(), 500)
			return
		}
		fmt.Println("📱 Pairing Code:", code)

		go func() {
			for i := 0; i < 60; i++ {
				time.Sleep(1 * time.Second)
				if tmp.Store.ID != nil {
					fmt.Println("✅ Paired!")
					client = tmp
					return
				}
			}
			fmt.Println("⏰ Pair timeout")
			tmp.Disconnect()
		}()

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"success":true,"code":"%s","number":"%s"}`, code, number)
	})

	http.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		if client != nil && client.IsConnected() {
			client.Disconnect()
		}
		if container != nil {
			devices, _ := container.GetAllDevices(context.Background())
			for _, d := range devices {
				_ = d.Delete(context.Background())
			}
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"success":true,"message":"Session delete ho gaya"}`)
	})

	fmt.Println("🌐 HTTP: 0.0.0.0:" + port)
	if err := http.ListenAndServe("0.0.0.0:"+port, nil); err != nil {
		fmt.Println("HTTP Error: " + err.Error())
		os.Exit(1)
	}
}

func main() {
	fmt.Println(LOGO)
	go startHTTP()

	dbLog := waLog.Stdout("DB", "ERROR", true)
	var err error
	container, err = sqlstore.New(context.Background(), "sqlite3",
		"file:auth/hina.db?_foreign_keys=on", dbLog)
	if err != nil {
		fmt.Println("DB Error:", err)
		os.Exit(1)
	}

	device, err := container.GetFirstDevice(context.Background())
	if err == nil && device.ID != nil {
		client = whatsmeow.NewClient(device, waLog.Stdout("WA", "ERROR", true))
		client.AddEventHandler(handleMessage)
		if err := client.Connect(); err == nil {
			fmt.Println("✅ Session restore ho gaya!")
		}
	} else {
		fmt.Println("📱 Session nahi hai!")
		fmt.Println("👉 Browser mein: YOUR_RAILWAY_URL/pair/923001234567")
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	fmt.Println("👋 Hina MD band ho raha hai...")
	if client != nil {
		client.Disconnect()
	}
}
