package gpx

import (
	"fmt"
	"time"
)

func GetInfo(gpxFile *GPX) Info {
	info := Info{}

	getGpxElementInfo(&info, gpxFile)

	if len(gpxFile.Tracks) > 0 {
		info.Tracks = len(gpxFile.Tracks)

		for _, t := range gpxFile.Tracks {
			info.Segments = append(info.Segments, len(t.Segments))

			for _, s := range t.Segments {
				si := Info{}

				getGpxElementInfo(&si, &s)

				info.SegmentsInfo = append(info.SegmentsInfo, si)
			}
		}
	}

	if len(gpxFile.Waypoints) > 0 {
		info.Waypoints = len(gpxFile.Waypoints)
	}

	if len(gpxFile.Routes) > 0 {
		info.Routes = len(gpxFile.Routes)
	}

	return info
}

func fl(f float64) string {
	return fmt.Sprintf("%.2f", f)
}

func (info Info) String() string {
	result := ""

	result += fmt.Sprintln("Points: ", info.Points)
	result += fmt.Sprintln("Length 2D: ", fl(info.Length2DKM), "km")
	result += fmt.Sprintln("Length 3D: ", fl(info.Length3DKM), "km")

	bounds := info.Bounds
	result += fmt.Sprintln("Bounds: ",
		bounds.MinLatitude, bounds.MaxLatitude,
		bounds.MinLongitude, bounds.MaxLongitude)

	result += fmt.Sprintln("Moving time: ", info.MovingTime.String())
	result += fmt.Sprintln("Stopped time: ", info.StoppedTime.String())

	result += fmt.Sprintln("Avg speed: ", fl(info.AvgSpdMPS), "m/s = ", fl(info.AvgSpdKPH), "km/h")
	result += fmt.Sprintln("Max speed: ", fl(info.MaxSpdMPS), "m/s = ", fl(info.MaxSpdKPH), "km/h")

	result += fmt.Sprintln("Total uphill: ", fl(info.UphillM), "m")
	result += fmt.Sprintln("Total downhill: ", fl(info.DownhillM), "m")

	result += fmt.Sprintln("Started: ", info.Start.Format(time.RFC3339))
	result += fmt.Sprintln("Ended: ", info.End.Format(time.RFC3339))

	if info.Tracks > 0 {
		result += fmt.Sprintln("Tracks:", info.Tracks)

		for i, s := range info.Segments {
			result += fmt.Sprintln("Track", i+1, "segments:", s)
		}
	}

	if info.Waypoints > 0 {
		result += fmt.Sprintln("Waypoints:", info.Waypoints)
	}

	if info.Routes > 0 {
		result += fmt.Sprintln("Routes:", info.Routes)
	}

	if info.ShowSegments {
		for i, s := range info.SegmentsInfo {
			result += fmt.Sprintln("\nSegment", i+1)

			result += s.String()
		}
	}

	return result
}

type Info struct {
	Points     int     `json:"points,omitempty"`
	Length2DKM float64 `json:"length_2d_km,omitempty"`
	Length3DKM float64 `json:"length_3d_km,omitempty"`

	Bounds      GpxBounds     `json:"bounds"`
	MovingTime  time.Duration `json:"moving_time,omitempty"`
	StoppedTime time.Duration `json:"stopped_time,omitempty"`

	AvgSpdMPS float64 `json:"avg_spd_mps,omitempty"`
	AvgSpdKPH float64 `json:"avg_spd_kph,omitempty"`

	MaxSpdMPS float64 `json:"max_spd_mps,omitempty"`
	MaxSpdKPH float64 `json:"max_spd_kph,omitempty"`

	UphillM   float64 `json:"uphill_m,omitempty"`
	DownhillM float64 `json:"downhill_m,omitempty"`

	Start time.Time `json:"start"`
	End   time.Time `json:"end"`

	Tracks       int    `json:"tracks,omitempty"`
	Segments     []int  `json:"segments,omitempty" description:"Segments count per Track."`
	SegmentsInfo []Info `json:"segments_info,omitempty" description:"Segments info."`
	Waypoints    int    `json:"waypoints,omitempty"`
	Routes       int    `json:"routes,omitempty"`

	ShowSegments bool `json:"-"`
}

// getGpxElementInfo pretty prints some basic information about this GPX elements.
func getGpxElementInfo(info *Info, gpxDoc GPXElementInfo) string {
	info.Points = gpxDoc.GetTrackPointsNo()
	info.Length2DKM = gpxDoc.Length2D() / 1000.0
	info.Length3DKM = gpxDoc.Length3D() / 1000.0

	result := ""

	info.Bounds = gpxDoc.Bounds()

	md := gpxDoc.MovingData()
	info.MovingTime = time.Duration(md.MovingTime * float64(time.Second))
	info.StoppedTime = time.Duration(md.StoppedTime * float64(time.Second))

	info.AvgSpdMPS = md.MovingDistance / md.MovingTime // m/s
	info.AvgSpdKPH = info.AvgSpdMPS * 60 * 60 / 1000.0

	info.MaxSpdMPS = md.MaxSpeed
	info.MaxSpdKPH = md.MaxSpeed * 60 * 60 / 1000.0

	updo := gpxDoc.UphillDownhill()
	info.UphillM = updo.Uphill
	info.DownhillM = updo.Downhill

	timeBounds := gpxDoc.TimeBounds()
	info.Start = timeBounds.StartTime
	info.End = timeBounds.EndTime

	return result
}
