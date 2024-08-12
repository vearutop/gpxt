package sigma

import "encoding/xml"

// Activity was generated 2024-08-01 16:52:14 by https://xml-to-go.github.io/ in Ukraine.
type Activity struct {
	XMLName  xml.Name `xml:"Activity"`
	Text     string   `xml:",chardata"`
	FileDate string   `xml:"fileDate,attr"`
	Revision string   `xml:"revision,attr"`
	Computer struct {
		Text         string `xml:",chardata"`
		Unit         string `xml:"unit,attr"`
		ActivityType string `xml:"activityType,attr"`
		DateCode     string `xml:"dateCode,attr"`
	} `xml:"Computer"`
	GeneralInformation struct {
		Text string `xml:",chardata"`
		User struct {
			Text   string `xml:",chardata"`
			Color  string `xml:"color,attr"`
			Gender string `xml:"gender,attr"`
		} `xml:"user"`
		Sport                       string  `xml:"sport"`
		GUID                        string  `xml:"GUID"`
		AltitudeDifferencesDownhill int     `xml:"altitudeDifferencesDownhill"`
		AltitudeDifferencesUphill   int     `xml:"altitudeDifferencesUphill"`
		AverageCadence              float64 `xml:"averageCadence"`
		AverageHeartrate            float64 `xml:"averageHeartrate"`
		AverageInclineDownhill      string  `xml:"averageInclineDownhill"`
		AverageInclineUphill        string  `xml:"averageInclineUphill"`
		AveragePower                string  `xml:"averagePower"`
		AverageSpeed                string  `xml:"averageSpeed"`
		Bike                        string  `xml:"bike"`
		Calories                    string  `xml:"calories"`
		DataType                    string  `xml:"dataType"`
		Description                 string  `xml:"description"`
		Distance                    string  `xml:"distance"`
		DistanceDownhill            string  `xml:"distanceDownhill"`
		DistanceUphill              string  `xml:"distanceUphill"`
		ExternalLink                string  `xml:"externalLink"`
		HrMax                       string  `xml:"hrMax"`
		IntensityZone1Start         string  `xml:"intensityZone1Start"`
		IntensityZone2Start         string  `xml:"intensityZone2Start"`
		IntensityZone3Start         string  `xml:"intensityZone3Start"`
		IntensityZone4Start         string  `xml:"intensityZone4Start"`
		IntensityZone4End           string  `xml:"intensityZone4End"`
		LinkedRouteId               string  `xml:"linkedRouteId"`
		LowerLimit                  string  `xml:"lowerLimit"`
		ManualTemperature           string  `xml:"manualTemperature"`
		MaximumAltitude             string  `xml:"maximumAltitude"`
		MaximumCadence              float64 `xml:"maximumCadence"`
		MaximumHeartrate            string  `xml:"maximumHeartrate"`
		MaximumInclineDownhill      string  `xml:"maximumInclineDownhill"`
		MaximumInclineUphill        string  `xml:"maximumInclineUphill"`
		MaximumPower                string  `xml:"maximumPower"`
		MaximumSpeed                string  `xml:"maximumSpeed"`
		MinimumAltitude             string  `xml:"minimumAltitude"`
		MinimumHeartrate            string  `xml:"minimumHeartrate"`
		Name                        string  `xml:"name"`
		PauseTime                   string  `xml:"pauseTime"`
		PowerZone1Start             string  `xml:"powerZone1Start"`
		PowerZone2Start             string  `xml:"powerZone2Start"`
		PowerZone3Start             string  `xml:"powerZone3Start"`
		PowerZone4Start             string  `xml:"powerZone4Start"`
		PowerZone5Start             string  `xml:"powerZone5Start"`
		PowerZone6Start             string  `xml:"powerZone6Start"`
		PowerZone7Start             string  `xml:"powerZone7Start"`
		PowerZone7End               string  `xml:"powerZone7End"`
		Rating                      string  `xml:"rating"`
		Feeling                     string  `xml:"feeling"`
		SamplingRate                string  `xml:"samplingRate"`
		StartDate                   string  `xml:"startDate"`
		Statistic                   string  `xml:"statistic"`
		TimeInIntensityZone1        string  `xml:"timeInIntensityZone1"`
		TimeInIntensityZone2        string  `xml:"timeInIntensityZone2"`
		TimeInIntensityZone3        string  `xml:"timeInIntensityZone3"`
		TimeInIntensityZone4        string  `xml:"timeInIntensityZone4"`
		TimeInPowerZone1            string  `xml:"timeInPowerZone1"`
		TimeInPowerZone2            string  `xml:"timeInPowerZone2"`
		TimeInPowerZone3            string  `xml:"timeInPowerZone3"`
		TimeInPowerZone4            string  `xml:"timeInPowerZone4"`
		TimeInPowerZone5            string  `xml:"timeInPowerZone5"`
		TimeInPowerZone6            string  `xml:"timeInPowerZone6"`
		TimeInPowerZone7            string  `xml:"timeInPowerZone7"`
		TimeInZone                  string  `xml:"timeInZone"`
		TimeOverIntensityZone       string  `xml:"timeOverIntensityZone"`
		TimeOverZone                string  `xml:"timeOverZone"`
		TimeUnderIntensityZone      string  `xml:"timeUnderIntensityZone"`
		TimeUnderZone               string  `xml:"timeUnderZone"`
		TrackProfile                string  `xml:"trackProfile"`
		TrainingTime                string  `xml:"trainingTime"`
		TrainingType                string  `xml:"trainingType"`
		TrainingZone                string  `xml:"trainingZone"`
		UnitId                      string  `xml:"unitId"`
		UpperLimit                  string  `xml:"upperLimit"`
		Weather                     string  `xml:"weather"`
		Wind                        string  `xml:"wind"`
		ActivityStatus              string  `xml:"activityStatus"`
		SharingInfo                 string  `xml:"sharingInfo"`
		Participant                 string  `xml:"Participant"`
	} `xml:"GeneralInformation"`
	Entries struct {
		Text  string  `xml:",chardata"`
		Entry []Entry `xml:"Entry"`
	} `xml:"Entries"`
	Markers struct {
		Text   string `xml:",chardata"`
		Marker []struct {
			Text                   string  `xml:",chardata"`
			AltitudeDownhill       string  `xml:"altitudeDownhill,attr"`
			AltitudeUphill         string  `xml:"altitudeUphill,attr"`
			AverageCadence         float64 `xml:"averageCadence,attr"`
			AverageHeartrate       string  `xml:"averageHeartrate,attr"`
			AverageInclineDownhill string  `xml:"averageInclineDownhill,attr"`
			AverageInclineUphill   string  `xml:"averageInclineUphill,attr"`
			AveragePower           string  `xml:"averagePower,attr"`
			AverageSpeed           string  `xml:"averageSpeed,attr"`
			Calories               string  `xml:"calories,attr"`
			Description            string  `xml:"description,attr"`
			Distance               string  `xml:"distance,attr"`
			DistanceAbsolute       string  `xml:"distanceAbsolute,attr"`
			DistanceDownhill       string  `xml:"distanceDownhill,attr"`
			DistanceUphill         string  `xml:"distanceUphill,attr"`
			Duration               int     `xml:"duration,attr"`
			MaximumAltitude        int     `xml:"maximumAltitude,attr"`
			MaximumCadence         float64 `xml:"maximumCadence,attr"`
			MaximumHeartrate       string  `xml:"maximumHeartrate,attr"`
			MaximumInclineDownhill string  `xml:"maximumInclineDownhill,attr"`
			MaximumInclineUphill   string  `xml:"maximumInclineUphill,attr"`
			MaximumPower           string  `xml:"maximumPower,attr"`
			MaximumSpeed           string  `xml:"maximumSpeed,attr"`
			Number                 string  `xml:"number,attr"`
			Time                   string  `xml:"time,attr"`
			TimeAbsolute           int     `xml:"timeAbsolute,attr"`
			Title                  string  `xml:"title,attr"`
			Type                   string  `xml:"type,attr"`
		} `xml:"Marker"`
	} `xml:"Markers"`
}

type Entry struct {
	Text                        string   `xml:",chardata"`
	Altitude                    int      `xml:"altitude,attr"`
	AltitudeDifferencesDownhill int      `xml:"altitudeDifferencesDownhill,attr"`
	AltitudeDifferencesUphill   int      `xml:"altitudeDifferencesUphill,attr"`
	Cadence                     *float64 `xml:"cadence,attr"`
	Calories                    float64  `xml:"calories,attr"`
	Distance                    float64  `xml:"distance,attr"`
	DistanceAbsolute            float64  `xml:"distanceAbsolute,attr"`
	DistanceDownhill            string   `xml:"distanceDownhill,attr"`
	DistanceUphill              string   `xml:"distanceUphill,attr"`
	Heartrate                   *float64 `xml:"heartrate,attr"`
	Incline                     string   `xml:"incline,attr"`
	Power                       *float64 `xml:"power,attr"`
	Speed                       string   `xml:"speed,attr"`
	Temperature                 string   `xml:"temperature,attr"`
	TrainingTime                string   `xml:"trainingTime,attr"`
	TrainingTimeAbsolute        int      `xml:"trainingTimeAbsolute,attr"` // seconds * 100
	PowerZone                   string   `xml:"powerZone,attr"`
	TimeBelowIntensityZones     string   `xml:"timeBelowIntensityZones,attr"`
	TimeInIntensityZone1        string   `xml:"timeInIntensityZone1,attr"`
	TimeInIntensityZone2        string   `xml:"timeInIntensityZone2,attr"`
	TimeInIntensityZone3        string   `xml:"timeInIntensityZone3,attr"`
	TimeInIntensityZone4        string   `xml:"timeInIntensityZone4,attr"`
	TimeAboveIntensityZones     string   `xml:"timeAboveIntensityZones,attr"`
	TimeBelowTargetZone         string   `xml:"timeBelowTargetZone,attr"`
	TimeInTargetZone            string   `xml:"timeInTargetZone,attr"`
	TimeAboveTargetZone         string   `xml:"timeAboveTargetZone,attr"`
	UseForChart                 string   `xml:"useForChart,attr"`
	UseForTrack                 string   `xml:"useForTrack,attr"`
	SpeedTime                   string   `xml:"speedTime,attr"`
}
