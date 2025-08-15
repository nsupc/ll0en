package main

import (
	"fmt"
	"ll0en/pkg/config"
	"ll0en/pkg/ns"
	"ll0en/pkg/sse"
	"log/slog"
	"os"
	"regexp"
	"strconv"

	"github.com/nsupc/eurogo/client"
	"github.com/nsupc/eurogo/telegrams"
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
	sse.New(happeningsUrl).Subscribe(func(e sse.Event) {
		if resignRegex.Match([]byte(e.Text)) {
			slog.Debug("resignation", slog.String("event", e.Text))

			go func() {
				matches := resignRegex.FindStringSubmatch(e.Text)
				nationName := matches[1]

				tmpl, err := eurocoreClient.GetTemplate(config.Telegrams.Resign.Template)
				if err != nil {
					slog.Error("unable to retrieve telegram template", slog.Any("error", err))
					return
				}

				telegram := telegrams.New(tmpl.Nation, nationName, strconv.Itoa(tmpl.Tgid), tmpl.Key, telegrams.Standard)

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

		if moveRegex.Match([]byte(e.Text)) {
			slog.Debug("move", slog.String("event", e.Text))

			go func() {
				matches := moveRegex.FindStringSubmatch(e.Text)
				nationName := matches[1]

				eligibility, err := nsClient.IsRecruitmentEligible(nationName, config.Region)
				if err != nil {
					slog.Error("unable to retrieve recruitment eligibility", slog.Any("error", err))
					return
				}

				if eligibility.CanRecruit {
					tmpl, err := eurocoreClient.GetTemplate(config.Telegrams.Move.Template)
					if err != nil {
						slog.Error("unable to retrieve telegram template", slog.Any("error", err))
						return
					}

					telegram := telegrams.New(tmpl.Nation, nationName, strconv.Itoa(tmpl.Tgid), tmpl.Key, telegrams.Recruitment)

					err = eurocoreClient.SendTelegram(telegram)
					if err != nil {
						slog.Error("unable to send move telegram", slog.Any("error", err))
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
