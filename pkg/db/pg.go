package db

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/GeertJohan/go.rice"
	"github.com/cenkalti/backoff"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog/log"
	"github.com/rubenv/sql-migrate"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
	"gitlab.com/mswkn/bot"
	"gitlab.com/mswkn/bot/pkg/config"
	"gitlab.com/mswkn/bot/pkg/db/models"
	"strings"
	"time"
)

func NewPgDb(conf config.Config) *sql.DB {
	lg := log.With().Str("comp", "db_pg").Logger()

	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = time.Second * 60

	var db *sql.DB
	conn := func() error {
		conStr := fmt.Sprintf(
			"dbname=%s host=%s user=%s password=%s sslmode=disable",
			conf.Database.Pg.Database,
			conf.Database.Pg.Host,
			conf.Database.Pg.Username,
			conf.Database.Pg.Password,
		)
		sqlDB, err := sql.Open("postgres", conStr)
		if err != nil {
			lg.Error().Err(err).Msg("could no create connection for database")
			return err
		}

		if err := sqlDB.Ping(); err != nil {
			lg.Error().Err(err).Msg("could no connect to database")
			return err
		}

		db = sqlDB
		return nil
	}
	if err := backoff.Retry(conn, bo); err != nil || db == nil {
		lg.Fatal().Err(err).Msg("could not connect to database")
	}

	box, err := rice.FindBox("../../sql/migrations")
	if err != nil {
		lg.Fatal().Err(err).Msg("could not load migrations")
	}
	migrations := &migrate.HttpFileSystemMigrationSource{
		FileSystem: box.HTTPBox(),
	}
	n, err := migrate.Exec(db, "postgres", migrations, migrate.Up)
	if err != nil {
		lg.Fatal().Err(err).Msg("could not apply migrations")
	}
	lg.Info().Int("count", n).Msg("applied migrations")

	return db
}

//ToDo implement periodic clean up for values not updated in the last three months
type PgInfoLinkRepository struct {
	db *sql.DB
}

func NewPgInfoLinkRepository(db *sql.DB) mswkn.InfoLinkRepository {
	p := &PgInfoLinkRepository{
		db: db,
	}
	return p
}

func (p *PgInfoLinkRepository) Add(ctx context.Context, il *mswkn.InfoLink) error {
	now := time.Now()
	mil := models.InfoLink{
		WKN:       strings.ToUpper(il.WKN),
		URL:       il.URL,
		CreatedAt: now,
		UpdatedAt: now,
	}
	return mil.Upsert(
		ctx,
		p.db,
		true,
		[]string{models.InfoLinkColumns.WKN},
		boil.Whitelist(models.InfoLinkColumns.URL, models.InfoLinkColumns.UpdatedAt),
		boil.Infer(),
	)
}

func (p *PgInfoLinkRepository) Get(ctx context.Context, wkn string) (*mswkn.InfoLink, error) {
	i, err := models.InfoLinks(qm.Where(models.SecurityColumns.WKN+"=?", strings.ToUpper(wkn))).One(ctx, p.db)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, mswkn.ErrInfoLinkNotFound
		}
		return nil, err
	}

	return &mswkn.InfoLink{
		WKN: i.WKN,
		URL: i.URL,
	}, nil
}

type PgSecurityRepository struct {
	db *sql.DB
}

func NewPgSecurityRepository(db *sql.DB) mswkn.SecurityRepository {
	p := &PgSecurityRepository{
		db: db,
	}
	return p
}

func (p *PgSecurityRepository) Add(ctx context.Context, sec *mswkn.Security) error {
	s := toDbSec(sec)
	err := s.Upsert(ctx, p.db, true, []string{"isin"}, boil.Whitelist("name"), boil.Infer())
	return err
}

func (p *PgSecurityRepository) AddBulk(ctx context.Context, secs []*mswkn.Security) error {
	now := time.Now()

	stm := strings.Builder{}
	stm.WriteString(fmt.Sprintf(
		"INSERT INTO %s (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)",
		models.TableNames.Securities,
		models.SecurityColumns.ID,
		models.SecurityColumns.Name,
		models.SecurityColumns.Isin,
		models.SecurityColumns.WKN,
		models.SecurityColumns.Underlying,
		models.SecurityColumns.Type,
		models.SecurityColumns.WarrantType,
		models.SecurityColumns.WarrantSubType,
		models.SecurityColumns.Strike,
		models.SecurityColumns.Expire,
		models.SecurityColumns.UpdatedAt,
		models.SecurityColumns.CreatedAt,
	))

	stm.WriteString(" VALUES ")

	values := make([]interface{}, 0, len(secs))
	placeholder := make([]string, 0, len(secs))
	i := 1
	for _, sec := range secs {
		valStr := fmt.Sprintf("(DEFAULT, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)", i, i+1, i+2, i+3, i+4, i+5, i+6, i+7, i+8, i+9, i+10)
		placeholder = append(placeholder, valStr)
		i += 11

		values = append(values, sec.Name)
		values = append(values, sec.ISIN)
		values = append(values, sec.WKN)
		values = append(values, sec.Underlying)
		values = append(values, sec.Type)
		values = append(values, sec.WarrantType)
		values = append(values, sec.WarrantSubType)
		values = append(values, sec.Strike)
		values = append(values, sec.Expire)
		values = append(values, now)
		values = append(values, now)
	}

	stm.WriteString(strings.Join(placeholder, ","))

	//ToDo check if created_at is not updated on conflict

	stm.WriteString(fmt.Sprintf(`
		ON CONFLICT (%s)
			DO UPDATE
				SET "%s" = EXCLUDED."%s",
					"%s" = EXCLUDED."%s",
					"%s" = EXCLUDED."%s",
					"%s" = EXCLUDED."%s",
					"%s" = EXCLUDED."%s",
					"%s" = EXCLUDED."%s",
					"%s" = EXCLUDED."%s",
					"%s" = EXCLUDED."%s",
					"%s" = EXCLUDED."%s",
					"%s" = EXCLUDED."%s",
					"%s" = EXCLUDED."%s"
	RETURNING "id"
	`,
		models.SecurityColumns.Isin,
		models.SecurityColumns.Name,
		models.SecurityColumns.Name,
		models.SecurityColumns.Isin,
		models.SecurityColumns.Isin,
		models.SecurityColumns.WKN,
		models.SecurityColumns.WKN,
		models.SecurityColumns.Underlying,
		models.SecurityColumns.Underlying,
		models.SecurityColumns.Type,
		models.SecurityColumns.Type,
		models.SecurityColumns.WarrantType,
		models.SecurityColumns.WarrantType,
		models.SecurityColumns.WarrantSubType,
		models.SecurityColumns.WarrantSubType,
		models.SecurityColumns.Strike,
		models.SecurityColumns.Strike,
		models.SecurityColumns.Expire,
		models.SecurityColumns.Expire,
		models.SecurityColumns.UpdatedAt,
		models.SecurityColumns.UpdatedAt,
		models.SecurityColumns.CreatedAt,
		models.SecurityColumns.CreatedAt,
	))

	sqlStr := stm.String()
	_, err := p.db.ExecContext(ctx, sqlStr, values...)
	return err
}

func (p *PgSecurityRepository) Get(ctx context.Context, wkn string) (*mswkn.Security, error) {
	s, err := models.Securities(qm.Where("wkn=?", strings.ToUpper(wkn))).One(ctx, p.db)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, mswkn.ErrSecurityNotFound
		}
		return nil, err
	}
	return fromDbSec(s), nil
}

func toDbSec(sec *mswkn.Security) *models.Security {
	s := &models.Security{
		ID:             0,
		Name:           sec.Name,
		Isin:           sec.ISIN,
		WKN:            sec.WKN,
		Underlying:     sec.Underlying,
		Type:           sec.Type,
		WarrantType:    sec.WarrantType,
		WarrantSubType: sec.WarrantSubType,
		Strike:         sec.Strike,
		//Expire:         nil,
	}

	if sec.Expire != nil {
		s.Expire = null.NewTime(*sec.Expire, true)
	}

	return s
}

func fromDbSec(sec *models.Security) *mswkn.Security {
	s := &mswkn.Security{
		Name:           sec.Name,
		ISIN:           sec.Isin,
		WKN:            sec.WKN,
		Underlying:     sec.Underlying,
		Type:           sec.Type,
		WarrantType:    sec.WarrantType,
		WarrantSubType: sec.WarrantSubType,
		Strike:         sec.Strike,
		Expire:         &sec.Expire.Time,
	}

	if !sec.Expire.Valid {
		s.Expire = nil
	}

	return s
}
