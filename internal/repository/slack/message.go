package slack

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/faruqrahmadani/slack-thread-alert/internal/domain"
	slackgo "github.com/slack-go/slack"
)

const (
	colorDanger  = "#E01E5A"
	colorWarning = "#ECB22E"
	colorGood    = "#2EB67D"
	colorDefault = "#9E9E9E"

	// Slack section text limit is 3000 characters
	maxSectionText = 2900
)

// buildSummary formats the parent message: the latest status of the alert cycle, readable at a glance.
//
//	🔴 ALERT · [P1] CPU high on api
//	┃ cpu over host:api was > 90.0 during the last 5m
//	┃ Started  10:00            Duration  ongoing for 6m
//	┃ Scope    host:api         Host      api-1
//	┃ <latest graph snapshot>
//	┃ View event · Monitor · datadog · Kredivo · updated 10:06
func buildSummary(alert *domain.Alert, startedAt, now time.Time) []slackgo.MsgOption {
	headline := fmt.Sprintf("%s *%s* · %s", statusEmoji(alert.Status), statusLabel(alert.Status), escape(titleWithPriority(alert)))

	var blocks []slackgo.Block

	if alert.Summary != "" {
		blocks = append(blocks, plainSection(alert.Summary))
	}

	fields := []*slackgo.TextBlockObject{
		mrkdwn("*Started*\n" + slackDate(startedAt)),
		mrkdwn("*Duration*\n" + durationText(alert.Status, startedAt, now)),
	}
	if alert.Scope != "" {
		fields = append(fields, mrkdwn("*Scope*\n`"+escape(alert.Scope)+"`"))
	}
	if alert.Hostname != "" {
		fields = append(fields, mrkdwn("*Host*\n`"+escape(alert.Hostname)+"`"))
	}
	blocks = append(blocks, slackgo.NewSectionBlock(nil, fields, nil))

	// Graph of the latest event, replaced on every update
	if alert.Snapshot != "" {
		blocks = append(blocks, slackgo.NewImageBlock(alert.Snapshot, "metric graph", "", nil))
	}

	footer := links(alert)
	footer = append(footer, alert.Source)
	if alert.OrgName != "" {
		footer = append(footer, escape(alert.OrgName))
	}
	footer = append(footer, "updated "+slackTime(now))
	blocks = append(blocks, slackgo.NewContextBlock("", mrkdwn(strings.Join(footer, " · "))))

	return message(headline, alert.Status, blocks)
}

// buildDetail formats a thread reply holding the full detail of one event.
//
//	🔴 ALERT at 10:00
//	┃ cpu over host:api was > 90.0 during the last 5m
//	┃ <message>
//	┃ <graph snapshot>
//	┃ View event · Monitor
func buildDetail(alert *domain.Alert) []slackgo.MsgOption {
	headline := fmt.Sprintf("%s *%s* at %s", statusEmoji(alert.Status), statusLabel(alert.Status), slackTime(alert.EventTime))

	var blocks []slackgo.Block

	if alert.Summary != "" {
		blocks = append(blocks, plainSection(alert.Summary))
	}

	if msg := toMrkdwn(alert.Message); msg != "" && alert.Message != alert.Summary {
		blocks = append(blocks, slackgo.NewSectionBlock(mrkdwn(truncate(msg, maxSectionText)), nil, nil))
	}

	if alert.Query != "" {
		blocks = append(blocks, slackgo.NewContextBlock("", mrkdwn("Query: `"+escape(alert.Query)+"`")))
	}

	if alert.Snapshot != "" {
		blocks = append(blocks, slackgo.NewImageBlock(alert.Snapshot, "metric graph", "", nil))
	}

	if l := links(alert); len(l) > 0 {
		blocks = append(blocks, slackgo.NewContextBlock("", mrkdwn(strings.Join(l, " · "))))
	}

	return message(headline, alert.Status, blocks)
}

// message puts the headline as text (also used for notifications) and the blocks in a colored attachment
func message(headline string, status domain.AlertStatus, blocks []slackgo.Block) []slackgo.MsgOption {
	return []slackgo.MsgOption{
		slackgo.MsgOptionText(headline, false),
		slackgo.MsgOptionAttachments(slackgo.Attachment{
			Color:  statusColor(status),
			Blocks: slackgo.Blocks{BlockSet: blocks},
		}),
	}
}

func titleWithPriority(alert *domain.Alert) string {
	if alert.Priority == "" {
		return alert.Title
	}
	return fmt.Sprintf("[%s] %s", alert.Priority, alert.Title)
}

func links(alert *domain.Alert) []string {
	var l []string
	if alert.Link != "" {
		l = append(l, fmt.Sprintf("<%s|View event>", alert.Link))
	}
	if alert.MonitorLink != "" {
		l = append(l, fmt.Sprintf("<%s|Monitor>", alert.MonitorLink))
	}
	return l
}

func durationText(status domain.AlertStatus, startedAt, now time.Time) string {
	d := now.Sub(startedAt)
	switch {
	case status == domain.AlertStatusRecovered:
		return "resolved after " + humanDuration(d)
	case d < time.Minute:
		return "just started"
	default:
		return "ongoing for " + humanDuration(d)
	}
}

// humanDuration renders e.g. 45s, 6m, 1h 5m, 2d 3h
func humanDuration(d time.Duration) string {
	d = d.Round(time.Second)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
	default:
		return fmt.Sprintf("%dd %dh", int(d.Hours())/24, int(d.Hours())%24)
	}
}

// slackDate renders in the reader's timezone, e.g. "Oct 20th at 3:26 PM"
func slackDate(t time.Time) string {
	return fmt.Sprintf("<!date^%d^{date_short_pretty} at {time}|%s>", t.Unix(), t.UTC().Format(time.RFC1123))
}

// slackTime renders in the reader's timezone, e.g. "3:26 PM"
func slackTime(t time.Time) string {
	return fmt.Sprintf("<!date^%d^{time}|%s>", t.Unix(), t.UTC().Format(time.Kitchen))
}

var (
	reHeading  = regexp.MustCompile(`(?m)^#{1,6}\s*(.+)$`)
	reBold     = regexp.MustCompile(`\*\*(.+?)\*\*`)
	reMdLink   = regexp.MustCompile(`\[([^\]]+)\]\(([^)\s]+)\)`)
	reTooBlank = regexp.MustCompile(`\n{3,}`)
)

// toMrkdwn converts common markdown (headings, **bold**, [text](url)) to Slack mrkdwn
func toMrkdwn(s string) string {
	s = escape(s)
	s = reHeading.ReplaceAllString(s, "*$1*")
	s = reBold.ReplaceAllString(s, "*$1*")
	s = reMdLink.ReplaceAllString(s, "<$2|$1>")
	s = reTooBlank.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}

// escape protects user content from being parsed as Slack control sequences
func escape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(s)
}

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}

func mrkdwn(text string) *slackgo.TextBlockObject {
	return slackgo.NewTextBlockObject(slackgo.MarkdownType, text, false, false)
}

// plainSection shows text as is, so characters like * in metric scopes are not parsed as formatting
func plainSection(text string) *slackgo.SectionBlock {
	return slackgo.NewSectionBlock(slackgo.NewTextBlockObject(slackgo.PlainTextType, truncate(text, maxSectionText), false, false), nil, nil)
}

func statusLabel(status domain.AlertStatus) string {
	switch status {
	case domain.AlertStatusTriggered:
		return "ALERT"
	case domain.AlertStatusWarn:
		return "WARNING"
	case domain.AlertStatusNoData:
		return "NO DATA"
	case domain.AlertStatusRecovered:
		return "RESOLVED"
	default:
		return strings.ToUpper(string(status))
	}
}

func statusColor(status domain.AlertStatus) string {
	switch status {
	case domain.AlertStatusTriggered:
		return colorDanger
	case domain.AlertStatusWarn:
		return colorWarning
	case domain.AlertStatusRecovered:
		return colorGood
	default:
		return colorDefault
	}
}

func statusEmoji(status domain.AlertStatus) string {
	switch status {
	case domain.AlertStatusTriggered:
		return ":red_circle:"
	case domain.AlertStatusWarn:
		return ":large_orange_circle:"
	case domain.AlertStatusNoData:
		return ":white_circle:"
	case domain.AlertStatusRecovered:
		return ":white_check_mark:"
	default:
		return ":grey_question:"
	}
}
