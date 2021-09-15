package state

import (
	"database/sql"
	"fmt"
	"github.com/alienrobotwizard/flotilla-os/core/config"
	"github.com/alienrobotwizard/flotilla-os/core/exceptions"
	"github.com/alienrobotwizard/flotilla-os/core/state/models"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"strings"
	"testing"
	"time"
)

func setUp() SQLManager {
	conf, _ := config.NewConfig(nil)
	conf.Set("database_url", "postgres://flotilla:flotilla@127.0.0.1:5432/flotilla")
	sm := SQLManager{}
	err := sm.Initialize(conf)
	if err != nil {
		log.Fatal(err)
	}
	return sm
}

func tearDown() {
	withDB(func(db *sql.DB) {
		db.Exec(`
			drop table if exists templates CASCADE;
			drop table if exists workers CASCADE;
			drop table if exists runs CASCADE;
		`)
	})
}

func withDB(f func(db *sql.DB)) {
	gdb, _ := gorm.Open(postgres.Open("postgres://flotilla:flotilla@127.0.0.1:5432/flotilla"), &gorm.Config{})
	db, err := gdb.DB()
	if err != nil {
		log.Fatal(err)
	}
	f(db)
}

func runSQLManagerTest(test func(sm SQLManager)) {
	sm := setUp()
	defer tearDown()
	test(sm)
}

func TestSQLManager_GetRun(t *testing.T) {
	runSQLManagerTest(func(sm SQLManager) {
		runID, image := "cupcake", "shoelace"

		withDB(func(db *sql.DB) {
			db.Exec("insert into runs (run_id, image) values ($1, $2)", runID, image)
		})

		r, err := sm.GetRun(runID)
		assert.NoError(t, err)
		assert.Equal(t, runID, r.RunID)
		assert.Equal(t, image, r.Image)

		_, err = sm.GetRun("failme")
		assert.Error(t, err)
		assert.Equal(t, true, errors.Is(err, exceptions.ErrRecordNotFound))
	})
}

func TestSQLManager_ListRuns(t *testing.T) {
	runSQLManagerTest(func(sm SQLManager) {
		rows := []struct {
			id      string
			image   string
			status  models.RunStatus
			started time.Time
			env     models.EnvList
		}{
			{id: "A", image: "cupcake", status: models.StatusRunning, started: time.Date(
				2020, 1, 1, 0, 0, 0, 0, time.UTC),
				env: models.EnvList{
					{Name: "E1", Value: "V1"},
				},
			},
			{id: "B", image: "applesauce", status: models.StatusStopped, started: time.Date(
				2020, 1, 2, 0, 0, 0, 0, time.UTC),
				env: models.EnvList{
					{Name: "E2", Value: "V2"},
				},
			},
			{id: "C", image: "ketchup", status: models.StatusQueued, started: time.Date(
				2020, 1, 3, 0, 0, 0, 0, time.UTC),
				env: models.EnvList{
					{Name: "E3", Value: "V3"},
				},
			},
			{id: "D", image: "cupcake", status: models.StatusStopped, started: time.Date(
				2020, 1, 4, 0, 0, 0, 0, time.UTC),
				env: models.EnvList{
					{Name: "E4", Value: "V4"},
				},
			},
		}
		withDB(func(db *sql.DB) {
			sql := "insert into runs (run_id, image, status, started_at, env) values ($1, $2, $3, $4, $5)"
			for _, row := range rows {
				fmt.Println(db.Exec(sql, row.id, row.image, row.status, row.started, row.env))
			}
		})

		// Test ordinary listing
		args := &ListRunsArgs{}
		lr, err := sm.ListRuns(args)
		assert.NoError(t, err)
		assert.Equal(t, int64(len(rows)), lr.Total)

		// Test listing with "like" filter
		args = &ListRunsArgs{
			ListArgs: ListArgs{
				Filters: map[string][]string{
					"image": {"ketch"},
				},
			},
		}

		lr, err = sm.ListRuns(args)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), lr.Total)

		// Test listing with exact match
		args = &ListRunsArgs{
			ListArgs: ListArgs{
				Filters: map[string][]string{
					"status": {string(models.StatusStopped)},
				},
			},
		}

		lr, err = sm.ListRuns(args)
		assert.NoError(t, err)
		assert.Equal(t, int64(2), lr.Total)

		// Test listing with range filter
		args = &ListRunsArgs{
			ListArgs: ListArgs{
				Filters: map[string][]string{
					"started_at_since": {"2020-01-02"},
					"started_at_until": {"2020-01-04"},
				},
			},
		}

		lr, err = sm.ListRuns(args)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), lr.Total)

		// Test listing with env filter
		args = &ListRunsArgs{
			EnvFilters: &EnvFilters{
				"E1": "V1",
			},
		}

		lr, err = sm.ListRuns(args)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), lr.Total)
	})
}

func TestSQLManager_CreateRun(t *testing.T) {
	runSQLManagerTest(func(sm SQLManager) {
		run := models.Run{
			Status: models.StatusQueued,
		}

		created, err := sm.CreateRun(run)
		assert.NoError(t, err)
		assert.Equal(t, true, strings.HasPrefix(created.RunID, models.DefaultEngine))
	})
}

func TestSQLManager_UpdateRun(t *testing.T) {
	runSQLManagerTest(func(sm SQLManager) {
		withDB(func(db *sql.DB) {
			sql := "insert into runs (run_id, status) values ($1, $2)"
			db.Exec(sql, "A", models.StatusQueued)
		})

		updates := models.Run{
			Status: models.StatusNeedsRetry,
		}

		updated, err := sm.UpdateRun("A", updates)
		assert.NoError(t, err)
		assert.Equal(t, models.StatusNeedsRetry, updated.Status)
	})
}

func TestSQLManager_GetTemplate(t *testing.T) {
	runSQLManagerTest(func(sm SQLManager) {
		templateID, templateName, templateVersion := "cupcake", "shoelace", int64(700)

		withDB(func(db *sql.DB) {
			db.Exec("insert into templates (template_id, template_name, version) values ($1, $2, $3)",
				templateID, templateName, templateVersion)
		})

		tmpl, err := sm.GetTemplate(&GetTemplateArgs{TemplateID: &templateID})
		assert.NoError(t, err)
		assert.Equal(t, templateID, tmpl.TemplateID)
		assert.Equal(t, templateName, tmpl.TemplateName)

		tmpl, err = sm.GetTemplate(&GetTemplateArgs{TemplateName: &templateName})
		assert.NoError(t, err)
		assert.Equal(t, templateID, tmpl.TemplateID)
		assert.Equal(t, templateName, tmpl.TemplateName)

		tmpl, err = sm.GetTemplate(&GetTemplateArgs{TemplateName: &templateName, TemplateVersion: &templateVersion})
		assert.NoError(t, err)
		assert.Equal(t, templateID, tmpl.TemplateID)
		assert.Equal(t, templateName, tmpl.TemplateName)
	})
}

func TestSQLManager_ListTemplates(t *testing.T) {
	runSQLManagerTest(func(sm SQLManager) {
		rows := []struct {
			id      string
			name    string
			version int64
		}{
			{id: "A", name: "cupcake", version: int64(1)},
			{id: "B", name: "applesauce", version: int64(2)},
			{id: "C", name: "ketchup", version: int64(2)},
			{id: "D", name: "cupcake", version: int64(2)},
		}
		withDB(func(db *sql.DB) {
			sql := "insert into templates (template_id, template_name, version) values ($1, $2, $3)"
			for _, row := range rows {
				db.Exec(sql, row.id, row.name, row.version)
			}
		})

		args := &ListArgs{}
		lr, err := sm.ListTemplates(args)
		assert.NoError(t, err)
		assert.Equal(t, int64(len(rows)), lr.Total)

		limit := 1
		args.Limit = &limit
		lr, err = sm.ListTemplates(args)
		assert.NoError(t, err)
		assert.Equal(t, int64(len(rows)), lr.Total)
		assert.Equal(t, 1, len(lr.Templates))

		sortBy, order := "template_name", "desc"
		args.SortBy = &sortBy
		args.Order = &order

		lr, err = sm.ListTemplates(args)
		assert.NoError(t, err)
		assert.Equal(t, int64(len(rows)), lr.Total)
		assert.Equal(t, rows[2].name, lr.Templates[0].TemplateName)
	})
}

func TestSQLManager_CreateTemplate(t *testing.T) {
	runSQLManagerTest(func(sm SQLManager) {
		tmpl := models.Template{
			TemplateName: "applesauce",
			Version:      7,
		}

		created, err := sm.CreateTemplate(tmpl)
		assert.NoError(t, err)
		assert.Equal(t, true, strings.HasPrefix(created.TemplateID, "tpl-"))

		tmpl = models.Template{
			TemplateName: tmpl.TemplateName,
			Version:      tmpl.Version,
		}
		_, err = sm.CreateTemplate(tmpl)
		assert.Error(t, err, "should not be able to create template with duplicate (template_name, version)")

		tmpl = models.Template{
			TemplateName: tmpl.TemplateName,
			Version:      8,
		}

		created, err = sm.CreateTemplate(tmpl)
		assert.NoError(t, err,
			"should be able to create template with duplicate template_name but different version")
	})
}

func TestSQLManager_GetWorker(t *testing.T) {
	runSQLManagerTest(func(sm SQLManager) {
		w, err := sm.GetWorker(models.RetryWorker, models.DefaultEngine)
		assert.NoError(t, err)
		assert.Equal(t, string(models.RetryWorker), w.WorkerType)
	})
}

func TestSQLManager_ListWorkers(t *testing.T) {
	runSQLManagerTest(func(sm SQLManager) {
		lr, err := sm.ListWorkers("eks")
		assert.NoError(t, err)
		// Essentially just testing that defaults are working
		assert.Equal(t, int64(3), lr.Total)
	})
}

func TestSQLManager_UpdateWorker(t *testing.T) {
	runSQLManagerTest(func(sm SQLManager) {
		updates := models.Worker{CountPerInstance: 100}
		updated, err := sm.UpdateWorker(models.RetryWorker, updates)
		assert.NoError(t, err)
		assert.Equal(t, updates.CountPerInstance, updated.CountPerInstance)
	})
}

func TestSQLManager_BatchUpdateWorkers(t *testing.T) {
	runSQLManagerTest(func(sm SQLManager) {

	})
}
