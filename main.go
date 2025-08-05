package main

import (
	"fmt"
	"html"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	_ "github.com/joho/godotenv/autoload"
)

var (
	startTimeStamp = time.Now()
	Token          = os.Getenv("TOKEN")
	ApiHash        = os.Getenv("API_HASH")
	ApiId          = os.Getenv("API_ID")
)

func filterTerabox(m *tg.NewMessage) bool {
	text := m.Text()
	if m.IsCommand() || text == "" || m.IsForward() || m.Message.ViaBotID == m.Client.Me().ID {
		return false
	}

	teraboxRegex := regexp.MustCompile(`(?i)(?:https?://)?(?:www\.)?(?:mirrobox\.com|nephobox\.com|freeterabox\.com|1024tera\.com|1024terabox\.com|terabox\.com|4funbox\.com|terabox\.app|terabox\.fun|tibibox\.com|momerybox\.com|teraboxapp\.com|4funbox\.co)/(?:s/[a-zA-Z0-9_-]+|sharing/link\?surl=[a-zA-Z0-9_-]+)`)
	match := teraboxRegex.MatchString(text)
	return match
}

func teraBoxHandle(m *tg.NewMessage) error {
	shareURL := m.Text()
	reply, err := m.Reply(fmt.Sprintf("🔍 Processing TeraBox link...\n\n<b>%s</b>", html.EscapeString(shareURL)), tg.SendOptions{
		ParseMode: tg.HTML,
	})
	if err != nil {
		return fmt.Errorf("failed to send initial message: %w", err)
	}

	info, err := getTeraBoxInfo(shareURL)
	if err != nil {
		_, _ = reply.Edit("❌ Error getting file info: " + html.EscapeString(err.Error()))
		return fmt.Errorf("getTeraBoxInfo failed: %w", err)
	}

	if len(info.List) == 0 {
		_, _ = reply.Edit("⚠️ No files found in the shared link.")
		return nil
	}

	var text strings.Builder
	keyboard := tg.NewKeyboard()
	var downloadErrors []error
	for i, file := range info.List {
		filename := html.EscapeString(file.ServerFilename)
		sizeReadable := formatBytes(file.Size)
		buttons := []tg.KeyboardButton{
			tg.Button.URL("📥 CDN", file.Dlink),
			tg.Button.URL("⚡ Fast CDN", file.DirectLink),
		}
		keyboard.AddRow(buttons...)
		fileMsg, err := reply.Reply(fmt.Sprintf("⬇️ Downloading %d/%d: <b>%s</b> (%s)",
			i+1, len(info.List), filename, sizeReadable), tg.SendOptions{
			ParseMode: tg.HTML,
		})

		if err != nil {
			downloadErrors = append(downloadErrors, fmt.Errorf("failed to send download message for %s: %w", filename, err))
			continue
		}

		path, mimeType, err := DownloadFile(file.DirectLink)
		if err != nil {
			_, _ = fileMsg.Edit(fmt.Sprintf("❌ Failed to download: <b>%s</b>\nError: %s",
				filename, html.EscapeString(err.Error())), tg.SendOptions{
				ParseMode: tg.HTML,
			})
			downloadErrors = append(downloadErrors, fmt.Errorf("download failed for %s: %w", filename, err))
			continue
		}

		_, err = fileMsg.Edit(fmt.Sprintf("✅ Downloaded: <b>%s</b>", filename), tg.SendOptions{
			Media:     path,
			MimeType:  mimeType,
			ParseMode: tg.HTML,
		})
		if err != nil {
			downloadErrors = append(downloadErrors, fmt.Errorf("failed to send file %s: %w", filename, err))
			continue
		}

		text.WriteString(fmt.Sprintf(
			"📁 <b>%s</b>\n📦 %s | <a href=\"%s\">Stream</a>\n\n",
			filename, sizeReadable, file.StreamURL,
		))
	}

	keyboard.AddRow(
		tg.Button.URL("🛠️ Source Code", "https://github.com/AshokShau/TeraBoxDl"),
	)

	summary := fmt.Sprintf("🎉 <b>Download Complete</b>\n\n%s", text.String())
	if len(downloadErrors) > 0 {
		summary += fmt.Sprintf("\n⚠️ <i>%d files failed to download</i>", len(downloadErrors))
	}

	_, err = reply.Edit(summary, tg.SendOptions{
		ParseMode:   tg.HTML,
		ReplyMarkup: keyboard.Build(),
		LinkPreview: false,
	})

	if err != nil {
		return fmt.Errorf("failed to send summary: %w", err)
	}
	return nil
}

// buildAndStart initializes and logs into the bot client
func buildAndStart(token string) (*tg.Client, bool) {
	apiId, err := strconv.Atoi(ApiId)
	if err != nil {
		log.Printf("❌ Failed to parse API ID: %v", err)
		return nil, false
	}

	client, err := tg.NewClient(tg.ClientConfig{
		AppID:   int32(apiId),
		AppHash: ApiHash,
		//FloodHandler: handleFlood,
		SessionName: "session",
	})
	if err != nil {
		log.Printf("❌ Failed to create client: %v", err)
		return nil, false
	}

	if _, err := client.Conn(); err != nil {
		log.Printf("❌ Connection error: %v", err)
		return nil, false
	}

	if err := client.LoginBot(token); err != nil {
		log.Printf("❌ Login error: %v", err)
		return nil, false
	}

	me, err := client.GetMe()
	if err != nil {
		log.Printf("❌ GetMe error: %v", err)
		return nil, false
	}
	uptime := time.Since(startTimeStamp).Round(time.Millisecond)
	log.Printf("✅ [Client] Logged in as @%s | Startup time: %s", me.Username, uptime)
	return client, true
}

// handleFlood delays on flood wait errors
func handleFlood(err error) bool {
	if wait := tg.GetFloodWait(err); wait > 0 {
		log.Printf("⚠️ Flood wait detected: sleeping for %ds", wait)
		time.Sleep(time.Duration(wait) * time.Second)
		return true
	}
	return false
}

// startHandle responds to the /start command with a welcome message.
func startHandle(m *tg.NewMessage) error {
	bot := m.Client.Me()
	name := html.EscapeString(m.Sender.FirstName)

	response := fmt.Sprintf(`
<b>👋 Hello %s!</b>

🎧 <b>Welcome to %s</b> — your personal <i>TeraBox</i> downloader bot!

📥 Just send me a supported share link, and I’ll fetch the file for you — fast and free!

🌐 <b>Supported Domains:</b>
<code>
terabox.com           |  freeterabox.com
mirrobox.com          |  1024tera.com
nephobox.com          |  4funbox.com
terabox.app           |  terabox.fun
tibibox.com           |  momerybox.com
teraboxapp.com
</code>

💡 <i>Shortened or region-specific links from the above are supported too and use FireFox browser for fast downloads</i>

⚙️ <i>Need help or updates?</i> Use the buttons below.`, name, html.EscapeString(bot.FirstName))

	keyboard := tg.NewKeyboard().
		AddRow(
			tg.Button.URL("💫 Fᴀʟʟᴇɴ Pʀᴏᴊᴇᴄᴛꜱ", "https://t.me/FallenProjects"),
		).AddRow(
		tg.Button.URL("🛠️ Sᴏᴜʀᴄᴇ Cᴏᴅᴇ", "https://github.com/AshokShau/TeraBoxDl"),
	)

	_, err := m.Reply(response, tg.SendOptions{
		ParseMode:   tg.HTML,
		ReplyMarkup: keyboard.Build(),
	})
	return err
}

// checkEnvVars validates required environment variables
func checkEnvVars(vars map[string]string) {
	for k, v := range vars {
		if v == "" {
			log.Fatalf("❌ Missing required environment variable: %s", k)
		}
	}
}

func main() {
	checkEnvVars(
		map[string]string{
			"TOKEN":    Token,
			"API_HASH": ApiHash,
			"API_ID":   ApiId,
		},
	)
	client, ok := buildAndStart(Token)
	if !ok {
		log.Fatal("❌ [Startup] Bot client initialization failed")
	}

	client.On("command:start", startHandle)
	client.On("message:*", teraBoxHandle, tg.FilterFunc(filterTerabox))
	client.Idle()
	log.Println("🛑 [Shutdown] Bot stopped.")
}
