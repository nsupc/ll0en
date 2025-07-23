package main

import (
	"encoding/json"
	"fmt"
	"ll0en/pkg/config"
	"ll0en/pkg/ns"
	"ll0en/pkg/sse"
	"log/slog"
	"os"
	"regexp"
	"strconv"

	"github.com/nsupc/eurogo/client"
	"github.com/nsupc/eurogo/models"
	gsse "github.com/tmaxmax/go-sse"
)

func main() {
	config, err := config.Read()
	if err != nil {
		slog.Error("", slog.Any("error", err))
		os.Exit(1)
	}

	resignRegex := regexp.MustCompile("^@@([a-z0-9_-]+)@@ resigned from the World Assembly.?$")
	moveRegex := regexp.MustCompile(fmt.Sprintf(`^@@([a-z0-9_-]+)@@ relocated from %%%%%s%%%% to %%%%[a-z0-9_-]+%%%%.?$`, config.Region))

	nsClient := ns.NewClient(config.User, config.MaxRequests)
	eurocoreClient := client.New(config.Eurocore.Username, config.Eurocore.Password, config.Eurocore.BaseUrl)

	happeningsUrl := fmt.Sprintf("https://www.nationstates.net/api/region:%s", config.Region)
	err = sse.New().Subscribe(happeningsUrl, func(e gsse.Event) {
		event := sse.Event{}

		err = json.Unmarshal([]byte(e.Data), &event)
		if err != nil {
			slog.Error("unable to unmarshal event", slog.Any("error", e))
			return
		}

		if resignRegex.Match([]byte(event.Text)) {
			slog.Debug("resignation", slog.String("event", event.Text))

			go func() {
				matches := resignRegex.FindStringSubmatch(event.Text)
				nationName := matches[1]

				var telegram models.NewTelegram

				if config.Telegrams.Resign.Template != "" {
					tmpl, err := eurocoreClient.GetTemplate(config.Telegrams.Resign.Template)
					if err != nil {
						slog.Error("unable to retrieve telegram template", slog.Any("error", err))
						return
					}

					telegram.Id = strconv.Itoa(tmpl.Tgid)
					telegram.Secret = tmpl.Key
					telegram.Recipient = nationName
					telegram.Sender = tmpl.Nation
					telegram.Type = "standard"
				} else {
					telegram.Id = strconv.Itoa(config.Telegrams.Resign.Id)
					telegram.Secret = config.Telegrams.Resign.Key
					telegram.Recipient = nationName
					telegram.Sender = config.Telegrams.Resign.Author
					telegram.Type = "standard"
				}

				err = eurocoreClient.SendTelegram(telegram)
				if err != nil {
					slog.Error("unable to send resign telegram", slog.Any("error", err))
					return
				} else {
					slog.Info("resign telegram sent", slog.String("nation", nationName))
				}
			}()

			return
		}

		if moveRegex.Match([]byte(event.Text)) {
			slog.Debug("move", slog.String("event", event.Text))

			go func() {
				matches := moveRegex.FindStringSubmatch(event.Text)
				nationName := matches[1]

				eligibility, err := nsClient.IsRecruitmentEligible(nationName, config.Region)
				if err != nil {
					slog.Error("unable to retrieve recruitment eligibility", slog.Any("error", err))
					return
				}

				if eligibility.CanRecruit {
					var telegram models.NewTelegram

					if config.Telegrams.Move.Template != "" {
						tmpl, err := eurocoreClient.GetTemplate(config.Telegrams.Move.Template)
						if err != nil {
							slog.Error("unable to retrieve telegram template", slog.Any("error", err))
							return
						}

						telegram.Id = strconv.Itoa(tmpl.Tgid)
						telegram.Secret = tmpl.Key
						telegram.Recipient = nationName
						telegram.Sender = tmpl.Nation
						telegram.Type = "recruitment"
					} else {
						telegram.Id = strconv.Itoa(config.Telegrams.Move.Id)
						telegram.Secret = config.Telegrams.Move.Key
						telegram.Recipient = nationName
						telegram.Sender = config.Telegrams.Move.Author
						telegram.Type = "recruitment"
					}

					err = eurocoreClient.SendTelegram(telegram)
					if err != nil {
						slog.Error("unable to send resign telegram", slog.Any("error", err))
						return
					} else {
						slog.Info("move telegram sent", slog.String("nation", nationName))
					}
				} else {
					slog.Info("nation not eligible for recruitment", slog.String("nation", nationName), slog.String("region", eligibility.Region))
				}
			}()

			return
		}
	})
	if err != nil {
		slog.Error("unable to subscribe to happenings", slog.Any("error", err))
	}
}
