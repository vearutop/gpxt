package runnerup_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/swaggest/assertjson"
	"github.com/vearutop/gpxt/runnerup"
)

func TestRepository_ListActivities(t *testing.T) {
	r, err := runnerup.NewRepository("testdata/runnerup.db.export.sqlite")
	require.NoError(t, err)

	testing.Coverage()

	defer func() {
		require.NoError(t, r.Close())
	}()

	l, err := r.ListActivities(context.Background(), 2)
	require.NoError(t, err)

	assertjson.EqualMarshal(t, []byte(`[
	  {
		"ID":1259,"StartTime":1689427398,"Distance":9543.745832465589,"Time":12235,
		"Type":2
	  },
	  {
		"ID":1258,"StartTime":1689417990,"Distance":7253.769977974705,"Time":9265,
		"Type":2
	  }
	]`), l)
}

func TestRepository_ListLocations(t *testing.T) {
	r, err := runnerup.NewRepository("testdata/runnerup.db.export.sqlite")
	require.NoError(t, err)

	testing.Coverage()

	defer func() {
		require.NoError(t, r.Close())
	}()

	l, err := r.ListLocations(context.Background(), 1258)
	require.NoError(t, err)

	assertjson.EqualMarshal(t, []byte(`[
	  {
		"ID":2832270,"ActivityID":1258,"Time":1689417989999,
		"Lon":13.575752153992653,"Lat":52.451339210383594,"Alt":25.64920086359386,
		"Accuracy":7.5,"Speed":0.4399999976158142,"Bearing":0,"Satellites":29
	  },
	  {
		"ID":2832271,"ActivityID":1258,"Time":1689417990997,
		"Lon":13.575776377692819,"Lat":52.45134289842099,"Alt":34.64936108978178,
		"Accuracy":7.5,"Speed":0.5699999928474426,"Bearing":0,"Satellites":30
	  },
	  {
		"ID":2832272,"ActivityID":1258,"Time":1689417992003,"Lon":13.57574905268848,
		"Lat":52.45128137525171,"Alt":30.64943612303503,"Accuracy":5.5,
		"Speed":0.3400000035762787,"Bearing":0,"Satellites":30
	  },
	  {
		"ID":2832273,"ActivityID":1258,"Time":1689417993000,
		"Lon":13.575738575309515,"Lat":52.45129717513919,"Alt":31.649299407609206,
		"Accuracy":4,"Speed":0.3100000023841858,"Bearing":0,"Satellites":29
	  },
	  {
		"ID":2832274,"ActivityID":1258,"Time":1689417994000,"Lon":13.57576422393322,
		"Lat":52.451307233422995,"Alt":32.64928274537054,"Accuracy":3,
		"Speed":0.5099999904632568,"Bearing":0,"Satellites":30
	  },
	  {
		"ID":2832275,"ActivityID":1258,"Time":1689417995000,
		"Lon":13.575763469561934,"Lat":52.45130932889879,"Alt":27.649367778414614,
		"Accuracy":2.5,"Speed":0.9899999499320984,"Bearing":0,"Satellites":27
	  },
	  {
		"ID":2832276,"ActivityID":1258,"Time":1689417996000,
		"Lon":13.575773360207677,"Lat":52.45131364557892,"Alt":26.6493674483618,
		"Accuracy":2.5,"Speed":1.1699999570846558,"Bearing":111.19999694824219,
		"Satellites":27
	  },
	  {
		"ID":2832277,"ActivityID":1258,"Time":1689417997000,
		"Lon":13.575789537280798,"Lat":52.451302371919155,"Alt":28.64940063638859,
		"Accuracy":2.5,"Speed":1.159999966621399,"Bearing":96.69999694824219,
		"Satellites":26
	  },
	  {
		"ID":2832278,"ActivityID":1258,"Time":1689417998000,
		"Lon":13.575799595564604,"Lat":52.45129755232483,"Alt":27.64943827390598,
		"Accuracy":2.5,"Speed":0.6100000143051147,"Bearing":76.80000305175781,
		"Satellites":29
	  },
	  {
		"ID":2832279,"ActivityID":1258,"Time":1689417999000,"Lon":13.57580185867846,
		"Lat":52.45130178518593,"Alt":28.64946366339022,"Accuracy":2,
		"Speed":0.4099999964237213,"Bearing":69.5,"Satellites":29
	  },
	  {
		"ID":2832280,"ActivityID":1258,"Time":1689418000000,
		"Lon":13.575803618878126,"Lat":52.4513043416664,"Alt":28.649474203266216,
		"Accuracy":2,"Speed":0.3799999952316284,"Bearing":65.9000015258789,
		"Satellites":29
	  },
	  {
		"ID":2832281,"ActivityID":1258,"Time":1689418001000,
		"Lon":13.575805043801665,"Lat":52.45130111463368,"Alt":29.649481733007846,
		"Accuracy":2,"Speed":0.38999998569488525,"Bearing":65.69999694824219,
		"Satellites":28
	  },
	  {
		"ID":2832282,"ActivityID":1258,"Time":1689418002000,"Lon":13.57579548843205,
		"Lat":52.45130140800029,"Alt":29.64948302003465,"Accuracy":2,
		"Speed":0.5999999642372131,"Bearing":81.80000305175781,"Satellites":28
	  },
	  {
		"ID":2832283,"ActivityID":1258,"Time":1689418003000,
		"Lon":13.575810324400663,"Lat":52.4513001088053,"Alt":28.649455009749133,
		"Accuracy":2,"Speed":0.44999998807907104,"Bearing":82.69999694824219,
		"Satellites":22
	  },
	  {
		"ID":2832284,"ActivityID":1258,"Time":1689418004000,
		"Lon":13.575814934447408,"Lat":52.45129738468677,"Alt":29.649497733399606,
		"Accuracy":2,"Speed":0.9599999785423279,"Bearing":105.9000015258789,
		"Satellites":27
	  },
	  {
		"ID":2832285,"ActivityID":1258,"Time":1689418005000,
		"Lon":13.575828010216355,"Lat":52.45129818096757,"Alt":29.64950890246584,
		"Accuracy":2,"Speed":0.3799999952316284,"Bearing":111.80000305175781,
		"Satellites":28
	  },
	  {
		"ID":2832286,"ActivityID":1258,"Time":1689418006000,
		"Lon":13.575841085985303,"Lat":52.45129763614386,"Alt":30.64954831935168,
		"Accuracy":2,"Speed":0.4899999797344208,"Bearing":116.19999694824219,
		"Satellites":29
	  },
	  {
		"ID":2832287,"ActivityID":1258,"Time":1689418007000,
		"Lon":13.575862124562263,"Lat":52.45130069553852,"Alt":30.64958651851162,
		"Accuracy":2,"Speed":0.6100000143051147,"Bearing":105.19999694824219,
		"Satellites":30
	  },
	  {
		"ID":2832288,"ActivityID":1258,"Time":1689418008000,
		"Lon":13.575867237523198,"Lat":52.45130861643702,"Alt":30.649651552927445,
		"Accuracy":2,"Speed":0.38999998569488525,"Bearing":97.5999984741211,
		"Satellites":32
	  },
	  {
		"ID":2832289,"ActivityID":1258,"Time":1689418009000,
		"Lon":13.575864052399993,"Lat":52.451312388293445,"Alt":30.64967387418438,
		"Accuracy":2,"Speed":0.35999998450279236,"Bearing":95.80000305175781,
		"Satellites":32
	  },
	  {
		"ID":2832290,"ActivityID":1258,"Time":1689418010000,
		"Lon":13.575871679931879,"Lat":52.45131670497358,"Alt":30.649667873246607,
		"Accuracy":2,"Speed":0.6699999570846558,"Bearing":86.80000305175781,
		"Satellites":30
	  }
	]`), l)
}
