package ns

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type RecruitmentStatus struct {
	Name       string
	Region     string
	CanRecruit bool
}

func (r *RecruitmentStatus) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, attr := range start.Attr {
		if attr.Name.Local == "id" {
			r.Name = attr.Value
		}
	}

	type Aux struct {
		Region     string `xml:"REGION"`
		CanRecruit string `xml:"TGCANRECRUIT"`
	}

	var aux Aux
	if err := d.DecodeElement(&aux, &start); err != nil {
		return err
	}

	r.Region = strings.ToLower(strings.ReplaceAll(aux.Region, " ", "_"))
	r.CanRecruit = aux.CanRecruit == "1"

	return nil
}

func (c *Client) IsRecruitmentEligible(name string, region string) (*RecruitmentStatus, error) {
	nationName := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(name)), " ", "_")
	regionName := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(region)), " ", "_")

	url := fmt.Sprintf("https://www.nationstates.net/cgi-bin/api.cgi?nation=%s&q=region+tgcanrecruit;from=%s", nationName, regionName)

	req, err := http.NewRequest("GET", url, http.NoBody)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	status := RecruitmentStatus{}
	err = xml.Unmarshal(body, &status)
	if err != nil {
		return nil, err
	}

	return &status, nil
}
