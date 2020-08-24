package metricsgroup

import (
	"compass/internal/util"
	"compass/pkg/datasource"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jinzhu/gorm"
	"io"
	"regexp"
	"sort"
)

const (
	Active    = "ACTIVE"
	Completed = "COMPLETED"
)

type MetricsGroup struct {
	util.BaseModel
	Name        string    `json:"name"`
	Metrics     []Metric  `json:"metrics"`
	Status      string    `json:"status"`
	WorkspaceID uuid.UUID `json:"-"`
	CircleID    uuid.UUID `json:"circleId"`
}

type MetricGroupResume struct {
	util.BaseModel
	Name              string `json:"name"`
	Thresholds        int    `json:"thresholds"`
	ThresholdsReached int    `json:"thresholdsReached"`
	Metrics           int    `json:"metricsCount"`
}

func (metricsGroup MetricsGroup) Validate() []error {
	ers := make([]error, 0)

	if metricsGroup.Name == "" {
		ers = append(ers, errors.New("Name is required"))
	}

	if metricsGroup.CircleID == uuid.Nil {
		ers = append(ers, errors.New("CircleID is required"))
	}

	for _, m := range metricsGroup.Metrics {
		if len(m.Validate()) > 0 {
			ers = append(ers, m.Validate()...)
		}
	}

	return ers
}

type Condition int

const (
	EQUAL Condition = iota
	GREATER_THEN
	LOWER_THEN
)

var Periods = map[string]string{
	"s":   "s",
	"m":   "m",
	"h":   "h",
	"d":   "d",
	"w":   "w",
	"y":   "y",
	"MAX": "MAX",
}

func (c Condition) String() string {
	return [...]string{"EQUAL", "GREATER_THEN", "LOWER_THEN"}[c]
}

func (main Main) PeriodValidate(currentPeriod string) error {
	reg, err := regexp.Compile("[0-9]")
	if err != nil {
		util.Error(util.PeriodValidateRegexError, "PeriodValidate", err, currentPeriod)
		return err
	}

	if currentPeriod != "" && !reg.Match([]byte(currentPeriod)) {
		err := errors.New("Invalid period: not found number")
		util.Error(util.PeriodValidateError, "PeriodValidate", err, currentPeriod)
		return err
	}

	unit := reg.ReplaceAllString(currentPeriod, "")
	_, ok := Periods[unit]
	if !ok && currentPeriod != "" {
		err := errors.New("Invalid period: not found unit")
		util.Error(util.PeriodValidateError, "PeriodValidate", err, currentPeriod)
		return err
	}

	return nil
}

func (main Main) Parse(metricsGroup io.ReadCloser) (MetricsGroup, error) {
	var newMetricsGroup *MetricsGroup
	err := json.NewDecoder(metricsGroup).Decode(&newMetricsGroup)
	if err != nil {
		util.Error(util.GeneralParseError, "Parse", err, metricsGroup)
		return MetricsGroup{}, err
	}
	return *newMetricsGroup, nil
}

func (main Main) FindAll() ([]MetricsGroup, error) {
	var metricsGroups []MetricsGroup
	db := main.db.Set("gorm:auto_preload", true).Find(&metricsGroups)
	if db.Error != nil {
		util.Error(util.FindMetricsGroupError, "FindAll", db.Error, metricsGroups)
		return []MetricsGroup{}, db.Error
	}
	return metricsGroups, nil
}

func (main Main) getAllMetricsWithConditions(metrics []Metric) int {
	metricsWithConditions := 0
	for _, metric := range metrics {
		if metric.Condition != "" {
			metricsWithConditions++
		}
	}

	return metricsWithConditions
}

func (main Main) getAllMetricsFinished(metrics []Metric) int {
	metricsFinished := 0
	for _, metric := range metrics {
		if metric.MetricExecution.Status == Completed {
			metricsFinished++
		}
	}

	return metricsFinished
}

func (main Main) getAllMetricsInGroup(metrics []Metric) int {
	metricsTotal := 0
	for _, _ = range metrics {
		metricsTotal++
	}

	return metricsTotal
}

func (main Main) ResumeByCircle(circleId string) ([]MetricGroupResume, error) {
	var db *gorm.DB
	var metricsGroups []MetricsGroup
	var metricsGroupsResume []MetricGroupResume

	if circleId == "" {
		db = main.db.Set("gorm:auto_preload", true).Find(&metricsGroups)
	} else {
		circleIdParsed, _ := uuid.Parse(circleId)
		db = main.db.Set("gorm:auto_preload", true).Where("circle_id=?", circleIdParsed).Find(&metricsGroups)
	}

	if db.Error != nil {
		util.Error(util.ResumeByCircleError, "ResumeByCircle", db.Error, metricsGroups)
		return []MetricGroupResume{}, db.Error
	}

	for _, group := range metricsGroups {
		metricsGroupsResume = append(metricsGroupsResume, MetricGroupResume{
			group.BaseModel,
			group.Name,
			main.getAllMetricsWithConditions(group.Metrics),
			main.getAllMetricsFinished(group.Metrics),
			main.getAllMetricsInGroup(group.Metrics),
		})
	}

	main.sortResumeMetrics(metricsGroupsResume)

	return metricsGroupsResume, nil
}

func (main Main) sortResumeMetrics(metricsGroupResume []MetricGroupResume) {

	sort.SliceStable(metricsGroupResume, func(i, j int) bool {

		if (metricsGroupResume[i].ThresholdsReached == metricsGroupResume[i].Thresholds) &&
			(metricsGroupResume[j].ThresholdsReached == metricsGroupResume[j].Thresholds) &&
			(metricsGroupResume[i].ThresholdsReached > metricsGroupResume[j].ThresholdsReached) {
			return true
		}

		if metricsGroupResume[i].Thresholds == 0 {
			return false
		}

		if metricsGroupResume[j].Thresholds == 0 {
			return true
		}

		if metricsGroupResume[i].ThresholdsReached == metricsGroupResume[i].Thresholds {
			return true
		}

		if metricsGroupResume[j].ThresholdsReached == metricsGroupResume[j].Thresholds {
			return false
		}

		if metricsGroupResume[i].ThresholdsReached == 0 && metricsGroupResume[j].ThresholdsReached == 0 {
			return metricsGroupResume[i].Thresholds > metricsGroupResume[j].Thresholds
		}

		return metricsGroupResume[i].ThresholdsReached > metricsGroupResume[j].ThresholdsReached

	})
}

func (main Main) Save(metricsGroup MetricsGroup) (MetricsGroup, error) {
	metricsGroup.Status = Active
	for i := 0; i < len(metricsGroup.Metrics); i++ {
		metricsGroup.Metrics[i].Status = Active
	}
	db := main.db.Create(&metricsGroup)
	if db.Error != nil {
		util.Error(util.SaveMetricsGroupError, "Save", db.Error, metricsGroup)
		return MetricsGroup{}, db.Error
	}
	return metricsGroup, nil
}

func (main Main) FindById(id string) (MetricsGroup, error) {
	metricsGroup := MetricsGroup{}
	db := main.db.Set("gorm:auto_preload", true).Where("id = ?", id).First(&metricsGroup)
	if db.Error != nil {
		util.Error(util.FindMetricsGroupError, "FindById", db.Error, "Id = "+id)
		return MetricsGroup{}, db.Error
	}
	return metricsGroup, nil
}

func (main Main) FindActiveMetricGroups() ([]MetricsGroup, error) {
	var metricsGroups []MetricsGroup
	db := main.db.Preload("Metrics").Where("status = ?", Active).Find(&metricsGroups)
	if db.Error != nil {
		util.Error(util.FindMetricsGroupError, "FindActiveMetricGroups", db.Error, metricsGroups)
		return []MetricsGroup{}, db.Error
	}
	return metricsGroups, nil
}

func (main Main) FindCircleMetricGroups(circleId string) ([]MetricsGroup, error) {
	var metricsGroups []MetricsGroup
	db := main.db.Set("gorm:auto_preload", true).Where("circle_id = ?", circleId).Find(&metricsGroups)
	if db.Error != nil {
		util.Error(util.FindMetricsGroupError, "FindCircleMetricGroups", db.Error, "CircleId= "+circleId)
		return []MetricsGroup{}, db.Error
	}
	return metricsGroups, nil
}

func (main Main) Update(id string, metricsGroup MetricsGroup) (MetricsGroup, error) {
	db := main.db.Table("metrics_groups").Where("id = ?", id).Update(&metricsGroup)
	if db.Error != nil {
		util.Error(util.UpdateMetricsGroupError, "Update", db.Error, metricsGroup)
		return MetricsGroup{}, db.Error
	}
	return metricsGroup, nil
}

func (main Main) Remove(id string) error {
	db := main.db.Where("id = ?", id).Delete(MetricsGroup{})
	if db.Error != nil {
		util.Error(util.RemoveMetricsGroupError, "Remove", db.Error, id)
		return db.Error
	}
	return nil
}

func (main Main) getQueryByMetric(metric Metric) []byte {
	if metric.Query != "" {
		return []byte(metric.Query)
	}

	return []byte(metric.Metric)
}

func (main Main) query(metric Metric, period string) (interface{}, error) {

	dataSourceResult, err := main.datasourceMain.FindById(metric.DataSourceID.String())
	if err != nil {
		notFoundErr := errors.New("Not found data source: " + metric.DataSourceID.String())
		util.Error(util.QueryFindDatasourceError, "Query", notFoundErr, metric.DataSourceID.String())
		return nil, notFoundErr
	}

	plugin, err := main.pluginMain.GetPluginBySrc(dataSourceResult.PluginSrc)
	if err != nil {
		util.Error(util.QueryGetPluginError, "Query", err, dataSourceResult.PluginSrc)
		return nil, err
	}

	getQuery, err := plugin.Lookup("Query")
	if err != nil {
		util.Error(util.PluginLookupError, "Query", err, plugin)
		return nil, err
	}

	query := main.getQueryByMetric(metric)
	dataSourceConfigurationData, _ := json.Marshal(dataSourceResult.Data)
	filters, _ := json.Marshal(metric.Filters)
	return getQuery.(func(datasourceConfiguration, query, filters, period []byte) (interface{}, error))(dataSourceConfigurationData, query, filters, []byte(period))

}

func (main Main) QueryByGroupID(id, period string) ([]datasource.MetricValues, error) {
	var metricsValues []datasource.MetricValues
	metricsGroup, err := main.FindById(id)
	if err != nil {
		notFoundErr := errors.New("Not found metrics group: " + id)
		util.Error(util.FindMetricsGroupError, "QueryByGroupID", notFoundErr, id)
		return nil, notFoundErr
	}

	for _, metric := range metricsGroup.Metrics {

		query, err := main.query(metric, period)
		if err != nil {
			util.Error(util.QueryByGroupIdError, "QueryByGroupID", err, metric)
			return nil, err
		}

		metricsValues = append(metricsValues, datasource.MetricValues{
			ID:       metric.ID,
			Nickname: metric.Nickname,
			Values:   query,
		})
	}

	return metricsValues, nil
}

func (main Main) ResultQuery(metric Metric) (float64, error) {
	dataSourceResult, err := main.datasourceMain.FindById(metric.DataSourceID.String())
	if err != nil {
		notFoundErr := errors.New("Not found data source: " + metric.DataSourceID.String())
		util.Error(util.QueryFindDatasourceError, "ResultQuery", notFoundErr, metric.DataSourceID.String())
		return 0, notFoundErr
	}

	plugin, err := main.pluginMain.GetPluginBySrc(dataSourceResult.PluginSrc)
	if err != nil {
		util.Error(util.QueryGetPluginError, "ResultQuery", err, dataSourceResult.PluginSrc)
		return 0, err
	}

	getQuery, err := plugin.Lookup("Result")
	if err != nil {
		util.Error(util.PluginLookupError, "ResultQuery", err, plugin)
		return 0, err
	}

	dataSourceConfigurationData, _ := json.Marshal(dataSourceResult.Data)
	filters, _ := json.Marshal(metric.Filters)
	query := main.getQueryByMetric(metric)
	return getQuery.(func(datasourceConfiguration, metric, filters []byte) (float64, error))(dataSourceConfigurationData, query, filters)
}

func (main Main) ResultByGroup(group MetricsGroup) ([]datasource.MetricResult, error) {
	var metricsResults []datasource.MetricResult
	for _, metric := range group.Metrics {

		result, err := main.ResultQuery(metric)
		if err != nil {
			util.Error(util.ResultQueryError, "ResultByGroup", err, metric)
			return nil, err
		}

		metricsResults = append(metricsResults, datasource.MetricResult{
			ID:       metric.ID,
			Nickname: metric.Nickname,
			Result:   result,
		})
	}

	return metricsResults, nil
}

func (main Main) ResultByID(id string) ([]datasource.MetricResult, error) {
	metricsGroup, err := main.FindById(id)
	if err != nil {
		notFoundErr := errors.New("Not found metrics group: " + id)
		util.Error(util.FindMetricsGroupError, "ResultByID", notFoundErr, id)
		return nil, notFoundErr
	}

	return main.ResultByGroup(metricsGroup)
}
