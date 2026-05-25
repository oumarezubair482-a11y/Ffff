package main

import "fmt"

var LOGO = `
╔══════════════════════════════╗
║      🌸  HINA MD  🌸        ║
║    WhatsApp Bot v1.0         ║
║    Fast • Smart • Go         ║
╚══════════════════════════════╝
`

func getMenu() string {
	return `🌸 *HINA MD Bot Menu* 🌸

━━━━━━━━━━━━━━━━━━━━
⚙️ *Basic Commands:*
━━━━━━━━━━━━━━━━━━━━
• .menu  — Full menu dikhao
• .ping  — Bot speed check
• .owner — Owner info
• .alive — Bot status check
• .id    — Chat/User ID

━━━━━━━━━━━━━━━━━━━━
✨ *Auto Features:*
━━━━━━━━━━━━━━━━━━━━
• Auto Reaction — Har message pe emoji

━━━━━━━━━━━━━━━━━━━━
_🌸 Powered by Hina MD v1.0_`
}

func getAlive() string {
	return fmt.Sprintf(`✅ *Hina MD Online Hai!*
🌸 Version : 1.0
⚡ Language : Go
🚀 Status  : Running
👑 Owner   : Hina`)
}

var reactions = []string{"🌸", "❤️", "😊", "🔥", "✨", "💯", "👍", "🎉"}

func getReaction(index int) string {
	return reactions[index%len(reactions)]
}
