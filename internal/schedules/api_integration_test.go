package schedules

import (
	"encoding/json"
	"flag"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/CanalTP/gonavitia"
	"github.com/CanalTP/gormungandr"
	"github.com/CanalTP/gormungandr/internal/checker"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var kraken *gormungandr.Kraken

func init() {
	gin.SetMode(gin.TestMode)
}

func TestMain(m *testing.M) {
	flag.Parse() //required to get Short() from testing
	if testing.Short() {
		log.Warn("skipping test Docker in short mode.")
		os.Exit(m.Run())
	}

	mockManager, err := checker.NewMockManager()
	if err != nil {
		log.Fatalf("Could not initialize mocks: %s", err)
	}
	kraken, err = mockManager.DepartureBoardTest()
	if err != nil {
		log.Fatalf("Could not start departure_board_test: %s", err)
	}
	//Run tests
	code := m.Run()

	// You can't defer this because os.Exit doesn't care for defer
	mockManager.Close()

	os.Exit(code)
}

func TestRouteSchedules(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test Docker in short mode.")
	}
	//t.Parallel()
	assert := assert.New(t)
	require := require.New(t)
	c, engine := gin.CreateTestContext(httptest.NewRecorder())
	SetupApi(engine, kraken, &NullPublisher{}, SkipAuth())

	c.Request = httptest.NewRequest("GET", "/v1/coverage/foo/routes/line:A:0/route_schedules?from_datetime=20120615T080000", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, c.Request)
	require.Equal(200, w.Code)

	var response gonavitia.RouteScheduleResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.Nil(err)

	require.Len(response.RouteSchedules, 1)
	require.NotNil(response.Context)
	assert.Equal("UTC", response.Context.Timezone)
	schedules := response.RouteSchedules[0]
	checker.IsValidRouteSchedule(t, schedules)

	scheduleLinks := make(map[string]string)
	for _, l := range schedules.Links {
		scheduleLinks[*l.Type] = *l.Id
	}
	assert.Equal("line:A", scheduleLinks["line"])
	assert.Equal("line:A:0", scheduleLinks["route"])
	assert.Equal("base_network", scheduleLinks["network"])

	require.Len(schedules.Table.Headers, 4)

	headsigns := []string{}
	headerByHeadsign := make(map[string]*gonavitia.Header)
	for _, h := range schedules.Table.Headers {
		headsigns = append(headsigns, *h.DisplayInfo.Headsign)
		headerByHeadsign[*h.DisplayInfo.Headsign] = h
	}
	assert.ElementsMatch([]string{"week", "week_bis", "all", "wednesday"}, headsigns)

	headerLinks := make(map[string]string)
	for _, l := range headerByHeadsign["all"].Links {
		headerLinks[*l.Type] = *l.Id
	}
	assert.Equal("all", headerLinks["vehicle_journey"])
	assert.Equal("physical_mode:0", headerLinks["physical_mode"])

	//TODO tests on notes when implemented

}
