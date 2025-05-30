package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/gotify/plugin-api"
)

// GetGotifyPluginInfo returns gotify plugin info
func GetGotifyPluginInfo() plugin.Info {
    return plugin.Info{
    Version: "1.1",
    Author: "chacuavip10",
    Name: "Gotify 2 Telegram",
    Description: "Telegram message fowarder for gotify, with proxy support",
        ModulePath: "https://github.com/chacuavip10/gotify2telegram",
    }
}

// Plugin is the plugin instance
type Plugin struct {
    ws *websocket.Conn;
    msgHandler plugin.MessageHandler;
    debugLogger *log.Logger;
    telegram_chatid string;
    telegram_bot_token string;
    telegram_proxy_url string;
    gotify_host string;
}

type GotifyMessage struct {
    Id uint32;
    Appid uint32;
    Message string;
    Title string;
    Priority uint32;
    Date string;
}

type Payload struct {
    ChatID string `json:"chat_id"`
    Text   string `json:"text"`
    Parse_mode  string `json:"parse_mode"`
}

func (p *Plugin) send_msg_to_telegram(msg string) {
    step_size := 4090
    sending_message := ""
    parse_mode_tele := "HTML"
    if strings.HasPrefix(msg, "```") {parse_mode_tele = "Markdown"}
    for i:=0; i<len(msg); i+=step_size {
        if i+step_size < len(msg) {
			sending_message = msg[i : i+step_size]
		} else {
			sending_message = msg[i:]
		}

        data := Payload{
        // Fill struct
            ChatID: p.telegram_chatid,
            Text: sending_message,
            Parse_mode: parse_mode_tele,
        }
        payloadBytes, err := json.Marshal(data)
        if err != nil {
            p.debugLogger.Println("Create json false")
            return
        }
        body := bytes.NewBuffer(payloadBytes)

        client := &http.Client{}
        if p.telegram_proxy_url != "" {
            proxy, err := url.Parse(p.telegram_proxy_url)
            if err != nil {
                return
            }
            client.Transport = &http.Transport{
                Proxy: http.ProxyURL(proxy),
            }
        } else {
            p.debugLogger.Println("Proxy URL is empty, using default transport")
            client.Transport = http.DefaultTransport}
        resp, err := client.Post("https://api.telegram.org/bot"+ p.telegram_bot_token +"/sendMessage", "application/json", body)

        if err != nil {
            p.debugLogger.Printf("Send request false: %v\n", err)
            return
        }
        p.debugLogger.Println("HTTP request was sent successfully")

        if resp.StatusCode == http.StatusOK {
            p.debugLogger.Println("The message was forwarded successfully to Telegram")
        } else {
            // Log infor for debugging
            p.debugLogger.Println("The message was failed to forwarded to Telegram")
        }

        defer resp.Body.Close()
    }
}

func (p *Plugin) connect_websocket() {
    for {
        ws, _, err := websocket.DefaultDialer.Dial(p.gotify_host, nil)
        if err == nil {
            p.ws = ws
            break
        }
        p.debugLogger.Printf("Cannot connect to websocket: %v\n", err)
        time.Sleep(5)
    }
    p.debugLogger.Println("WebSocket connected successfully, ready for forwarding")
}

func (p *Plugin) get_websocket_msg(url string, token string) {
    p.gotify_host = url + "/stream?token=" + token
    p.telegram_chatid = os.Getenv("TELEGRAM_CHAT_ID")
    p.debugLogger.Printf("chatid: %v\n", p.telegram_chatid)
    p.telegram_bot_token = os.Getenv("TELEGRAM_BOT_TOKEN")
    p.debugLogger.Printf("Bot token: %v\n", p.telegram_bot_token)
    p.telegram_proxy_url = os.Getenv("TELEGRAM_PROXY_URL")
    p.debugLogger.Printf("Proxy URL: %v\n", p.telegram_proxy_url)

    go p.connect_websocket()

    for {
        msg := &GotifyMessage{}
        if p.ws == nil {
            time.Sleep(3)
            continue
        }
        err := p.ws.ReadJSON(msg)
        if err != nil {
            p.debugLogger.Printf("Error while reading websocket: %v\n", err)
            p.connect_websocket()
            continue
        }
        p.send_msg_to_telegram(msg.Message)
    }
}

// SetMessageHandler implements plugin.Messenger
// Invoked during initialization
func (p *Plugin) SetMessageHandler(h plugin.MessageHandler) {
    p.debugLogger = log.New(os.Stdout, "Gotify 2 Telegram: ", log.Lshortfile)
    p.msgHandler = h
}

func (p *Plugin) Enable() error {
    go p.get_websocket_msg(os.Getenv("GOTIFY_HOST"), os.Getenv("GOTIFY_CLIENT_TOKEN"))
    return nil
}

// Disable implements plugin.Plugin
func (p *Plugin) Disable() error {
    if p.ws != nil {
        p.ws.Close()
    }
    return nil
}

// NewGotifyPluginInstance creates a plugin instance for a user context.
func NewGotifyPluginInstance(ctx plugin.UserContext) plugin.Plugin {
    return &Plugin{}
}

func main() {
    panic("this should be built as go plugin")
    // For testing
    // p := &Plugin{nil, nil, "", "", ""}
    // p.get_websocket_msg(os.Getenv("GOTIFY_HOST"), os.Getenv("GOTIFY_CLIENT_TOKEN"))
}
