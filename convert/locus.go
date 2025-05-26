// Package convert provides tools to import bespoke GPX flavors.
package convert

import (
	"encoding/xml"
	"time"

	"github.com/tkrajina/gpxgo/gpx"
	"github.com/vearutop/gpxt/ns"
)

// FromLocus reads Locus Map for Android GPX format.
func FromLocus(data []byte) (gpx.GPX, error) {
	var (
		result gpx.GPX
		input  LocusGpx
	)

	err := xml.Unmarshal(data, &input)
	if err != nil {
		return result, err
	}

	result.Time = &input.Metadata.Time
	result.Description = input.Rte.Desc
	result.Creator = input.Creator
	result.Name = input.Rte.Name

	for _, rp := range input.Rte.Rtept {
		p := gpx.GPXPoint{}

		p.Latitude = rp.Lat
		p.Longitude = rp.Lon
		p.Elevation = *gpx.NewNullableFloat64(rp.Ele)
		p.Timestamp = rp.Time
		p.HorizontalDilution = *gpx.NewNullableFloat64(rp.Hdop)

		if rp.Extensions.TrackPointExtension.Course != "" {
			p.Extensions.GetOrCreateNode(ns.TpxNs, ns.TpxPath, "course").Data = rp.Extensions.TrackPointExtension.Course
		}

		result.AppendPoint(&p)
	}

	return result, err
}

// LocusGpx is mapping of Locus Map, Android GPX flavor.
type LocusGpx struct {
	XMLName        xml.Name `xml:"gpx"`
	Text           string   `xml:",chardata"`
	Version        string   `xml:"version,attr"`
	Creator        string   `xml:"creator,attr"`
	Xmlns          string   `xml:"xmlns,attr"`
	Xsi            string   `xml:"xsi,attr"`
	SchemaLocation string   `xml:"schemaLocation,attr"`
	Gpxx           string   `xml:"gpxx,attr"`
	Gpxtrkx        string   `xml:"gpxtrkx,attr"`
	Gpxtpx         string   `xml:"gpxtpx,attr"`
	Locus          string   `xml:"locus,attr"`
	Metadata       struct {
		Text string    `xml:",chardata"`
		Desc string    `xml:"desc"`
		Time time.Time `xml:"time"`
	} `xml:"metadata"`
	Rte struct {
		Text       string `xml:",chardata"`
		Name       string `xml:"name"`
		Desc       string `xml:"desc"`
		Extensions struct {
			Text string `xml:",chardata"`
			Line struct {
				Text       string `xml:",chardata"`
				Xmlns      string `xml:"xmlns,attr"`
				Color      string `xml:"color"`
				Opacity    string `xml:"opacity"`
				Width      string `xml:"width"`
				Extensions struct {
					Text        string `xml:",chardata"`
					LsColorBase string `xml:"lsColorBase"`
					LsColoring  string `xml:"lsColoring"`
					LsWidth     string `xml:"lsWidth"`
					LsUnits     string `xml:"lsUnits"`
				} `xml:"extensions"`
			} `xml:"line"`
			Activity string `xml:"activity"`
		} `xml:"extensions"`
		Rtept []struct {
			Text       string    `xml:",chardata"`
			Lat        float64   `xml:"lat,attr"`
			Lon        float64   `xml:"lon,attr"`
			Ele        float64   `xml:"ele"`
			Time       time.Time `xml:"time"`
			Hdop       float64   `xml:"hdop"`
			Extensions struct {
				Text                string `xml:",chardata"`
				TrackPointExtension struct {
					Text   string `xml:",chardata"`
					Course string `xml:"course"`
				} `xml:"TrackPointExtension"`
			} `xml:"extensions"`
		} `xml:"rtept"`
	} `xml:"rte"`
}
