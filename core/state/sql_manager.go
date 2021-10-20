package state

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/alienrobotwizard/flotilla-os/core/config"
	"github.com/alienrobotwizard/flotilla-os/core/exceptions"
	"github.com/alienrobotwizard/flotilla-os/core/state/models"
	"github.com/pkg/errors"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"strings"
)

type SQLManager struct {
	db *gorm.DB
}

var likeFields = map[string]bool{
	"image":       true,
	"alias":       true,
	"group_name":  true,
	"command":     true,
	"text":        true,
	"exit_reason": true,
}

func NewSQLManager(ctx context.Context, c *config.Config) (Manager, error) {
	dbURL := c.GetString("database_url")
	db, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{})
	if err != nil {
		return nil, errors.Wrap(err, "Unable to open database")
	}

	if err := db.WithContext(ctx).AutoMigrate(&models.Run{}, &models.Template{}, &models.Worker{}); err != nil {
		return nil, errors.Wrap(err, "Unable to auto-migrate database")
	}

	if err := initWorkerTable(ctx, c, db); err != nil {
		return nil, err
	}

	go func(ctx context.Context, db *gorm.DB) {
		select {
		case <-ctx.Done():
			if sqlDB, err := db.DB(); err == nil {
				sqlDB.Close()
			}
		}
	}(ctx, db)

	return &SQLManager{
		db: db,
	}, nil
}

func (m *SQLManager) GetTemplate(ctx context.Context, args *GetTemplateArgs) (models.Template, error) {
	var (
		t    models.Template
		vals []interface{}
	)

	if args.TemplateID != nil {
		vals = append(vals, "template_id = ?", *args.TemplateID)
	} else if args.TemplateName != nil {
		qual := "template_name = ?"
		if args.TemplateVersion != nil {
			qual += " AND version = ?"
			vals = append(vals, qual, *args.TemplateName, *args.TemplateVersion)
		} else {
			vals = append(vals, qual, *args.TemplateName)
		}
	}

	if result := m.db.WithContext(ctx).First(&t, vals...); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return t, exceptions.ErrRecordNotFound
		} else {
			return t, errors.Wrap(result.Error, "issue getting template with")
		}
	}
	return t, nil
}

func (m *SQLManager) ListTemplates(ctx context.Context, args *ListArgs) (models.TemplateList, error) {
	var lr models.TemplateList

	q := m.db.WithContext(ctx).Model(&models.Template{})
	q.Count(&lr.Total)

	q = q.Limit(args.GetLimit()).Offset(args.GetOffset())
	if args.SortBy != nil {
		q = q.Order(fmt.Sprintf("%s %s", *args.SortBy, args.GetOrder()))
	}
	if q = q.Find(&lr.Templates); q.Error != nil {
		return lr, errors.Wrap(q.Error, "problem listing templates")
	} else {
		return lr, nil
	}
}

func (m *SQLManager) CreateTemplate(ctx context.Context, t models.Template) (models.Template, error) {
	if result := m.db.WithContext(ctx).Create(&t); result.Error != nil {
		return t, result.Error
	} else {
		return t, nil
	}
}

func (m *SQLManager) GetRun(ctx context.Context, runID string) (models.Run, error) {
	var r models.Run
	if result := m.db.WithContext(ctx).First(&r, "run_id = ?", runID); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return r, exceptions.ErrRecordNotFound
		} else {
			return r, errors.Wrapf(result.Error, "issue getting run with runID: [%s]", runID)
		}
	}
	return r, nil
}

func (m *SQLManager) CreateRun(ctx context.Context, r models.Run) (models.Run, error) {
	if result := m.db.WithContext(ctx).Create(&r); result.Error != nil {
		return r, result.Error
	} else {
		return r, nil
	}
}

func (m *SQLManager) UpdateRun(ctx context.Context, runID string, updates models.Run) (models.Run, error) {
	var r models.Run
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		updates.RunID = runID
		tx = tx.Clauses(clause.Locking{Strength: "UPDATE"})
		tx = tx.First(&r, "run_id = ?", runID)
		if res := tx.Updates(updates); res.Error != nil {
			return res.Error
		}
		return nil
	})

	return r, err
}

func (m *SQLManager) ListRuns(ctx context.Context, args *ListRunsArgs) (models.RunList, error) {
	var lr models.RunList

	q := m.db.WithContext(ctx).Model(&models.Run{})
	if args.Filters != nil {
		q = m.applyFilters(q, args.Filters)
	}

	if args.EnvFilters != nil {
		q = m.applyEnvFilters(q, *args.EnvFilters)
	}

	if args.Engines != nil {
		q = q.Where("engine in ?", *args.Engines)
	}

	q.Count(&lr.Total)

	q = q.Limit(args.GetLimit()).Offset(args.GetOffset())
	if args.SortBy != nil {
		q = q.Order(fmt.Sprintf("%s %s", *args.SortBy, args.GetOrder()))
	}
	if q = q.Find(&lr.Runs); q.Error != nil {
		return lr, errors.Wrap(q.Error, "problem listing runs")
	} else {
		return lr, nil
	}
}

func (m *SQLManager) applyFilters(q *gorm.DB, filters map[string][]string) *gorm.DB {
	for k, v := range filters {
		if len(v) > 1 {
			q = q.Where(fmt.Sprintf("%s in ?", k), v)
		} else if len(v) == 1 {
			if likeFields[k] {
				q = q.Where(fmt.Sprintf("%s like ?", k), fmt.Sprintf("%%%s%%", v[0]))
			} else if strings.HasSuffix(k, "_since") {
				field := strings.Replace(k, "_since", "", -1)
				q = q.Where(fmt.Sprintf("%s > ?", field), v[0])
			} else if strings.HasSuffix(k, "_until") {
				field := strings.Replace(k, "_until", "", -1)
				q = q.Where(fmt.Sprintf("%s < ?", field), v[0])
			} else {
				q = q.Where(map[string]string{k: v[0]})
			}
		}
	}
	return q
}

func (m *SQLManager) applyEnvFilters(q *gorm.DB, filters map[string]string) *gorm.DB {
	for k, v := range filters {
		q = q.Where(fmt.Sprintf(`env @> '[{"name":"%s","value":"%s"}]'`, k, v))
	}
	return q
}

func (m *SQLManager) ListWorkers(ctx context.Context, engine string) (models.WorkersList, error) {
	var lr models.WorkersList

	q := m.db.WithContext(ctx).Model(&models.Worker{}).Where("engine = ?", engine)
	q.Count(&lr.Total)

	if q = q.Find(&lr.Workers); q.Error != nil {
		return lr, errors.Wrap(q.Error, "problem listing workers")
	} else {
		return lr, nil
	}
}

func (m *SQLManager) BatchUpdateWorkers(ctx context.Context, updates []models.Worker) (models.WorkersList, error) {
	var wl models.WorkersList
	for _, w := range updates {
		if _, err := m.UpdateWorker(ctx, models.WorkerType(w.WorkerType), w); err != nil {
			return wl, err
		}
	}
	return m.ListWorkers(ctx, models.DefaultEngine)
}

func (m *SQLManager) GetWorker(ctx context.Context, workerType models.WorkerType, engine string) (models.Worker, error) {
	var w models.Worker
	if result := m.db.WithContext(ctx).First(&w, "worker_type = ? AND engine = ?", workerType, engine); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return w, exceptions.ErrRecordNotFound
		} else {
			return w, errors.Wrapf(result.Error,
				"issue getting worker with type: [%s] and engine: [%s]", workerType, engine)
		}
	}
	return w, nil
}

func (m *SQLManager) UpdateWorker(ctx context.Context, workerType models.WorkerType, updates models.Worker) (models.Worker, error) {
	var (
		err      error
		existing models.Worker
	)

	engine := models.DefaultEngine
	if len(updates.Engine) > 0 {
		engine = updates.Engine
	}

	err = m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		tx = tx.Clauses(clause.Locking{Strength: "UPDATE"})
		tx = tx.First(&existing, "worker_type = ? AND engine = ?", workerType, engine)
		if res := tx.Updates(updates); res.Error != nil {
			return res.Error
		}
		return nil
	})
	return existing, err
}

func initWorkerTable(ctx context.Context, c *config.Config, db *gorm.DB) error {
	// Get worker count from configuration (set to 1 as default)

	for _, engine := range models.Engines {
		retryCount := int64(1)
		retryCountKey := fmt.Sprintf("worker.%s.retry_worker_count_per_instance", engine)
		if c.IsSet(retryCountKey) {
			retryCount = int64(c.GetInt(retryCountKey))
		}
		submitCount := int64(1)
		submitCountKey := fmt.Sprintf("worker.%s.submit_worker_count_per_instance", engine)
		if c.IsSet(submitCountKey) {
			submitCount = int64(c.GetInt(submitCountKey))
		}
		statusCount := int64(1)
		statusCountKey := fmt.Sprintf("worker.%s.status_worker_count_per_instance", engine)
		if c.IsSet(submitCountKey) {
			statusCount = int64(c.GetInt(statusCountKey))
		}

		var err error
		insert := `
		INSERT INTO workers (worker_type, count_per_instance, engine)
		VALUES ('retry', ?, @engine), ('submit', ?, @engine), ('status', ?, @engine) ON CONFLICT DO NOTHING;
	`
		err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			return tx.Exec(insert, retryCount, submitCount, statusCount, sql.Named("engine", engine)).Error
		})

		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}
